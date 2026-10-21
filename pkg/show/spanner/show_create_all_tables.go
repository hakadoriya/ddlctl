package spanner

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/hakadoriya/z.go/databasez/sqlz"

	"github.com/hakadoriya/ddlctl/pkg/apperr"
	"github.com/hakadoriya/ddlctl/pkg/logs"
)

// NOTE: https://cloud.google.com/spanner/docs/information-schema?hl=ja

type sqlQueryerContext = interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

const (
	querySelectTableName = `SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '';`
)

type informationSchemaTable struct {
	TableName string `db:"TABLE_NAME"`
}

const (
	querySelectTableOptionRowDeletionPolicy = `SELECT TABLE_NAME, ROW_DELETION_POLICY_EXPRESSION FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '' AND TABLE_NAME = ? AND ROW_DELETION_POLICY_EXPRESSION != '';`
)

type informationSchemaTableOptionRowDeletionPolicy struct {
	TableName                   string `db:"TABLE_NAME"`
	RowDeletionPolicyExpression string `db:"ROW_DELETION_POLICY_EXPRESSION"`
}

const (
	queryShowCreateAllTables = `SELECT TABLE_NAME, COLUMN_NAME, COLUMN_DEFAULT, IS_NULLABLE, SPANNER_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ? ORDER BY TABLE_NAME, ORDINAL_POSITION;`
)

type informationSchemaColumn struct {
	TableName     string  `db:"TABLE_NAME"`
	ColumnName    string  `db:"COLUMN_NAME"`
	ColumnDefault *string `db:"COLUMN_DEFAULT"`
	IsNullable    string  `db:"IS_NULLABLE"`
	SpannerType   string  `db:"SPANNER_TYPE"`
}

func (c *informationSchemaColumn) String() string {
	d := fmt.Sprintf("%s %s", c.ColumnName, c.SpannerType)
	if c.ColumnDefault != nil {
		d += fmt.Sprintf(" DEFAULT (%s)", *c.ColumnDefault)
	}
	if c.IsNullable == "NO" {
		d += " NOT NULL"
	}
	return d
}

const (
	queryShowTableColumnOptions = `SELECT COLUMN_NAME, OPTION_NAME, OPTION_VALUE FROM INFORMATION_SCHEMA.COLUMN_OPTIONS WHERE TABLE_NAME = ?;`
)

type informationSchemaColumnOption struct {
	ColumnName  string `db:"COLUMN_NAME"`
	OptionName  string `db:"OPTION_NAME"`
	OptionValue string `db:"OPTION_VALUE"`
}

func (c *informationSchemaColumnOption) String() string {
	return fmt.Sprintf("%s = %s", c.OptionName, c.OptionValue)
}

const (
	// columnOptionNameLocalityGroup is the column option that Cloud Spanner implicitly
	// assigns to every column.
	//
	// ref. https://cloud.google.com/spanner/docs/reference/standard-sql/data-definition-language#column_options
	columnOptionNameLocalityGroup = "locality_group"

	// columnOptionValueDefaultLocalityGroup is the value Cloud Spanner assigns when no
	// locality group is explicitly specified.
	columnOptionValueDefaultLocalityGroup = "default"
)

// isImplicitlyAssigned reports whether the column option is implicitly assigned by Cloud Spanner
// rather than explicitly specified by the user.
//
// Cloud Spanner exposes `locality_group = default` in INFORMATION_SCHEMA.COLUMN_OPTIONS for every
// column even when the user has never specified a locality group. Including it in the DDL that
// represents the current schema makes ddlctl report a diff against DDL that has no OPTIONS clause,
// and the generated `ALTER COLUMN ... DROP OPTIONS` fails because Cloud Spanner has no such syntax.
//
// Only the implicit default is skipped; an explicitly specified locality group is still reported
// so that changes to it are detected.
func (c *informationSchemaColumnOption) isImplicitlyAssigned() bool {
	return c.OptionName == columnOptionNameLocalityGroup && c.OptionValue == columnOptionValueDefaultLocalityGroup
}

