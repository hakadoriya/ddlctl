package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hakadoriya/z.go/databasez/sqlz"

	apperr "github.com/hakadoriya/ddlctl/pkg/apperr"
)

type sqlQueryerContext = interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

type showCreateAllTablesConfig struct {
	database string
}

type ShowCreateAllTablesOption interface {
	apply(cfg *showCreateAllTablesConfig)
}

type showCreateAllTablesOptionDatabase struct{ database string }

func (o *showCreateAllTablesOptionDatabase) apply(config *showCreateAllTablesConfig) {
	config.database = o.database
}

func WithShowCreateAllTablesOptionSchema(database string) ShowCreateAllTablesOption { //nolint:ireturn
	return &showCreateAllTablesOptionDatabase{database: database}
}

func ShowCreateAllTables(ctx context.Context, db sqlQueryerContext, opts ...ShowCreateAllTablesOption) (query string, err error) {
	dbz := sqlz.NewDB(db)

	cfg := new(showCreateAllTablesConfig)
	for _, opt := range opts {
		opt.apply(cfg)
	}

	databaseQuoted := func() string {
		if cfg.database != "" {
			return fmt.Sprintf("`%s`", cfg.database)
		}
		return "database()"
	}()

	type TableName struct {
		TableName string `db:"TABLE_NAME"`
	}

	tableNames := new([]*TableName)
	tableNamesQuery := "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = " + databaseQuoted
	if err := dbz.QueryContext(ctx, tableNames, tableNamesQuery); err != nil {
		return "", apperr.Errorf("dbz.QueryContext: q=%s: %w", tableNamesQuery, err)
	}

	// type CreateStatement struct {
	// 	CreateStatement string `db:"create_statement"`
	// }
	type ShowCreateTable struct {
		TableName       string `db:"Table"`
		CreateStatement string `db:"Create Table"`
	}
	for _, tn := range *tableNames {
		showCreateTable := new(ShowCreateTable)
		showCreateTableQuery := fmt.Sprintf("SHOW CREATE TABLE `%s`", tn.TableName)
		if err := dbz.QueryContext(ctx, showCreateTable, showCreateTableQuery); err != nil {
			return "", apperr.Errorf("dbz.QueryContext: q=%s: %w", showCreateTableQuery, err)
		}
		query += showCreateTable.CreateStatement + ";\n"

		// MEMO: for INDEX
		// showCreateIndex := fmt.Sprintf("SELECT CONCAT('CREATE INDEX ', INDEX_NAME, ' ON ', TABLE_NAME, ' (', GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX), ');') AS 'create_statement' FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = %s AND INDEX_NAME IS NOT NULL AND INDEX_NAME != 'PRIMARY' AND TABLE_NAME = '%s' GROUP BY INDEX_NAME, TABLE_NAME;", databaseQuoted, tn.TableName)
		// createStatements := new([]*CreateStatement)
		// if err := dbz.QueryContext(ctx, createStatements, showCreateIndex); err != nil {
		// 	return "", apperr.Errorf("dbz.QueryContext: q=%s: %w", showCreateIndex, err)
		// }
		// for _, createStatement := range *createStatements {
		// 	query += createStatement.CreateStatement + "\n"
		// }
	}

	return query, nil
}
