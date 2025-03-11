package ddlctlgo

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/hakadoriya/z.go/pathz/filepathz"
	"github.com/hakadoriya/z.go/slicez"

	"github.com/hakadoriya/ddlctl/pkg/apperr"
	"github.com/hakadoriya/ddlctl/pkg/internal/config"
	"github.com/hakadoriya/ddlctl/pkg/internal/generator"
	langutil "github.com/hakadoriya/ddlctl/pkg/internal/lang/util"
	"github.com/hakadoriya/ddlctl/pkg/internal/util"
	"github.com/hakadoriya/ddlctl/pkg/logs"
)

const (
	Language                                = "go"
	DDLCTL_ERROR_STRUCT_FIELD_TAG_NOT_FOUND = "DDLCTL_ERROR_STRUCT_FIELD_TAG_NOT_FOUND" //nolint:revive,stylecheck
)

//nolint:cyclop
func Parse(ctx context.Context, src string) (*generator.DDL, error) {
	// MEMO: get absolute path for parser.ParseFile()
	sourceAbs := util.Abs(src)

	info, err := os.Stat(sourceAbs)
	if err != nil {
		return nil, apperr.Errorf("os.Stat: %w", err)
	}

	ddl := generator.NewDDL(ctx)

	if info.IsDir() {
		if err := filepath.WalkDir(sourceAbs, walkDirFn(ctx, ddl)); err != nil {
			return nil, apperr.Errorf("filepath.WalkDir: %w", err)
		}

		return ddl, nil
	}

	stmts, err := parseFile(ctx, sourceAbs)
	if err != nil {
		return nil, apperr.Errorf("Parse: %w", err)
	}
	ddl.Stmts = append(ddl.Stmts, stmts...)

	return ddl, nil
}

//nolint:gochecknoglobals
var fileSuffix = ".go"

func walkDirFn(ctx context.Context, ddl *generator.DDL) func(path string, d os.DirEntry, err error) error {
	return func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err //nolint:wrapcheck
		}

		if d.IsDir() || !strings.HasSuffix(path, fileSuffix) || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		stmts, err := parseFile(ctx, path)
		if err != nil {
			if errors.Is(err, apperr.ErrDDLTagGoAnnotationNotFoundInSource) {
				logs.Debug.Printf("parseFile: %s: %v", path, err)
				return nil
			}
			return apperr.Errorf("parseFile: %w", err)
		}

		ddl.Stmts = append(ddl.Stmts, stmts...)

		return nil
	}
}