const (
	queryShowPrimaryKey = `-- SHOW TABLES
SELECT
    i.INDEX_NAME,
    i.INDEX_TYPE,
    ic.COLUMN_NAME,
    ic.COLUMN_ORDERING,
    ic.ORDINAL_POSITION
FROM
    INFORMATION_SCHEMA.INDEXES AS i
INNER JOIN
    INFORMATION_SCHEMA.INDEX_COLUMNS AS ic
ON
    i.TABLE_NAME = ic.TABLE_NAME
WHERE
    i.TABLE_NAME = ?
    AND i.INDEX_TYPE = "PRIMARY_KEY"
    AND ic.INDEX_TYPE = "PRIMARY_KEY"
ORDER BY
    i.TABLE_NAME, ic.ORDINAL_POSITION
;
`
)

type informationSchemaPrimaryKey struct {
	// INDEXES https://cloud.google.com/spanner/docs/information-schema?hl=ja#indexes
	// INDEX_COLUMNS https://cloud.google.com/spanner/docs/information-schema?hl=ja#index_columns
	IndexName       string `db:"INDEX_NAME"`
	IndexType       string `db:"INDEX_TYPE"`
	ColumnName      string `db:"COLUMN_NAME"`
	ColumnOrdering  string `db:"COLUMN_ORDERING"`
	OrdinalPosition int    `db:"ORDINAL_POSITION"`
}

const (
	querySelectIndexes = `SELECT DISTINCT INDEX_NAME, IS_UNIQUE FROM INFORMATION_SCHEMA.INDEXES WHERE TABLE_NAME = ? AND INDEX_TYPE != "PRIMARY_KEY";`
)

type informationSchemaIndexName struct {
	IndexName string `db:"INDEX_NAME"`
	IsUnique  bool   `db:"IS_UNIQUE"`
}

const (
	queryShowIndexes = `-- SHOW INDEXES
SELECT
    ic.INDEX_NAME,
    i.INDEX_TYPE,
    ic.COLUMN_NAME,
    ic.COLUMN_ORDERING,
    ic.ORDINAL_POSITION
FROM
    INFORMATION_SCHEMA.INDEXES AS i
INNER JOIN
    INFORMATION_SCHEMA.INDEX_COLUMNS AS ic
ON
    i.TABLE_NAME = ic.TABLE_NAME
WHERE
    i.TABLE_NAME = ? 
    AND i.INDEX_TYPE != "PRIMARY_KEY" 
    AND ic.INDEX_NAME != "PRIMARY_KEY" 
ORDER BY
    i.TABLE_NAME, ic.INDEX_NAME, ic.ORDINAL_POSITION
;
`
)

type informationSchemaIndex struct {
	// INDEXES https://cloud.google.com/spanner/docs/information-schema?hl=ja#indexes
	// INDEX_COLUMNS https://cloud.google.com/spanner/docs/information-schema?hl=ja#index_columns
	IndexName       string `db:"INDEX_NAME"`
	IndexType       string `db:"INDEX_TYPE"`
	ColumnName      string `db:"COLUMN_NAME"`
	ColumnOrdering  string `db:"COLUMN_ORDERING"`
	OrdinalPosition int    `db:"ORDINAL_POSITION"`
}

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

