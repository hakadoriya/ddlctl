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
	querySelectTableName = `SELECT TABLE_NAME, PARENT_TABLE_NAME, ON_DELETE_ACTION, INTERLEAVE_TYPE FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '';`
)

type informationSchemaTable struct {
	// TABLES https://cloud.google.com/spanner/docs/information-schema?hl=ja#tables
	TableName       string  `db:"TABLE_NAME"`
	ParentTableName *string `db:"PARENT_TABLE_NAME"`
	OnDeleteAction  *string `db:"ON_DELETE_ACTION"`
	InterleaveType  *string `db:"INTERLEAVE_TYPE"`
}

const (
	// interleaveTypeIn は PARENT なしの INTERLEAVE IN でインターリーブしたテーブルの INTERLEAVE_TYPE
	interleaveTypeIn = "IN"
	// onDeleteActionCascade は ON DELETE CASCADE を指定したテーブルの ON_DELETE_ACTION
	onDeleteActionCascade = "CASCADE"
)

// sortParentsFirst は、親テーブルが子テーブルより前に来るように tables を並べ替えた slice を返す。
// 親子関係のないテーブル同士は元の順序を保つ。一覧にない親テーブルは無視する。
//
// INFORMATION_SCHEMA.TABLES は子テーブルを親テーブルより先に返すことがある (Cloud Spanner エミュレータで確認)。
// その順のまま INTERLEAVE IN PARENT を出力すると、ShowCreateAllTables の出力を空のデータベースに適用したときに、
// 子テーブルの CREATE TABLE が親テーブルを見つけられずに失敗する。
func sortParentsFirst(tables []*informationSchemaTable) []*informationSchemaTable {
	byName := make(map[string]*informationSchemaTable, len(tables))
	for _, tbl := range tables {
		byName[tbl.TableName] = tbl
	}

	sorted := make([]*informationSchemaTable, 0, len(tables))
	visited := make(map[string]bool, len(tables))
	var visit func(tbl *informationSchemaTable)
	visit = func(tbl *informationSchemaTable) {
		// NOTE: 親を辿る前に訪問済みにするので、仮に親子関係が循環していても無限に再帰しない
		if visited[tbl.TableName] {
			return
		}
		visited[tbl.TableName] = true
		if tbl.ParentTableName != nil {
			if parent, ok := byName[*tbl.ParentTableName]; ok {
				visit(parent)
			}
		}
		sorted = append(sorted, tbl)
	}
	for _, tbl := range tables {
		visit(tbl)
	}
	return sorted
}

// writeInterleaveInParent は tbl が INTERLEAVE IN PARENT でインターリーブしていれば、その句を PRIMARY KEY 句に続けて query に書き込む。
//
// INTERLEAVE IN PARENT を落とすと定義ファイルと食い違い、diff / apply が DROP TABLE + CREATE TABLE を生成してしまう。
func writeInterleaveInParent(query *strings.Builder, tbl *informationSchemaTable) {
	// NOTE: 親を持たないテーブルの INTERLEAVE_TYPE は、Cloud Spanner のドキュメントでは空文字列だが、
	// Cloud Spanner エミュレータ 1.5.38 では "IN PARENT" になる (実測) ので、PARENT_TABLE_NAME で判定する
	if tbl.ParentTableName == nil {
		return
	}
	// NOTE: PARENT なしの INTERLEAVE IN は pkg/ddl/spanner のパーサーが未対応なので出力しない。
	// 出力すると diff / apply が ShowCreateAllTables の出力をパースできずに失敗する
	if tbl.InterleaveType != nil && *tbl.InterleaveType == interleaveTypeIn {
		return
	}
	query.WriteString(",\nINTERLEAVE IN PARENT " + *tbl.ParentTableName)
	// NOTE: NO ACTION は ON DELETE を省略したときの既定値なので CASCADE だけを出力する (pkg/ddl/spanner のパーサーも同じ形に揃える)
	if tbl.OnDeleteAction != nil && *tbl.OnDeleteAction == onDeleteActionCascade {
		query.WriteString(" ON DELETE " + onDeleteActionCascade)
	}
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

// columnOrderingDesc は INDEX_COLUMNS.COLUMN_ORDERING が降順のカラムを表す値
const columnOrderingDesc = "DESC"

const (
	querySelectIndexes = `SELECT DISTINCT INDEX_NAME, IS_UNIQUE FROM INFORMATION_SCHEMA.INDEXES WHERE TABLE_NAME = ? AND INDEX_TYPE != "PRIMARY_KEY";`
)

type informationSchemaIndexName struct {
	IndexName string `db:"INDEX_NAME"`
	IsUnique  bool   `db:"IS_UNIQUE"`
}

const (
	// NOTE: INDEX_COLUMNS 自身が INDEX_TYPE を持つので INDEXES とは JOIN しない。
	// JOIN すると、JOIN キーを 1 つでも取りこぼしたときに各カラムがインデックスの数だけ重複し、しかも fake driver のテストでは検出できない。
	queryShowIndexes = `-- SHOW INDEXES
SELECT
    INDEX_NAME,
    INDEX_TYPE,
    COLUMN_NAME,
    COLUMN_ORDERING,
    ORDINAL_POSITION
FROM
    INFORMATION_SCHEMA.INDEX_COLUMNS
WHERE
    TABLE_NAME = ?
    AND INDEX_TYPE != "PRIMARY_KEY"
ORDER BY
    INDEX_NAME, ORDINAL_POSITION
;
`
)

type informationSchemaIndex struct {
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
	tables = sortParentsFirst(tables)

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
				// NOTE: ASC は省略時の既定値なので DESC だけを出力する。
				// DESC を落とすと定義ファイルの主キーと食い違い、diff / apply が DROP TABLE + CREATE TABLE を生成してしまう。
				if pk.ColumnOrdering == columnOrderingDesc {
					query.WriteString(" " + columnOrderingDesc)
				}
				if i != primaryKeyColumnsLastIndex {
					query.WriteString(", ")
				}
			}
			query.WriteString(")")
		}

		writeInterleaveInParent(&query, tbl)

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

		if len(indexNames) > 0 {
			allIndexColumns := make([]*informationSchemaIndex, 0)
			if err := dbz.QueryContext(ctx, &allIndexColumns, queryShowIndexes, tbl.TableName); err != nil {
				return "", apperr.Errorf("dbz.QueryContext: %w", err)
			}
			writeCreateIndexes(&query, tbl.TableName, indexNames, allIndexColumns)
		}

		if tblIdx != tablesLastIndex {
			query.WriteString("\n")
		}
	}

	return query.String(), nil
}

// writeCreateIndexes は tableName のセカンダリインデックスごとの CREATE INDEX 文を query に書き込む。
//
// allIndexColumns は queryShowIndexes の結果で、テーブルの全インデックスのカラムを含むため、インデックス名で振り分けて使う。
// 振り分けずに使うと、全インデックスのカラムが連結された CREATE INDEX を出力してしまう。
func writeCreateIndexes(query *strings.Builder, tableName string, indexNames []*informationSchemaIndexName, allIndexColumns []*informationSchemaIndex) {
	for _, indexName := range indexNames {
		columnNames := make([]string, 0)
		for _, idx := range allIndexColumns {
			if idx.IndexName == indexName.IndexName {
				columnNames = append(columnNames, idx.ColumnName)
			}
		}

		query.WriteString("CREATE ")
		if indexName.IsUnique {
			query.WriteString("UNIQUE ")
		}
		fmt.Fprintf(query, "INDEX %s ON %s (%s);\n", indexName.IndexName, tableName, strings.Join(columnNames, ", "))
	}
}
