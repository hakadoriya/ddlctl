package spanner

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/quick"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
	require "github.com/hakadoriya/z.go/testingz/requirez"

	"github.com/hakadoriya/ddlctl/pkg/ddl"
	ddlspanner "github.com/hakadoriya/ddlctl/pkg/ddl/spanner"
	"github.com/hakadoriya/ddlctl/pkg/internal/fakesql"
)

func TestInformationSchemaColumnOption_isImplicitlyAssigned(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name     string
		option   *informationSchemaColumnOption
		expected bool
	}{
		{
			// Cloud Spanner exposes this for every column even when the user has never
			// specified a locality group. Including it makes ddlctl emit an invalid
			// `ALTER COLUMN ... DROP OPTIONS` against DDL that has no OPTIONS clause.
			name:     "true,locality_group=default is implicitly assigned",
			option:   &informationSchemaColumnOption{ColumnName: "Id", OptionName: "locality_group", OptionValue: "default"},
			expected: true,
		},
		{
			// An explicitly specified locality group must still be reported so that
			// changes to it are detected.
			name:     "false,locality_group with a non-default value is explicit",
			option:   &informationSchemaColumnOption{ColumnName: "Id", OptionName: "locality_group", OptionValue: "archive"},
			expected: false,
		},
		{
			name:     "false,allow_commit_timestamp is explicit",
			option:   &informationSchemaColumnOption{ColumnName: "CreatedAt", OptionName: "allow_commit_timestamp", OptionValue: "TRUE"},
			expected: false,
		},
		{
			// Guard against matching on the value alone.
			name:     "false,another option whose value happens to be default",
			option:   &informationSchemaColumnOption{ColumnName: "Id", OptionName: "some_other_option", OptionValue: "default"},
			expected: false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.option.isImplicitlyAssigned())
		})
	}
}

// showCreateAllTablesFixture はテーブル名とクエリの組ごとに INFORMATION_SCHEMA の問い合わせ結果を保持する
type showCreateAllTablesFixture map[string]map[string]*fakesql.Rows

// queryFunc は fixture に応答する QueryFunc を返す。failQuery に一致するクエリには errQuery を返す。
func (f showCreateAllTablesFixture) queryFunc(t *testing.T, tableNames []string, failQuery string, errQuery error) fakesql.QueryFunc {
	t.Helper()

	return func(query string, args []driver.NamedValue) (*fakesql.Rows, error) {
		if query == failQuery {
			return nil, errQuery
		}
		if query == querySelectTableName {
			r := &fakesql.Rows{Columns: []string{"TABLE_NAME"}}
			for _, tableName := range tableNames {
				r.Values = append(r.Values, []driver.Value{tableName})
			}
			return r, nil
		}
		// querySelectTableName 以外のクエリは全てテーブル名を唯一の引数に取る
		require.Equal(t, 1, len(args))
		tableName, ok := args[0].Value.(string)
		require.True(t, ok)
		r, ok := f[tableName][query]
		if !ok {
			t.Errorf("unexpected query: table=%s: %s", tableName, query)
			return nil, fakesql.ErrNoRowsPrepared
		}
		return r, nil
	}
}

