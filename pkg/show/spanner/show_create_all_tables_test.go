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

	// 主キーの各カラムの並び順 (ASC / DESC) を生成し、ShowCreateAllTables の出力をパースした結果が
	// 同じ並び順で書いた DDL と差分なしになることを検証する。
	// 差分があると ddlctl diff / apply が毎回 DROP TABLE + CREATE TABLE を生成し、テーブルのデータが消える。
	property := func(descs []bool) bool {
		if len(descs) == 0 {
			// 主キーのないテーブルは対象外
			return true
		}
		if len(descs) > maxPrimaryKeyColumns {
			descs = descs[:maxPrimaryKeyColumns]
		}

		columns := &fakesql.Rows{Columns: []string{"TABLE_NAME", "COLUMN_NAME", "COLUMN_DEFAULT", "IS_NULLABLE", "SPANNER_TYPE"}}
		primaryKey := &fakesql.Rows{Columns: []string{"INDEX_NAME", "INDEX_TYPE", "COLUMN_NAME", "COLUMN_ORDERING", "ORDINAL_POSITION"}}
		columnDefs := make([]string, 0, len(descs))
		keyParts := make([]string, 0, len(descs))
		for i, desc := range descs {
			name := fmt.Sprintf("Key%d", i)
			ordering, keyPart := "ASC", name
			if desc {
				ordering, keyPart = "DESC", name+" DESC"
			}
			columns.Values = append(columns.Values, []driver.Value{"T", name, nil, "NO", "STRING(MAX)"})
			primaryKey.Values = append(primaryKey.Values, []driver.Value{"PRIMARY_KEY", "PRIMARY_KEY", name, ordering, int64(i + 1)})
			columnDefs = append(columnDefs, "    "+name+" STRING(MAX) NOT NULL")
			keyParts = append(keyParts, keyPart)
		}
		expected := "CREATE TABLE T (\n" + strings.Join(columnDefs, ",\n") + "\n) PRIMARY KEY (" + strings.Join(keyParts, ", ") + ");\n"

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
		return true
	}
	require.NoError(t, quick.Check(property, nil))
}
