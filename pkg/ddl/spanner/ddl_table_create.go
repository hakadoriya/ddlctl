package spanner

import (
	"strings"

	"github.com/hakadoriya/ddlctl/pkg/ddl/internal"
)

// MEMO: https://cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#create_table

var _ Stmt = (*CreateTableStmt)(nil)

type CreateTableStmt struct {
	Comment           string
	Indent            string
	IfNotExists       bool
	Name              *ObjectName
	Columns           []*Column
	Constraints       Constraints
	Options           Options
	RowDeletionPolicy *Option
}

func (s *CreateTableStmt) GetNameForDiff() string {
	return s.Name.StringForDiff()
}

//nolint:cyclop
func (s *CreateTableStmt) String() string {
	var str string
	if s.Comment != "" {
		comments := strings.Split(s.Comment, "\n")
		for i := range comments {
			if comments[i] != "" {
				str += CommentPrefix + comments[i] + "\n"
			}
		}
	}
	str += "CREATE TABLE "
	if s.IfNotExists {
		str += "IF NOT EXISTS "
	}
	str += s.Name.String() + " (\n"
	lastIndex := len(s.Columns) - 1
	hasConstraint := len(s.Constraints) > 0
	for i, v := range s.Columns {
		str += Indent
		str += v.String()
		if i != lastIndex || hasConstraint {
			str += ",\n"
		} else {
			str += "\n"
		}
	}
	if len(s.Constraints) > 0 {
		lastConstraint := len(s.Constraints) - 1
		for i, v := range s.Constraints {
			str += Indent
			str += v.String()
			if i != lastConstraint {
				str += ",\n"
			} else {
				str += "\n"
			}
		}
	}
	str += ")"
	if len(s.Options) > 0 {
		str += " "
		lastIndex := len(s.Options) - 1
		for i, v := range s.Options {
			str += v.String()
			if i != lastIndex {
				str += ",\n"
			}
		}
	}
	if s.RowDeletionPolicy != nil {
		str += ",\n" + s.RowDeletionPolicy.String()
	}

	str += ";\n"
	return str
}

func (*CreateTableStmt) isStmt()            {}
func (s *CreateTableStmt) GoString() string { return internal.GoString(*s) }
