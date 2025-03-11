package cockroachdb

import (
	"strings"

	"github.com/hakadoriya/ddlctl/pkg/ddl/internal"
)

// MEMO: https://www.cockroachlabs.com/docs/stable/drop-index

var _ Stmt = (*DropIndexStmt)(nil)

type DropIndexStmt struct {
	Comment  string
	IfExists bool
	Name     *Ident
}

func (s *DropIndexStmt) GetNameForDiff() string {
	return s.Name.StringForDiff()
}

func (s *DropIndexStmt) String() string {
	var str string
	if s.Comment != "" {
		comments := strings.Split(s.Comment, "\n")
		for i := range comments {
			if comments[i] != "" {
				str += CommentPrefix + comments[i] + "\n"
			}
		}
	}
	str += "DROP INDEX "
	if s.IfExists {
		str += "IF EXISTS "
	}
	str += s.Name.String() + ";\n"
	return str
}

func (*DropIndexStmt) isStmt()            {}
func (s *DropIndexStmt) GoString() string { return internal.GoString(*s) }
