package cockroachdb

import (
	"strings"

	"github.com/hakadoriya/z.go/stringz"

	"github.com/kunitsucom/ddlctl/pkg/ddl/internal"
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
	var str string
	if s.Comment != "" {
		comments := strings.Split(s.Comment, "\n")
		for i := range comments {
			if comments[i] != "" {
				str += CommentPrefix + comments[i] + "\n"
			}
		}
	}
	str += "CREATE "
	if s.Unique {
		str += "UNIQUE "
	}
	str += "INDEX "
	if s.IfNotExists {
		str += "IF NOT EXISTS "
	}
	str += s.Name.String() + " ON " + s.TableName.String()
	if s.UsingPreColumns != nil {
		str += " " + s.UsingPreColumns.String()
	}
	str += " (" + stringz.JoinStringers(", ", s.Columns...) + ")"
	if s.UsingPostColumns != nil {
		str += " " + s.UsingPostColumns.String()
	}
	str += ";\n"
	return str
}

func (s *CreateIndexStmt) StringForDiff() string {
	str := "CREATE "
	if s.Unique {
		str += "UNIQUE "
	}
	str += "INDEX "
	str += s.Name.StringForDiff() + " ON " + s.TableName.StringForDiff()
	if s.UsingPreColumns != nil {
		str += " " + s.UsingPreColumns.String()
	}
	str += " ("
	for i, c := range s.Columns {
		if i > 0 {
			str += ", "
		}
		str += c.StringForDiff()
	}
	str += ")"
	if s.UsingPostColumns != nil {
		str += " " + s.UsingPostColumns.String()
	}
	str += ";\n"
	return str
}

func (*CreateIndexStmt) isStmt()            {}
func (s *CreateIndexStmt) GoString() string { return internal.GoString(*s) }
