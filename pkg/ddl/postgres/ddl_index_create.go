package postgres

import (
	"strings"

	"github.com/hakadoriya/z.go/stringz"

	"github.com/hakadoriya/ddlctl/pkg/ddl/internal"
)

// MEMO: https://www.cockroachlabs.com/docs/stable/create-index

var _ Stmt = (*CreateIndexStmt)(nil)

type CreateIndexStmt struct {
	Comment     string
	Unique      bool
	IfNotExists bool
	Name        *Ident
	TableName   *ObjectName
	Using       []*Ident
	Columns     []*ColumnIdent
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
	if len(s.Using) > 0 {
		str += " USING "
		str += stringz.JoinStringers(" ", s.Using...)
	}
	str += " (" + stringz.JoinStringers(", ", s.Columns...) + ");\n"
	return str
}

func (s *CreateIndexStmt) StringForDiff() string {
	str := "CREATE "
	if s.Unique {
		str += "UNIQUE "
	}
	str += "INDEX "
	str += s.Name.StringForDiff() + " ON " + s.TableName.StringForDiff()
	// TODO: add USING
	str += " ("
	for i, c := range s.Columns {
		if i > 0 {
			str += ", "
		}
		str += c.StringForDiff()
	}
	str += ");\n"
	return str
}

func (*CreateIndexStmt) isStmt()            {}
func (s *CreateIndexStmt) GoString() string { return internal.GoString(*s) }
