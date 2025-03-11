package cockroachdb

import (
	"context"
	"database/sql"

	"github.com/hakadoriya/z.go/databasez/sqlz"

	"github.com/hakadoriya/ddlctl/pkg/apperr"
)

type sqlQueryerContext = interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

const (
	queryShowCreateAllTables = `-- CREATE TABLE
SHOW CREATE ALL TABLES
;
`
)

func ShowCreateAllTables(ctx context.Context, db sqlQueryerContext) (query string, err error) {
	dbz := sqlz.NewDB(db)

	type CreateStatement struct {
		CreateStatement string `db:"create_statement"`
	}

	createTableStmts := new([]*CreateStatement)
	if err := dbz.QueryContext(ctx, createTableStmts, queryShowCreateAllTables); err != nil {
		return "", apperr.Errorf("dbz.QueryContext: %w", err)
	}
	for _, stmt := range *createTableStmts {
		query += stmt.CreateStatement + "\n"
	}

	return query, nil
}
