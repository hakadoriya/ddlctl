package mysql

import (
	"strings"

	"github.com/hakadoriya/ddlctl/pkg/ddl/internal"
)

// MEMO: https://dev.mysql.com/doc/refman/8.0/ja/drop-index.html

var _ Stmt = (*DropIndexStmt)(nil)

type DropIndexStmt struct {
	Comment  string
	IfExists bool
	Name     *ObjectName
}

func (s *DropIndexStmt) GetNameForDiff() string {
	return s.Name.StringForDiff()
}

func (s *DropIndexStmt) String() string {
	var str strings.Builder
	if s.Comment != "" {
		comments := strings.Split(s.Comment, "\n")
		for i := range comments {
			if comments[i] != "" {
				str.WriteString(CommentPrefix + comments[i] + "\n")
			}
		}
	}
	str.WriteString("DROP INDEX ")
	if s.IfExists {
		str.WriteString("IF EXISTS ")
	}
	str.WriteString(s.Name.String() + ";\n")
	return str.String()
}

func (*DropIndexStmt) isStmt()            {}
func (s *DropIndexStmt) GoString() string { return internal.GoString(*s) }
