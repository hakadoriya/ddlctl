package spanner

import (
	"io"

	apperr "github.com/hakadoriya/ddlctl/pkg/apperr"
	ddlast "github.com/hakadoriya/ddlctl/pkg/internal/generator"
	"github.com/hakadoriya/ddlctl/pkg/logs"
)

const (
	Dialect       = "spanner"
	CommentPrefix = "--"
	Quotation     = "`"
)

func Fprint(w io.Writer, ddl *ddlast.DDL) error {
	var buf string

	for _, header := range ddl.Header {
		fprintComment(&buf, "", header)
	}

	for _, statement := range ddl.Stmts {
		buf += "\n"
		switch stmt := statement.(type) {
		case *ddlast.CreateTableStmt:
			fprintCreateTable(&buf, ddl.Indent, stmt)
		case *ddlast.CreateIndexStmt:
			fprintCreateIndex(&buf, ddl.Indent, stmt)
		default:
			logs.Warn.Printf("unknown statement type: %T: %v", stmt, apperr.ErrNotSupported)
			continue
		}
	}

	if _, err := io.WriteString(w, buf); err != nil {
		return apperr.Errorf("io.WriteString: %w", err)
	}
	return nil
}

func fprintComment(buf *string, indent string, comment string) {
	if comment == "" {
		*buf += indent + CommentPrefix + "\n"
		return
	}

	*buf += indent + CommentPrefix + " " + comment + "\n"
	return //nolint:gosimple
}
