package ddlctlgo

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"sync"

	"github.com/hakadoriya/z.go/pathz/filepathz"

	"github.com/hakadoriya/ddlctl/pkg/apperr"
	"github.com/hakadoriya/ddlctl/pkg/internal/config"
	"github.com/hakadoriya/ddlctl/pkg/logs"
)

type ddlSource struct {
	Position token.Position
	// TypeSpec is used to guess the table name if the CREATE TABLE annotation is not found.
	TypeSpec *ast.TypeSpec
	// StructType is used to determine the column name. If the tag specified by --go-column-tag is not found, the field name is used.
	StructType   *ast.StructType
	CommentGroup *ast.CommentGroup
}

//nolint:gochecknoglobals
var (
	_DDLTagGoCommentLineRegex     *regexp.Regexp
	_DDLTagGoCommentLineRegexOnce sync.Once
)

const (
	//	                                          ________________ <- 1. comment prefix
	//	                                                          __ <- 2. tag name
	//	                                                                            ___ <- 4. comment suffix
	_DDLTagGoCommentLineRegexFormat       = `^\s*(//+\s*|/\*\s*)?(%s)\s*:\s*(.*)\s*(\*/)?`
	_DDLTagGoCommentLineRegexContentIndex = /*                               ^^ 3. tag value */ 3
)

func DDLTagGoCommentLineRegex() *regexp.Regexp {
	_DDLTagGoCommentLineRegexOnce.Do(func() {
		_DDLTagGoCommentLineRegex = regexp.MustCompile(fmt.Sprintf(_DDLTagGoCommentLineRegexFormat, config.DDLTagGo()))
	})
	return _DDLTagGoCommentLineRegex
}

//
//nolint:cyclop
func extractDDLSourceFromDDLTagGo(_ context.Context, fset *token.FileSet, f *ast.File) ([]*ddlSource, error) {
	ddlSrc := make([]*ddlSource, 0)

	for commentedNode, commentGroups := range ast.NewCommentMap(fset, f, f.Comments) {
		for _, commentGroup := range commentGroups {
		CommentGroupLoop:
			for _, commentLine := range commentGroup.List {
				logs.Trace.Printf("commentLine=%s: %s", filepathz.ExtractShortPath(fset.Position(commentGroup.Pos()).String()), commentLine.Text)
				// NOTE: If the comment line matches the DDLTagGo, it is assumed to be a comment line for the struct.
				if matches := DDLTagGoCommentLineRegex().FindStringSubmatch(commentLine.Text); len(matches) > _DDLTagGoCommentLineRegexContentIndex {
					s := &ddlSource{
						Position:     fset.Position(commentLine.Pos()),
						CommentGroup: commentGroup,
					}
					ast.Inspect(commentedNode, func(node ast.Node) bool {
						switch n := node.(type) {
						case *ast.TypeSpec:
							s.TypeSpec = n
							switch t := n.Type.(type) {
							case *ast.StructType:
								s.StructType = t
								return false
							default: // noop
							}
						default: // noop
						}
						return true
					})
					ddlSrc = append(ddlSrc, s)
					break CommentGroupLoop // NOTE: There may be multiple "DDLTagGo"s in the same commentGroup, so once you find the first one, break.
				}
			}
		}
	}

	if len(ddlSrc) == 0 {
		return nil, apperr.Errorf("go-ddl-tag=%s: %w", config.DDLTagGo(), apperr.ErrDDLTagGoAnnotationNotFoundInSource)
	}

	return ddlSrc, nil
}