func TestShowCreateAllTables(t *testing.T) {
	t.Parallel()

	var (
		columnColumns            = []string{"TABLE_NAME", "COLUMN_NAME", "COLUMN_DEFAULT", "IS_NULLABLE", "SPANNER_TYPE"}
		columnOptionColumns      = []string{"COLUMN_NAME", "OPTION_NAME", "OPTION_VALUE"}
		indexColumns             = []string{"INDEX_NAME", "INDEX_TYPE", "COLUMN_NAME", "COLUMN_ORDERING", "ORDINAL_POSITION"}
		rowDeletionPolicyColumns = []string{"TABLE_NAME", "ROW_DELETION_POLICY_EXPRESSION"}
		indexNameColumns         = []string{"INDEX_NAME", "IS_UNIQUE"}
	)

	tableNames := []string{"Users", "Logs"}
	fixture := showCreateAllTablesFixture{
		// カラムオプション (暗黙に付与されるものを含む) ・ DEFAULT ・ UNIQUE INDEX を持つテーブル
		"Users": {
			queryShowCreateAllTables: {Columns: columnColumns, Values: [][]driver.Value{
				{"Users", "Id", nil, "NO", "STRING(36)"},
				{"Users", "Name", nil, "YES", "STRING(MAX)"},
				{"Users", "CreatedAt", "CURRENT_TIMESTAMP()", "NO", "TIMESTAMP"},
			}},
			queryShowTableColumnOptions: {Columns: columnOptionColumns, Values: [][]driver.Value{
				{"Id", "locality_group", "default"},
				{"CreatedAt", "allow_commit_timestamp", "TRUE"},
				{"CreatedAt", "locality_group", "archive"},
			}},
			queryShowPrimaryKey: {Columns: indexColumns, Values: [][]driver.Value{
				{"PRIMARY_KEY", "PRIMARY_KEY", "Id", "ASC", int64(1)},
			}},
			querySelectTableOptionRowDeletionPolicy: {Columns: rowDeletionPolicyColumns},
			querySelectIndexes: {Columns: indexNameColumns, Values: [][]driver.Value{
				{"UsersByName", true},
			}},
			queryShowIndexes: {Columns: indexColumns, Values: [][]driver.Value{
				{"UsersByName", "INDEX", "Name", "ASC", int64(1)},
			}},
		},
		// 複合主キー・ ROW DELETION POLICY ・複合 INDEX を持ち、カラムオプションを持たないテーブル
		"Logs": {
			queryShowCreateAllTables: {Columns: columnColumns, Values: [][]driver.Value{
				{"Logs", "UserId", nil, "NO", "STRING(36)"},
				{"Logs", "LogId", nil, "NO", "INT64"},
				{"Logs", "LoggedAt", nil, "NO", "TIMESTAMP"},
				{"Logs", "Message", nil, "YES", "STRING(MAX)"},
			}},
			queryShowTableColumnOptions: {Columns: columnOptionColumns},
			queryShowPrimaryKey: {Columns: indexColumns, Values: [][]driver.Value{
				{"PRIMARY_KEY", "PRIMARY_KEY", "UserId", "ASC", int64(1)},
				{"PRIMARY_KEY", "PRIMARY_KEY", "LogId", "ASC", int64(2)},
			}},
			querySelectTableOptionRowDeletionPolicy: {Columns: rowDeletionPolicyColumns, Values: [][]driver.Value{
				{"Logs", "OLDER_THAN(LoggedAt, INTERVAL 30 DAY)"},
			}},
			querySelectIndexes: {Columns: indexNameColumns, Values: [][]driver.Value{
				{"LogsByUserIdLoggedAt", false},
			}},
			queryShowIndexes: {Columns: indexColumns, Values: [][]driver.Value{
				{"LogsByUserIdLoggedAt", "INDEX", "UserId", "ASC", int64(1)},
				{"LogsByUserIdLoggedAt", "INDEX", "LoggedAt", "ASC", int64(2)},
			}},
		},
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db := fakesql.OpenDB(fixture.queryFunc(t, tableNames, "", nil))
		t.Cleanup(func() { _ = db.Close() })

		const expected = `CREATE TABLE Users (
    Id STRING(36) NOT NULL,
    Name STRING(MAX),
    CreatedAt TIMESTAMP DEFAULT (CURRENT_TIMESTAMP()) NOT NULL OPTIONS (allow_commit_timestamp = TRUE, locality_group = archive)
) PRIMARY KEY (Id);
CREATE UNIQUE INDEX UsersByName ON Users (Name);

CREATE TABLE Logs (
    UserId STRING(36) NOT NULL,
    LogId INT64 NOT NULL,
    LoggedAt TIMESTAMP NOT NULL,
    Message STRING(MAX)
) PRIMARY KEY (UserId, LogId),
ROW DELETION POLICY (OLDER_THAN(LoggedAt, INTERVAL 30 DAY));
CREATE INDEX LogsByUserIdLoggedAt ON Logs (UserId, LoggedAt);
`
		actual, err := ShowCreateAllTables(context.Background(), db)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("success,NoTables", func(t *testing.T) {
		t.Parallel()

		db := fakesql.OpenDB(fixture.queryFunc(t, nil, "", nil))
		t.Cleanup(func() { _ = db.Close() })

		actual, err := ShowCreateAllTables(context.Background(), db)
		require.NoError(t, err)
		assert.Equal(t, "", actual)
	})

	for name, failQuery := range map[string]string{
		"querySelectTableName":                    querySelectTableName,
		"queryShowCreateAllTables":                queryShowCreateAllTables,
		"queryShowTableColumnOptions":             queryShowTableColumnOptions,
		"queryShowPrimaryKey":                     queryShowPrimaryKey,
		"querySelectTableOptionRowDeletionPolicy": querySelectTableOptionRowDeletionPolicy,
		"querySelectIndexes":                      querySelectIndexes,
		"queryShowIndexes":                        queryShowIndexes,
	} {
		t.Run("failure,"+name, func(t *testing.T) {
			t.Parallel()

			errQuery := errors.New("query failed")
			db := fakesql.OpenDB(fixture.queryFunc(t, tableNames, failQuery, errQuery))
			t.Cleanup(func() { _ = db.Close() })

			_, err := ShowCreateAllTables(context.Background(), db)
			require.ErrorIs(t, err, errQuery)
		})
	}
}

func TestShowCreateAllTables_PrimaryKeyOrderingRoundTrip(t *testing.T) {
	t.Parallel()

	// 主キーのカラム数の上限 (Cloud Spanner の上限 16 に合わせる)
	const maxPrimaryKeyColumns = 16

	// 定義ファイル側の並び順の書き方。ASC は省略・明示・小文字、DESC は大文字・小文字・先頭だけ大文字にゆらす
	ascSpellings := []string{"", " ASC", " asc"}
	descSpellings := []string{" DESC", " desc", " Desc"}

	// 主キーの各カラムの並び順 (ASC / DESC) と定義ファイル側の書き方を生成し、ShowCreateAllTables の出力をパースした結果が
	// 同じ並び順で書いた DDL と差分なしになることを検証する。あわせて、1 カラムだけ並び順を反転した DDL とは差分が出ることを検証する。
	// 差分なしにならないと ddlctl diff / apply が毎回 DROP TABLE + CREATE TABLE を生成し、テーブルのデータが消える。
	// 差分が出ないと、主キーの並び順の変更を見落とす。
	property := func(specs []uint8, flip uint8) bool {
		if len(specs) == 0 {
			// 主キーのないテーブルは対象外
			return true
		}
		if len(specs) > maxPrimaryKeyColumns {
			specs = specs[:maxPrimaryKeyColumns]
		}
		flipped := int(flip) % len(specs)

		columns := &fakesql.Rows{Columns: []string{"TABLE_NAME", "COLUMN_NAME", "COLUMN_DEFAULT", "IS_NULLABLE", "SPANNER_TYPE"}}
		primaryKey := &fakesql.Rows{Columns: []string{"INDEX_NAME", "INDEX_TYPE", "COLUMN_NAME", "COLUMN_ORDERING", "ORDINAL_POSITION"}}
		columnDefs := make([]string, 0, len(specs))
		keyParts := make([]string, 0, len(specs))
		flippedKeyParts := make([]string, 0, len(specs))
		for i, spec := range specs {
			name := fmt.Sprintf("Key%d", i)
			desc := spec%2 == 1
			spelling := int(spec/2) % len(ascSpellings)
			ordering, keyPart, flippedKeyPart := "ASC", name+ascSpellings[spelling], name+descSpellings[spelling]
			if desc {
				ordering, keyPart, flippedKeyPart = "DESC", name+descSpellings[spelling], name+ascSpellings[spelling]
			}
			columns.Values = append(columns.Values, []driver.Value{"T", name, nil, "NO", "STRING(MAX)"})
			primaryKey.Values = append(primaryKey.Values, []driver.Value{"PRIMARY_KEY", "PRIMARY_KEY", name, ordering, int64(i + 1)})
			columnDefs = append(columnDefs, "    "+name+" STRING(MAX) NOT NULL")
			keyParts = append(keyParts, keyPart)
			if i == flipped {
				flippedKeyParts = append(flippedKeyParts, flippedKeyPart)
			} else {
				flippedKeyParts = append(flippedKeyParts, keyPart)
			}
		}
		ddlOf := func(keyParts []string) string {
			return "CREATE TABLE T (\n" + strings.Join(columnDefs, ",\n") + "\n) PRIMARY KEY (" + strings.Join(keyParts, ", ") + ");\n"
		}
		expected, flippedDDL := ddlOf(keyParts), ddlOf(flippedKeyParts)

		fixture := showCreateAllTablesFixture{"T": {
			queryShowCreateAllTables:                columns,
			queryShowTableColumnOptions:             {Columns: []string{"COLUMN_NAME", "OPTION_NAME", "OPTION_VALUE"}},
			queryShowPrimaryKey:                     primaryKey,
			querySelectTableOptionRowDeletionPolicy: {Columns: []string{"TABLE_NAME", "ROW_DELETION_POLICY_EXPRESSION"}},
			querySelectIndexes:                      {Columns: []string{"INDEX_NAME", "IS_UNIQUE"}},
		}}
		db := fakesql.OpenDB(fixture.queryFunc(t, []string{"T"}, "", nil))
		defer func() { _ = db.Close() }()

		actual, err := ShowCreateAllTables(context.Background(), db)
		if err != nil {
			t.Logf("ShowCreateAllTables: %v", err)
			return false
		}
		before, err := ddlspanner.NewParser(ddlspanner.NewLexer(actual)).Parse()
		if err != nil {
			t.Logf("parse ShowCreateAllTables output: %v\n%s", err, actual)
			return false
		}
		after, err := ddlspanner.NewParser(ddlspanner.NewLexer(expected)).Parse()
		if err != nil {
			t.Logf("parse expected DDL: %v\n%s", err, expected)
			return false
		}
		if _, err := ddlspanner.Diff(before, after); !errors.Is(err, ddl.ErrNoDifference) {
			t.Logf("unexpected difference: err=%v\n--- ShowCreateAllTables\n%s--- expected\n%s", err, actual, expected)
			return false
		}
		flippedAfter, err := ddlspanner.NewParser(ddlspanner.NewLexer(flippedDDL)).Parse()
		if err != nil {
			t.Logf("parse flipped DDL: %v\n%s", err, flippedDDL)
			return false
		}
		if _, err := ddlspanner.Diff(before, flippedAfter); err != nil {
			t.Logf("expected a difference: err=%v\n--- ShowCreateAllTables\n%s--- flipped\n%s", err, actual, flippedDDL)
			return false
		}
		return true
	}
	require.NoError(t, quick.Check(property, nil))
}

func TestShowCreateAllTables_IndexColumnsRoundTrip(t *testing.T) {
	t.Parallel()

	const (
		// インデックスに使えるカラムの数
		numIndexableColumns = 16
		// 1 テーブルあたりのインデックス数の上限
		maxIndexes = 8
		// 1 インデックスあたりのカラム数の上限
		maxIndexColumns = 4
	)

	// テーブルに張るインデックスの組 (各インデックスのカラム数・先頭カラム・ UNIQUE の有無) を生成し、ShowCreateAllTables の出力をパースした結果が
	// 同じインデックスを書いた DDL と差分なしになることを検証する。
	// 差分なしにならないと ddlctl diff / apply が毎回 DROP INDEX + CREATE INDEX を生成する。
	property := func(specs []uint16) bool {
		if len(specs) > maxIndexes {
			specs = specs[:maxIndexes]
		}

		columns := &fakesql.Rows{
			Columns: []string{"TABLE_NAME", "COLUMN_NAME", "COLUMN_DEFAULT", "IS_NULLABLE", "SPANNER_TYPE"},
			Values:  [][]driver.Value{{"T", "Id", nil, "NO", "STRING(MAX)"}},
		}
		columnDefs := make([]string, 0, 1+numIndexableColumns)
		columnDefs = append(columnDefs, "    Id STRING(MAX) NOT NULL")
		for i := range numIndexableColumns {
			name := fmt.Sprintf("Col%d", i)
			columns.Values = append(columns.Values, []driver.Value{"T", name, nil, "YES", "STRING(MAX)"})
			columnDefs = append(columnDefs, "    "+name+" STRING(MAX)")
		}

		indexNames := &fakesql.Rows{Columns: []string{"INDEX_NAME", "IS_UNIQUE"}}
		// 実際の queryShowIndexes と同じく、テーブルの全インデックスのカラムを INDEX_NAME ・ ORDINAL_POSITION の順に返す
		indexColumns := &fakesql.Rows{Columns: []string{"INDEX_NAME", "INDEX_TYPE", "COLUMN_NAME", "COLUMN_ORDERING", "ORDINAL_POSITION"}}
		indexDDLs := make([]string, 0, len(specs))
		for i, spec := range specs {
			name := fmt.Sprintf("Idx%d", i)
			numColumns := int(spec)%maxIndexColumns + 1
			firstColumn := int(spec) / maxIndexColumns % numIndexableColumns
			unique := spec/(maxIndexColumns*numIndexableColumns)%2 == 1

			indexNames.Values = append(indexNames.Values, []driver.Value{name, unique})
			columnNames := make([]string, 0, numColumns)
			for j := range numColumns {
				columnName := fmt.Sprintf("Col%d", (firstColumn+j)%numIndexableColumns)
				indexColumns.Values = append(indexColumns.Values, []driver.Value{name, "INDEX", columnName, "ASC", int64(j + 1)})
				columnNames = append(columnNames, columnName)
			}
			createIndex := "CREATE INDEX "
			if unique {
				createIndex = "CREATE UNIQUE INDEX "
			}
			indexDDLs = append(indexDDLs, createIndex+name+" ON T ("+strings.Join(columnNames, ", ")+");\n")
		}
		expected := "CREATE TABLE T (\n" + strings.Join(columnDefs, ",\n") + "\n) PRIMARY KEY (Id);\n" + strings.Join(indexDDLs, "")

		fixture := showCreateAllTablesFixture{"T": {
			queryShowCreateAllTables:    columns,
			queryShowTableColumnOptions: {Columns: []string{"COLUMN_NAME", "OPTION_NAME", "OPTION_VALUE"}},
			queryShowPrimaryKey: {Columns: []string{"INDEX_NAME", "INDEX_TYPE", "COLUMN_NAME", "COLUMN_ORDERING", "ORDINAL_POSITION"}, Values: [][]driver.Value{
				{"PRIMARY_KEY", "PRIMARY_KEY", "Id", "ASC", int64(1)},
			}},
			querySelectTableOptionRowDeletionPolicy: {Columns: []string{"TABLE_NAME", "ROW_DELETION_POLICY_EXPRESSION"}},
			querySelectIndexes:                      indexNames,
			queryShowIndexes:                        indexColumns,
		}}
		db := fakesql.OpenDB(fixture.queryFunc(t, []string{"T"}, "", nil))
		defer func() { _ = db.Close() }()

		actual, err := ShowCreateAllTables(context.Background(), db)
		if err != nil {
			t.Logf("ShowCreateAllTables: %v", err)
			return false
		}
		before, err := ddlspanner.NewParser(ddlspanner.NewLexer(actual)).Parse()
		if err != nil {
			t.Logf("parse ShowCreateAllTables output: %v\n%s", err, actual)
			return false
		}
		after, err := ddlspanner.NewParser(ddlspanner.NewLexer(expected)).Parse()
		if err != nil {
			t.Logf("parse expected DDL: %v\n%s", err, expected)
			return false
		}
		if _, err := ddlspanner.Diff(before, after); !errors.Is(err, ddl.ErrNoDifference) {
			t.Logf("unexpected difference: err=%v\n--- ShowCreateAllTables\n%s--- expected\n%s", err, actual, expected)
			return false
		}
		return true
	}
	require.NoError(t, quick.Check(property, nil))
}