//nolint:cyclop,funlen,gocognit
func parseFile(ctx context.Context, filename string) ([]generator.Stmt, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, apperr.Errorf("parser.ParseFile: %w", err)
	}

	ddlSrc, err := extractDDLSourceFromDDLTagGo(ctx, fset, f)
	if err != nil {
		return nil, apperr.Errorf("extractDDLSourceFromDDLTagGo: %w", err)
	}

	dumpDDLSource(fset, ddlSrc)

	stmts := make([]generator.Stmt, 0)
	for _, r := range ddlSrc {
		createTableStmt := &generator.CreateTableStmt{}

		// source
		createTableStmt.SourceFile = r.Position.Filename
		createTableStmt.SourceLine = r.Position.Line

		// CREATE TABLE (or INDEX) / CONSTRAINT / OPTIONS (from comments)
		comments := slicez.Map(r.CommentGroup.List, func(_ int, comment *ast.Comment) string {
			return strings.TrimLeftFunc(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(comment.Text, "//"), "/*"), "*/"), unicode.IsSpace)
		})
		for _, comment := range comments {
			logs.Debug.Printf("[COMMENT DETECTED]: %s:%d: %s", createTableStmt.SourceFile, createTableStmt.SourceLine, comment)

			// NOTE: CREATE INDEX may be written in CREATE TABLE annotation, so process it here
			if /* CREATE INDEX */ matches := langutil.StmtRegexCreateIndex.Regex.FindStringSubmatch(comment); len(matches) > langutil.StmtRegexCreateIndex.Index {
				commentMatchedCreateIndex := comment
				source := fset.Position(extractContainingCommentFromCommentGroup(r.CommentGroup, commentMatchedCreateIndex).Pos())
				createIndexStmt := &generator.CreateIndexStmt{
					Comments:   []string{commentMatchedCreateIndex},
					SourceFile: source.Filename,
					SourceLine: source.Line,
				}
				createIndexStmt.SetCreateIndex(matches[langutil.StmtRegexCreateIndex.Index])
				stmts = append(stmts, createIndexStmt)
				continue
			}

			if /* CREATE TABLE */ matches := langutil.StmtRegexCreateTable.Regex.FindStringSubmatch(comment); len(matches) > langutil.StmtRegexCreateTable.Index {
				createTableStmt.SetCreateTable(matches[langutil.StmtRegexCreateTable.Index])
			} else if /* CONSTRAINT */ matches := langutil.StmtRegexCreateTableConstraint.Regex.FindStringSubmatch(comment); len(matches) > langutil.StmtRegexCreateTableConstraint.Index {
				createTableStmt.Constraints = append(createTableStmt.Constraints, &generator.CreateTableConstraint{
					Constraint: matches[langutil.StmtRegexCreateTableConstraint.Index],
				})
			} else if /* OPTIONS */ matches := langutil.StmtRegexCreateTableOptions.Regex.FindStringSubmatch(comment); len(matches) > langutil.StmtRegexCreateTableOptions.Index {
				createTableStmt.Options = append(createTableStmt.Options, &generator.CreateTableOption{
					Option: matches[langutil.StmtRegexCreateTableOptions.Index],
				})
			}
			// comment
			createTableStmt.Comments = append(createTableStmt.Comments, comment)
		}

		// CREATE TABLE (default: struct name)
		if r.TypeSpec != nil && createTableStmt.CreateTable == "" {
			name := r.TypeSpec.Name.String()
			source := fset.Position(r.CommentGroup.Pos())
			createTableStmt.Comments = append(createTableStmt.Comments, fmt.Sprintf("WARN: the comment (%s:%d) does not have a key for table (%s: table: CREATE TABLE <table>), so the struct name \"%s\" is used as the table name.", filepathz.ExtractShortPath(source.Filename), source.Line, config.DDLTagGo(), name))
			createTableStmt.SetCreateTable(name)
		}

		// columns
		if r.StructType != nil {
			for _, field := range r.StructType.Fields.List {
				// skip not exported fields
				if field.Names == nil || !field.Names[0].IsExported() {
					continue
				}

				column := &generator.CreateTableColumn{}

				tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))

				// column name
				switch columnName := tag.Get(config.ColumnTagGo()); columnName {
				case "-":
					createTableStmt.Comments = append(createTableStmt.Comments, fmt.Sprintf("NOTE: the \"%s\" struct's \"%s\" field has a tag for column name (`%s:\"-\"`), so the field is ignored.", r.TypeSpec.Name, field.Names[0], config.ColumnTagGo()))
					continue
				case "":
					name := field.Names[0].Name
					column.Comments = append(column.Comments, fmt.Sprintf("WARN: the \"%s\" struct's \"%s\" field does not have a tag for column name (`%s:\"<ColumnName>\"`), so the field name \"%s\" is used as the column name.", r.TypeSpec.Name, field.Names[0], config.ColumnTagGo(), name))
					column.ColumnName = name
				default:
					column.ColumnName = columnName
				}

				// column type and constraint
				switch columnTypeConstraint := tag.Get(config.DDLTagGo()); columnTypeConstraint {
				case "", "-":
					// NOTE: ignore no-annotation fields
					// column.Comments = append(column.Comments, fmt.Sprintf("ERROR: the \"%s\" struct's \"%s\" field does not have a tag for column type and constraint (`%s:\"<TYPE> [CONSTRAINT]\"`)", r.TypeSpec.Name, field.Names[0], config.DDLTagGo()))
					// column.TypeConstraint = DDLCTL_ERROR_STRUCT_FIELD_TAG_NOT_FOUND
					continue
				default:
					column.TypeConstraint = columnTypeConstraint
				}

				// primary key
				switch primaryKey := tag.Get(config.PKTagGo()); primaryKey {
				case "true", "1":
					createTableStmt.PrimaryKey = append(createTableStmt.PrimaryKey, column.ColumnName)
				case "", "-":
					// do nothing
				default:
					column.Comments = append(column.Comments, fmt.Sprintf("WARN: the field \"%s\" does not have valid primary key tag (`%s:\"true\"`), so the column is not used as primary key.", field.Names[0], config.PKTagGo()))
				}

				// comments
				comments := strings.Split(strings.Trim(field.Doc.Text(), "\n"), "\n")
				column.Comments = append(column.Comments, langutil.TrimCommentElementTailEmpty(langutil.TrimCommentElementHasPrefix(comments, config.DDLTagGo()))...)

				createTableStmt.Columns = append(createTableStmt.Columns, column)
			}
		}

		if createTableStmt.CreateTable == "" {
			// CREATE TABLE (ERROR)
			source := fset.Position(r.CommentGroup.Pos())
			createTableStmt.Comments = append(createTableStmt.Comments, fmt.Sprintf("WARN: the comment (%s:%d) does not have a key for table (%s: table: CREATE TABLE <table>), or the comment is not associated with struct.", filepathz.ExtractShortPath(source.Filename), source.Line, config.DDLTagGo()))
		} else if len(createTableStmt.Columns) == 0 {
			// columns (ERROR)
			source := fset.Position(r.CommentGroup.Pos())
			createTableStmt.Comments = append(createTableStmt.Comments, fmt.Sprintf("ERROR: the comment (%s:%d) does not have struct fields for column type and constraint (`%s:\"<TYPE> [CONSTRAINT]\"`), or the comment is not associated with struct.", filepathz.ExtractShortPath(source.Filename), source.Line, config.DDLTagGo()))
		}

		if len(createTableStmt.Columns) > 0 {
			// NOTE: append only if there are columns
			stmts = append(stmts, createTableStmt)
		} else {
			logs.Warn.Printf("parseFile: %s:%d: %s", createTableStmt.SourceFile, createTableStmt.SourceLine, "no columns")
		}
	}

	sort.Slice(stmts, func(i, j int) bool {
		return fmt.Sprintf("%s:%09d", stmts[i].GetSourceFile(), stmts[i].GetSourceLine()) < fmt.Sprintf("%s:%09d", stmts[j].GetSourceFile(), stmts[j].GetSourceLine())
	})

	for i := range stmts {
		logs.Trace.Print(fmt.Sprintf("%s:%09d", stmts[i].GetSourceFile(), stmts[i].GetSourceLine()))
	}

	return stmts, nil
}

func extractContainingCommentFromCommentGroup(commentGroup *ast.CommentGroup, sub string) *ast.Comment {
	for _, commentLine := range commentGroup.List {
		if strings.Contains(commentLine.Text, sub) {
			return commentLine
		}
	}
	return nil
}
