package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hakadoriya/z.go/databasez/sqlz"

	apperr "github.com/kunitsucom/ddlctl/pkg/apperr"
)

type sqlQueryerContext = interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

const (
	formatShowCreateAllTables = `-- CREATE TABLE
SELECT
    'CREATE TABLE ' || clmn.table_schema || '.' || clmn.table_name || ' (' || E'\n' || '  ' ||
    clmn.column_defs ||
    (CASE WHEN cnst.constraint_defs IS NOT NULL THEN ',' || E'\n' || '  ' || cnst.constraint_defs ELSE '' END) || E'\n' || ');' AS create_statement
FROM
    (
        -- COLUMN DEFINITIONS
        SELECT
            c.table_schema,
            c.table_name,
            string_agg(
                c.column_name || ' ' || c.data_type ||
                (CASE WHEN c.character_maximum_length IS NOT NULL THEN '(' || c.character_maximum_length || ')' ELSE '' END) ||
                (CASE WHEN c.is_nullable = 'NO' THEN ' NOT NULL' ELSE '' END) ||
                (CASE WHEN c.column_default IS NOT NULL THEN ' DEFAULT ' || c.column_default ELSE '' END),
                ',' || E'\n' || '  ' ORDER BY c.ordinal_position
            ) AS column_defs
        FROM
            information_schema.columns c
        WHERE
            c.table_schema = '%s'
        GROUP BY
            c.table_schema, c.table_name
    ) clmn
LEFT JOIN
    (
        -- CONSTRAINT DEFINITIONS
        SELECT
            tc.table_schema,
            tc.table_name,
            string_agg(
                'CONSTRAINT ' || tc.constraint_name || ' ' || tc.constraint_type || ' (' || kcu.column_names || ')' ||
                (CASE WHEN rc.update_rule IS NOT NULL THEN ' ON UPDATE ' || rc.update_rule ELSE '' END) ||
                (CASE WHEN rc.delete_rule IS NOT NULL THEN ' ON DELETE ' || rc.delete_rule ELSE '' END),
                ',' || E'\n' || '  '
            ) AS constraint_defs
        FROM
            information_schema.table_constraints tc
        JOIN
            (
                -- CONSTRAINT COLUMN NAMES
                SELECT
                    kcu.table_schema,
                    kcu.table_name,
                    kcu.constraint_name,
                    string_agg(kcu.column_name, ', ' ORDER BY kcu.ordinal_position) AS column_names
                FROM
                    information_schema.key_column_usage kcu
                GROUP BY
                    kcu.table_schema, kcu.table_name, kcu.constraint_name
            ) kcu ON tc.table_schema = kcu.table_schema AND tc.table_name = kcu.table_name AND tc.constraint_name = kcu.constraint_name
        LEFT JOIN
            information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name
        WHERE
            tc.constraint_type IN ('PRIMARY KEY', 'FOREIGN KEY', 'UNIQUE')
        GROUP BY
            tc.table_schema, tc.table_name
    ) cnst ON clmn.table_schema = cnst.table_schema AND clmn.table_name = cnst.table_name
ORDER BY
    clmn.table_schema, clmn.table_name
;
`
	// 	formatShowCreateAllIndexes = `-- CREATE INDEX
	// SELECT
	//     indexdef AS create_statement
	// FROM
	//     pg_indexes
	// WHERE
	//     schemaname = '%s' AND indexname NOT IN (
	//         SELECT constraint_name
	//         FROM information_schema.table_constraints
	//         WHERE constraint_type = 'PRIMARY KEY' AND table_schema = '%s'
	//     )
	// ;
	// `
	// MEMO: PRIMARY KEY 以外にも UNIQUE 制約も INDEX として扱われるため PRIMARY KEY 以外も除外するようにした
	formatShowCreateAllIndexes = `-- CREATE INDEX
SELECT
    indexdef AS create_statement
FROM
    pg_indexes
WHERE
    schemaname = '%s' AND indexname NOT IN (
        SELECT constraint_name
        FROM information_schema.table_constraints
        WHERE table_schema = '%s'
    )
;
`
)

type showCreateAllTablesConfig struct {
	schema string
}

type ShowCreateAllTablesOption interface {
	apply(cfg *showCreateAllTablesConfig)
}

type showCreateAllTablesOptionSchema struct{ schema string }

func (o *showCreateAllTablesOptionSchema) apply(config *showCreateAllTablesConfig) {
	config.schema = o.schema
}

func WithShowCreateAllTablesOptionSchema(schema string) ShowCreateAllTablesOption { //nolint:ireturn
	return &showCreateAllTablesOptionSchema{schema: schema}
}

func ShowCreateAllTables(ctx context.Context, db sqlQueryerContext, opts ...ShowCreateAllTablesOption) (query string, err error) {
	dbz := sqlz.NewDB(db)

	cfg := &showCreateAllTablesConfig{
		schema: "public",
	}
	for _, opt := range opts {
		opt.apply(cfg)
	}

	type CreateStatement struct {
		CreateStatement string `db:"create_statement"`
	}

	createTableStmts := new([]*CreateStatement)
	if err := dbz.QueryContext(ctx, createTableStmts, fmt.Sprintf(formatShowCreateAllTables, cfg.schema)); err != nil {
		return "", apperr.Errorf("dbz.QueryContext: %w", err)
	}
	for _, stmt := range *createTableStmts {
		query += stmt.CreateStatement + "\n"
	}

	createIndexStmts := new([]*CreateStatement)
	if err := dbz.QueryContext(ctx, createIndexStmts, fmt.Sprintf(formatShowCreateAllIndexes, cfg.schema, cfg.schema)); err != nil {
		return "", apperr.Errorf("dbz.QueryContext: %w", err)
	}
	for _, stmt := range *createIndexStmts {
		query += stmt.CreateStatement + ";\n"
	}

	return query, nil
}
