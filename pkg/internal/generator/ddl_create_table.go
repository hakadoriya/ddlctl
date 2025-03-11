package generator

import (
	"regexp"

	"github.com/hakadoriya/ddlctl/pkg/internal/lang/util"
)

var _ Stmt = (*CreateTableStmt)(nil)

type CreateTableStmt struct {
	SourceFile  string
	SourceLine  int
	Comments    []string                 // -- <Comment>
	CreateTable string                   // CREATE TABLE [IF NOT EXISTS] <Table>
	Columns     []*CreateTableColumn     // ( <Column>, ...
	Constraints []*CreateTableConstraint // <Constraint> )
	Options     []*CreateTableOption     // <Options>;
	PrimaryKey  []string                 // PRIMARY KEY ( <Column>, ... )
}

func (stmt *CreateTableStmt) GetSourceFile() string {
	return stmt.SourceFile
}

func (stmt *CreateTableStmt) GetSourceLine() int {
	return stmt.SourceLine
}

func (*CreateTableStmt) private() {}

//nolint:gochecknoglobals
var stmtRegexCreateTable = &util.StmtRegex{
	Regex: regexp.MustCompile(`(?i)\s*CREATE\s+(.*)?TABLE\s+(.*)?(\S+)`),
	Index: 3, //nolint:mnd // Index 3 is TABLE name
}

func (stmt *CreateTableStmt) SetCreateTable(createTable string) {
	if len(stmtRegexCreateTable.Regex.FindStringSubmatch(createTable)) > stmtRegexCreateTable.Index {
		stmt.CreateTable = createTable
		return
	}

	stmt.CreateTable = "CREATE TABLE " + createTable
}

type CreateTableColumn struct {
	Comments       []string
	ColumnName     string
	TypeConstraint string
}

type CreateTableConstraint struct {
	Comments   []string
	Constraint string
}

type CreateTableOption struct {
	Comments []string
	Option   string
}
