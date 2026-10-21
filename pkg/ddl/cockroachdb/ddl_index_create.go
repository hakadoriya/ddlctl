package cockroachdb

import (
	"strings"

	"github.com/hakadoriya/z.go/stringz"

	"github.com/hakadoriya/ddlctl/pkg/ddl/internal"
)

// MEMO: https://www.cockroachlabs.com/docs/stable/create-index

var _ Stmt = (*CreateIndexStmt)(nil)

type CreateIndexStmt struct {
	Comment          string
	Unique           bool
	IfNotExists      bool
	Name             *Ident
	TableName        *ObjectName
	UsingPreColumns  *Using
	Columns          []*ColumnIdent
	UsingPostColumns *Using
}

func (s *CreateIndexStmt) GetNameForDiff() string {
	return s.Name.StringForDiff()
}

func (s *CreateIndexStmt) String() string {
	var str strings.Builder
	if s.Comment != "" {
		comments := strings.Split(s.Comment, "\n")
		for i := range comments {
			if comments[i] != "" {
				str.WriteString(CommentPrefix + comments[i] + "\n")
			}
		}
	}
	str.WriteString("CREATE ")
	if s.Unique {
		str.WriteString("UNIQUE ")
	}
	str.WriteString("INDEX ")
	if s.IfNotExists {
		str.WriteString("IF NOT EXISTS ")
	}
	str.WriteString(s.Name.String() + " ON " + s.TableName.String())
	if s.UsingPreColumns != nil {
		str.WriteString(" " + s.UsingPreColumns.String())
	}
	str.WriteString(" (" + stringz.JoinStringers(", ", s.Columns...) + ")")
	if s.UsingPostColumns != nil {
		str.WriteString(" " + s.UsingPostColumns.String())
	}
	str.WriteString(";\n")
	return str.String()
}

func (s *CreateIndexStmt) StringForDiff() string {
	var str strings.Builder
	str.WriteString("CREATE ")
	if s.Unique {
		str.WriteString("UNIQUE ")
	}
	str.WriteString("INDEX ")
	str.WriteString(s.Name.StringForDiff() + " ON " + s.TableName.StringForDiff())
	if s.UsingPreColumns != nil {
		str.WriteString(" " + s.UsingPreColumns.String())
	}
	str.WriteString(" (")
	for i, c := range s.Columns {
		if i > 0 {
			str.WriteString(", ")
		}
		str.WriteString(c.StringForDiff())
	}
	str.WriteString(")")
	if s.UsingPostColumns != nil {
		str.WriteString(" " + s.UsingPostColumns.String())
	}
	str.WriteString(";\n")
	return str.String()
}

func (*CreateIndexStmt) isStmt()            {}
func (s *CreateIndexStmt) GoString() string { return internal.GoString(*s) }
