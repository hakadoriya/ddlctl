package mysql

import (
	"strings"

	"github.com/hakadoriya/ddlctl/pkg/ddl/internal"
)

// MEMO: https://dev.mysql.com/doc/refman/8.0/ja/create-table.html

var _ Stmt = (*CreateTableStmt)(nil)

type CreateTableStmt struct {
	Comment     string
	Indent      string
	IfNotExists bool
	Name        *ObjectName
	Columns     []*Column
	Constraints Constraints
	Options     Options
}

func (s *CreateTableStmt) GetNameForDiff() string {
	return s.Name.StringForDiff()
}

//nolint:cyclop
func (s *CreateTableStmt) String() string {
	var str strings.Builder
	if s.Comment != "" {
		comments := strings.Split(s.Comment, "\n")
		for i := range comments {
			if comments[i] != "" {
				str.WriteString(CommentPrefix + comments[i] + "\n")
			}
		}
	}
	str.WriteString("CREATE TABLE ")
	if s.IfNotExists {
		str.WriteString("IF NOT EXISTS ")
	}
	str.WriteString(s.Name.String() + " (\n")
	lastIndex := len(s.Columns) - 1
	hasConstraint := len(s.Constraints) > 0
	for i, v := range s.Columns {
		str.WriteString(Indent)
		str.WriteString(v.String())
		if i != lastIndex || hasConstraint {
			str.WriteString(",\n")
		} else {
			str.WriteString("\n")
		}
	}
	if len(s.Constraints) > 0 {
		lastConstraint := len(s.Constraints) - 1
		for i, v := range s.Constraints {
			str.WriteString(Indent)
			str.WriteString(v.String())
			if i != lastConstraint {
				str.WriteString(",\n")
			} else {
				str.WriteString("\n")
			}
		}
	}
	str.WriteString(")")
	if len(s.Options) > 0 {
		str.WriteString(" " + s.Options.String())
	}

	str.WriteString(";\n")
	return str.String()
}

func (*CreateTableStmt) isStmt()            {}
func (s *CreateTableStmt) GoString() string { return internal.GoString(*s) }