//nolint:cyclop,funlen,gocognit
func ShowCreateAllTables(ctx context.Context, db sqlQueryerContext, opts ...ShowCreateAllTablesOption) (string, error) {
	dbz := sqlz.NewDB(db)

	cfg := &showCreateAllTablesConfig{
		schema: "public",
	}
	for _, opt := range opts {
		opt.apply(cfg)
	}

	tables := make([]*informationSchemaTable, 0)
	if err := dbz.QueryContext(ctx, &tables, querySelectTableName); err != nil {
		return "", apperr.Errorf("dbz.QueryContext: %w", err)
	}

	var query strings.Builder
	tablesLastIndex := len(tables) - 1
	for tblIdx, tbl := range tables {
		// TABLE
		fmt.Fprintf(&query, "CREATE TABLE %s (\n", tbl.TableName)

		columns := make([]*informationSchemaColumn, 0)
		if err := dbz.QueryContext(ctx, &columns, queryShowCreateAllTables, tbl.TableName); err != nil {
			return "", apperr.Errorf("dbz.QueryContext: %w", err)
		}

		allColumnOptions := make([]*informationSchemaColumnOption, 0)
		if err := dbz.QueryContext(ctx, &allColumnOptions, queryShowTableColumnOptions, tbl.TableName); err != nil {
			return "", apperr.Errorf("dbz.QueryContext: %w", err)
		}

		columnsLastIndex := len(columns) - 1
		for colIdx, col := range columns {
			fmt.Fprintf(&query, "    %s", col)
			if len(allColumnOptions) > 0 {
				columnOptions := make([]*informationSchemaColumnOption, 0)
				for _, opt := range allColumnOptions {
					// NOTE: Skip options that Cloud Spanner assigns implicitly. See isImplicitlyAssigned.
					if col.ColumnName == opt.ColumnName && !opt.isImplicitlyAssigned() {
						columnOptions = append(columnOptions, opt)
					}
				}
				if len(columnOptions) > 0 {
					query.WriteString(" OPTIONS (")
					for columnOptionsIdx, opt := range columnOptions {
						query.WriteString(opt.String())
						if columnOptionsLastIndex := len(columnOptions) - 1; columnOptionsIdx != columnOptionsLastIndex {
							query.WriteString(", ")
						}
					}
					query.WriteString(")")
				}
			}
			if colIdx != columnsLastIndex {
				query.WriteString(",")
			}
			query.WriteString("\n")

			logs.Trace.Printf("table=%s: columns: %s", tbl.TableName, col)
		}
		query.WriteString(")")

		primaryKeyColumns := make([]*informationSchemaPrimaryKey, 0)
		if err := dbz.QueryContext(ctx, &primaryKeyColumns, queryShowPrimaryKey, tbl.TableName); err != nil {
			return "", apperr.Errorf("dbz.QueryContext: %w", err)
		}

		if len(primaryKeyColumns) > 0 {
			query.WriteString(" PRIMARY KEY (")
			primaryKeyColumnsLastIndex := len(primaryKeyColumns) - 1
			for i, pk := range primaryKeyColumns {
				query.WriteString(pk.ColumnName)
				if i != primaryKeyColumnsLastIndex {
					query.WriteString(", ")
				}
			}
			query.WriteString(")")
		}

		tableOptionRowDeletionPolicy := make([]*informationSchemaTableOptionRowDeletionPolicy, 0)
		if err := dbz.QueryContext(ctx, &tableOptionRowDeletionPolicy, querySelectTableOptionRowDeletionPolicy, tbl.TableName); err != nil {
			return "", apperr.Errorf("dbz.QueryContext: %w", err)
		}

		if len(tableOptionRowDeletionPolicy) > 0 {
			query.WriteString(",\nROW DELETION POLICY (")
			for _, opt := range tableOptionRowDeletionPolicy {
				query.WriteString(opt.RowDeletionPolicyExpression)
			}
			query.WriteString(")")
		}

		query.WriteString(";\n")

		// INDEX
		indexNames := make([]*informationSchemaIndexName, 0)
		if err := dbz.QueryContext(ctx, &indexNames, querySelectIndexes, tbl.TableName); err != nil {
			return "", apperr.Errorf("dbz.QueryContext: %w", err)
		}

		for _, indexName := range indexNames {
			indexes := make([]*informationSchemaIndex, 0)
			if err := dbz.QueryContext(ctx, &indexes, queryShowIndexes, tbl.TableName); err != nil {
				return "", apperr.Errorf("dbz.QueryContext: %w", err)
			}

			query.WriteString("CREATE ")
			if indexName.IsUnique {
				query.WriteString("UNIQUE ")
			}
			fmt.Fprintf(&query, "INDEX %s ON %s (", indexName.IndexName, tbl.TableName)

			indexesLastIndex := len(indexes) - 1
			for i, idx := range indexes {
				query.WriteString(idx.ColumnName)
				if i != indexesLastIndex {
					query.WriteString(", ")
				}
			}
			query.WriteString(");\n")
		}

		if tblIdx != tablesLastIndex {
			query.WriteString("\n")
		}
	}

	return query.String(), nil
}
