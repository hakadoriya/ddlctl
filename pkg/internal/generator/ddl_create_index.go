package generator

import (
	"regexp"

	"github.com/hakadoriya/ddlctl/pkg/internal/lang/util"
)

var _ Stmt = (*CreateIndexStmt)(nil)

type CreateIndexStmt struct {
	SourceFile  string
	SourceLine  int
	Comments    []string // -- <Comment>
	CreateIndex string   // CREATE INDEX [IF NOT EXISTS] <Index>
}

func (stmt *CreateIndexStmt) GetSourceFile() string {
	return stmt.SourceFile
}

func (stmt *CreateIndexStmt) GetSourceLine() int {
	return stmt.SourceLine
}

func (*CreateIndexStmt) private() {}

//nolint:gochecknoglobals
var stmtRegexCreateIndex = &util.StmtRegex{
	Regex: regexp.MustCompile(`(?i)\s*CREATE\s+(.*)?INDEX\s+(.*)?(\S+)`),
	Index: 3, //nolint:mnd // Index 3 is INDEX name
}

func (stmt *CreateIndexStmt) SetCreateIndex(createIndex string) {
	if len(stmtRegexCreateIndex.Regex.FindStringSubmatch(createIndex)) > stmtRegexCreateIndex.Index {
		stmt.CreateIndex = createIndex
		return
	}

	stmt.CreateIndex = "CREATE INDEX " + createIndex
}
