package cockroachdb

import (
	"strings"

	"github.com/hakadoriya/ddlctl/pkg/ddl/internal"
)

// MEMO: https://www.postgresql.jp/docs/11/sql-createtable.html

var _ Stmt = (*DropTableStmt)(nil)

type DropTableStmt struct {
	Comment  string
	IfExists bool
	Name     *ObjectName
}

func (s *DropTableStmt) GetNameForDiff() string {
	return s.Name.StringForDiff()
}

func (s *DropTableStmt) String() string {
	var str strings.Builder
	if s.Comment != "" {
		comments := strings.Split(s.Comment, "\n")
		for i := range comments {
			if comments[i] != "" {
				str.WriteString(CommentPrefix + comments[i] + "\n")
			}
		}
	}
	str.WriteString("DROP TABLE ")
	if s.IfExists {
		str.WriteString("IF EXISTS ")
	}
	str.WriteString(s.Name.String() + ";\n")
	return str.String()
}

func (*DropTableStmt) isStmt()            {}
func (s *DropTableStmt) GoString() string { return internal.GoString(*s) }
