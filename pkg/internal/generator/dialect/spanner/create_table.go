package spanner

import (
	"fmt"
	"strconv"

	"github.com/hakadoriya/z.go/pathz/filepathz"
	"github.com/hakadoriya/z.go/slicez"

	ddlast "github.com/hakadoriya/ddlctl/pkg/internal/generator"
)

//nolint:cyclop,funlen,gocognit
func fprintCreateTable(buf *string, indent string, stmt *ddlast.CreateTableStmt) {
	// source
	if stmt.SourceFile != "" {
		fprintComment(buf, "", fmt.Sprintf("source: %s:%d", filepathz.ExtractShortPath(stmt.SourceFile), stmt.SourceLine))
	}

	// comments
	for _, comment := range stmt.Comments {
		fprintComment(buf, "", comment)
	}

	if stmt.CreateTable != "" { //nolint:nestif
		// CREATE TABLE and Left Parenthesis
		*buf += stmt.CreateTable + " (\n"

		hasTableConstraint := len(stmt.Constraints) > 0

		// COLUMNS
		fprintCreateTableColumn(buf, indent, stmt.Columns, hasTableConstraint)

		// CONSTRAINT
		for i, constraint := range stmt.Constraints {
			fprintCreateTableConstraint(buf, indent, constraint)
			if lastConstraintIndex := len(stmt.Constraints) - 1; i != lastConstraintIndex {
				*buf += ","
			}
			*buf += "\n"
		}

		// Right Parenthesis
		*buf += ")"

		// PRIMARY KEY
		if len(stmt.PrimaryKey) > 0 {
			*buf += " PRIMARY KEY ("
			for i, primaryKey := range stmt.PrimaryKey {
				*buf += Quotation + primaryKey + Quotation
				if lastPrimaryKeyIndex := len(stmt.PrimaryKey) - 1; i != lastPrimaryKeyIndex {
					*buf += ", "
				}
			}
			*buf += ")"
			if hasTableOptions := len(stmt.Options) > 0; hasTableOptions {
				*buf += ","
			}
		}

		// OPTIONS
		for i, option := range stmt.Options {
			*buf += "\n"
			fprintCreateTableOption(buf, "", option)
			if lastOptionIndex := len(stmt.Options) - 1; i != lastOptionIndex {
				*buf += ","
			}
		}

		*buf += ";\n"
	}

	return //nolint:gosimple
}

func fprintCreateTableColumn(buf *string, indent string, columns []*ddlast.CreateTableColumn, tailComma bool) {
	columnNameMaxLength := 0
	slicez.ForEach(columns, func(_ int, elem *ddlast.CreateTableColumn) {
		if columnLength := len(elem.ColumnName); columnLength > columnNameMaxLength {
			columnNameMaxLength = columnLength
		}
	})
	const quotationCharsLength = 2
	columnNameFormat := "%-" + strconv.Itoa(quotationCharsLength+columnNameMaxLength) + "s"

	for i, column := range columns {
		for _, comment := range column.Comments {
			fprintComment(buf, indent, comment)
		}

		*buf += indent + fmt.Sprintf(columnNameFormat, Quotation+column.ColumnName+Quotation) + " " + column.TypeConstraint

		if lastColumn := len(columns) - 1; i == lastColumn && !tailComma {
			*buf += "\n"
		} else {
			*buf += ",\n"
		}
	}

	return //nolint:gosimple
}

func fprintCreateTableConstraint(buf *string, indent string, constraint *ddlast.CreateTableConstraint) {
	for _, comment := range constraint.Comments {
		fprintComment(buf, indent, comment)
	}

	*buf += indent + constraint.Constraint

	return //nolint:gosimple
}

func fprintCreateTableOption(buf *string, indent string, option *ddlast.CreateTableOption) {
	for _, comment := range option.Comments {
		fprintComment(buf, indent, comment)
	}

	*buf += indent + option.Option

	return //nolint:gosimple
}
