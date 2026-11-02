package spanner

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"slices"
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

	// インターリーブしていないテーブルとして返す
	tables := &fakesql.Rows{Columns: tablesColumns}
	for _, tableName := range tableNames {
		tables.Values = append(tables.Values, []driver.Value{tableName, nil, nil, nil})
	}
	return f.queryFuncWithTables(t, tables, failQuery, errQuery)
}

// tablesColumns は querySelectTableName の結果のカラム
var tablesColumns = []string{"TABLE_NAME", "PARENT_TABLE_NAME", "ON_DELETE_ACTION", "INTERLEAVE_TYPE"}

// queryFuncWithTables は querySelectTableName に tables を返し、それ以外のクエリに fixture で応答する QueryFunc を返す。
// failQuery に一致するクエリには errQuery を返す。
func (f showCreateAllTablesFixture) queryFuncWithTables(t *testing.T, tables *fakesql.Rows, failQuery string, errQuery error) fakesql.QueryFunc {
	t.Helper()

	return func(query string, args []driver.NamedValue) (*fakesql.Rows, error) {
		if query == failQuery {
			return nil, errQuery
		}
		if query == querySelectTableName {
			return tables, nil
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

	// 定義ファイル側の並び順の書き方。ASC は省略・明示・小文字、DESC は大文字・小文字・先頭だけ大文字にゆらす
	ascSpellings := []string{"", " ASC", " asc"}
	descSpellings := []string{" DESC", " desc", " Desc"}

	// テーブルに張るインデックスの組 (インデックス数と、各インデックスのカラム数・先頭カラム・ UNIQUE の有無・各カラムの並び順と定義ファイル側の書き方) を生成し、
	// ShowCreateAllTables の出力をパースした結果が、同じインデックスを書いた DDL と差分なしになることを検証する。
	// あわせて、インデックスがあれば、そのうち 1 つについて、カラムを 1 つ別のカラムに変えた DDL とも、1 カラムだけ並び順を反転した DDL とも差分が出ることを検証する。
	// 差分なしにならないと ddlctl diff / apply が毎回 DROP INDEX + CREATE INDEX を生成する。
	// 差分が出ないと、インデックスのカラムや並び順の変更を見落とす。
	// NOTE: インデックス数を specs の長さで決めると、testing/quick はスライスの長さを 0 から 49 で一様に選ぶため、ほとんどが上限に切り詰められる。
	// インデックス数は別の引数で 0 から maxIndexes まで一様に選ぶ。
	property := func(n uint8, allSpecs [maxIndexes]uint16, allOrderings [maxIndexes][maxIndexColumns]uint8, changed, flip uint8) bool {
		specs := allSpecs[:int(n)%(maxIndexes+1)]

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
		changedIndexDDLs := make([]string, 0, len(specs))
		flippedIndexDDLs := make([]string, 0, len(specs))
		changedIndex := -1
		if len(specs) > 0 {
			changedIndex = int(changed) % len(specs)
		}
		for i, spec := range specs {
			name := fmt.Sprintf("Idx%d", i)
			numColumns := int(spec)%maxIndexColumns + 1
			firstColumn := int(spec) / maxIndexColumns % numIndexableColumns
			unique := spec/(maxIndexColumns*numIndexableColumns)%2 == 1

			indexNames.Values = append(indexNames.Values, []driver.Value{name, unique})
			// 各カラムの名前と、定義ファイル側の並び順の書き方、およびその並び順を反転した書き方
			columnNames := make([]string, 0, numColumns)
			orderings := make([]string, 0, numColumns)
			flippedOrderings := make([]string, 0, numColumns)
			for j := range numColumns {
				columnName := fmt.Sprintf("Col%d", (firstColumn+j)%numIndexableColumns)
				desc := allOrderings[i][j]%2 == 1
				spelling := int(allOrderings[i][j]/2) % len(ascSpellings)
				columnOrdering, ordering, flippedOrdering := "ASC", ascSpellings[spelling], descSpellings[spelling]
				if desc {
					columnOrdering, ordering, flippedOrdering = "DESC", descSpellings[spelling], ascSpellings[spelling]
				}
				indexColumns.Values = append(indexColumns.Values, []driver.Value{name, "INDEX", columnName, columnOrdering, int64(j + 1)})
				columnNames = append(columnNames, columnName)
				orderings = append(orderings, ordering)
				flippedOrderings = append(flippedOrderings, flippedOrdering)
			}
			createIndexOf := func(columnNames, orderings []string) string {
				createIndex := "CREATE INDEX "
				if unique {
					createIndex = "CREATE UNIQUE INDEX "
				}
				parts := make([]string, 0, len(columnNames))
				for j := range columnNames {
					parts = append(parts, columnNames[j]+orderings[j])
				}
				return createIndex + name + " ON T (" + strings.Join(parts, ", ") + ");\n"
			}
			indexDDL := createIndexOf(columnNames, orderings)
			indexDDLs = append(indexDDLs, indexDDL)
			if i != changedIndex {
				changedIndexDDLs = append(changedIndexDDLs, indexDDL)
				flippedIndexDDLs = append(flippedIndexDDLs, indexDDL)
				continue
			}
			// 先頭のカラムを、このインデックスに含まれないカラム (numColumns < numIndexableColumns なので必ずある) に変える
			changedColumnNames := slices.Clone(columnNames)
			changedColumnNames[0] = fmt.Sprintf("Col%d", (firstColumn+numColumns)%numIndexableColumns)
			changedIndexDDLs = append(changedIndexDDLs, createIndexOf(changedColumnNames, orderings))
			// 1 カラムだけ並び順を反転する
			flipped := int(flip) % numColumns
			flippedColumnOrderings := slices.Clone(orderings)
			flippedColumnOrderings[flipped] = flippedOrderings[flipped]
			flippedIndexDDLs = append(flippedIndexDDLs, createIndexOf(columnNames, flippedColumnOrderings))
		}
		tableDDL := "CREATE TABLE T (\n" + strings.Join(columnDefs, ",\n") + "\n) PRIMARY KEY (Id);\n"
		expected := tableDDL + strings.Join(indexDDLs, "")
		changedDDL := tableDDL + strings.Join(changedIndexDDLs, "")
		flippedDDL := tableDDL + strings.Join(flippedIndexDDLs, "")

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
		if changedIndex < 0 {
			return true
		}
		for _, variant := range []struct{ name, ddl string }{{"changed", changedDDL}, {"flipped", flippedDDL}} {
			variantAfter, err := ddlspanner.NewParser(ddlspanner.NewLexer(variant.ddl)).Parse()
			if err != nil {
				t.Logf("parse %s DDL: %v\n%s", variant.name, err, variant.ddl)
				return false
			}
			if _, err := ddlspanner.Diff(before, variantAfter); err != nil {
				t.Logf("expected a difference: err=%v\n--- ShowCreateAllTables\n%s--- %s\n%s", err, actual, variant.name, variant.ddl)
				return false
			}
		}
		return true
	}
	require.NoError(t, quick.Check(property, nil))
}

func TestShowCreateAllTables_InterleaveInParentRoundTrip(t *testing.T) {
	t.Parallel()

	// 生成するテーブル数の上限 (テストの規模を抑えるため。Cloud Spanner の上限ではない)
	const maxTables = 8

	// 定義ファイル側の ON DELETE の書き方。NO ACTION は省略・大文字・小文字、CASCADE は大文字・小文字にゆらす
	noActionSpellings := []string{"", " ON DELETE NO ACTION", " on delete no action"}
	cascadeSpellings := []string{" ON DELETE CASCADE", " on delete cascade"}

	// 各テーブルのインターリーブの種類 (なし / INTERLEAVE IN PARENT / INTERLEAVE IN) ・親テーブル・ ON DELETE の動作と定義ファイル側の書き方を生成し、
	// ShowCreateAllTables の出力をパースした結果が、同じインターリーブを書いた DDL と差分なしになることを検証する。
	// あわせて、INTERLEAVE IN PARENT のテーブルがあれば、そのうち 1 つの ON DELETE の動作を反転した DDL とは差分が出ることを検証する。
	// 差分なしにならないと ddlctl diff / apply が毎回 DROP TABLE + CREATE TABLE を生成し、テーブルのデータが消える。
	// 差分が出ないと、ON DELETE の動作の変更を見落とす。
	// INTERLEAVE IN (PARENT なし) は pkg/ddl/spanner が未対応で定義ファイルに書けないため、ShowCreateAllTables も出力しないことを期待する。
	property := func(specs []uint16, flip uint8) bool {
		if len(specs) == 0 {
			return true
		}
		if len(specs) > maxTables {
			specs = specs[:maxTables]
		}

		tables := &fakesql.Rows{Columns: tablesColumns}
		fixture := showCreateAllTablesFixture{}
		// 各テーブルの INTERLEAVE IN PARENT 句と、ON DELETE の動作を反転した句
		clauses := make([]string, len(specs))
		flippedClauses := make([]string, len(specs))
		interleavedInParent := make([]int, 0, len(specs))
		for i, spec := range specs {
			name := fmt.Sprintf("T%d", i)
			kind := int(spec) % 3
			if i == 0 {
				// 先頭のテーブルには親にできるテーブルがない
				kind = 0
			}
			parent := fmt.Sprintf("T%d", int(spec)/3%max(i, 1))
			cascade := spec/(3*maxTables)%2 == 1
			spelling := int(spec) / (3 * maxTables * 2) % (len(noActionSpellings) * len(cascadeSpellings))
			// 親を持たないテーブルの INTERLEAVE_TYPE は、Cloud Spanner のドキュメントでは空文字列、Cloud Spanner エミュレータでは "IN PARENT"。
			// NULL でも INTERLEAVE IN PARENT を出力しないことをあわせて検証する
			rootInterleaveType := []driver.Value{"", "IN PARENT", nil}[int(spec)/(3*maxTables*2*6)%3]

			switch kind {
			case 0:
				tables.Values = append(tables.Values, []driver.Value{name, nil, nil, rootInterleaveType})
			case 1:
				noAction, cascadeClause := noActionSpellings[spelling%len(noActionSpellings)], cascadeSpellings[spelling%len(cascadeSpellings)]
				action, clause, flippedClause := "NO ACTION", noAction, cascadeClause
				if cascade {
					action, clause, flippedClause = "CASCADE", cascadeClause, noAction
				}
				tables.Values = append(tables.Values, []driver.Value{name, parent, action, "IN PARENT"})
				clauses[i] = ",\nINTERLEAVE IN PARENT " + parent + clause
				flippedClauses[i] = ",\nINTERLEAVE IN PARENT " + parent + flippedClause
				interleavedInParent = append(interleavedInParent, i)
			default:
				tables.Values = append(tables.Values, []driver.Value{name, parent, "NO ACTION", "IN"})
			}

			fixture[name] = map[string]*fakesql.Rows{
				queryShowCreateAllTables: {Columns: []string{"TABLE_NAME", "COLUMN_NAME", "COLUMN_DEFAULT", "IS_NULLABLE", "SPANNER_TYPE"}, Values: [][]driver.Value{
					{name, "Id", nil, "NO", "STRING(MAX)"},
				}},
				queryShowTableColumnOptions: {Columns: []string{"COLUMN_NAME", "OPTION_NAME", "OPTION_VALUE"}},
				queryShowPrimaryKey: {Columns: []string{"INDEX_NAME", "INDEX_TYPE", "COLUMN_NAME", "COLUMN_ORDERING", "ORDINAL_POSITION"}, Values: [][]driver.Value{
					{"PRIMARY_KEY", "PRIMARY_KEY", "Id", "ASC", int64(1)},
				}},
				querySelectTableOptionRowDeletionPolicy: {Columns: []string{"TABLE_NAME", "ROW_DELETION_POLICY_EXPRESSION"}},
				querySelectIndexes:                      {Columns: []string{"INDEX_NAME", "IS_UNIQUE"}},
			}
		}
		// flipped 番目のテーブルだけ ON DELETE の動作を反転した DDL を返す。flipped が -1 なら反転しない
		ddlOf := func(flipped int) string {
			ddls := make([]string, 0, len(specs))
			for i := range specs {
				clause := clauses[i]
				if i == flipped {
					clause = flippedClauses[i]
				}
				ddls = append(ddls, fmt.Sprintf("CREATE TABLE T%d (\n    Id STRING(MAX) NOT NULL\n) PRIMARY KEY (Id)%s;\n", i, clause))
			}
			return strings.Join(ddls, "\n")
		}
		expected := ddlOf(-1)

		db := fakesql.OpenDB(fixture.queryFuncWithTables(t, tables, "", nil))
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
		if len(interleavedInParent) == 0 {
			return true
		}
		flippedDDL := ddlOf(interleavedInParent[int(flip)%len(interleavedInParent)])
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

// TestShowCreateAllTables_InterleaveInParent は INTERLEAVE IN PARENT 句の出力形式とテーブルの出力順を完全一致で検証する。
// TestShowCreateAllTables_InterleaveInParentRoundTrip は出力をパースして比べるので、パーサーが受け付ける範囲の違い
// (句の順序・ ON DELETE NO ACTION の明示・テーブルの順序) を検出できない。
func TestShowCreateAllTables_InterleaveInParent(t *testing.T) {
	t.Parallel()

	columnColumns := []string{"TABLE_NAME", "COLUMN_NAME", "COLUMN_DEFAULT", "IS_NULLABLE", "SPANNER_TYPE"}
	primaryKeyColumns := []string{"INDEX_NAME", "INDEX_TYPE", "COLUMN_NAME", "COLUMN_ORDERING", "ORDINAL_POSITION"}
	// tableOf は UserId と keyColumn を主キーに持つテーブルの fixture を返す。keyColumn が空なら UserId だけを主キーにする
	tableOf := func(name, keyColumn string, rowDeletionPolicy ...driver.Value) map[string]*fakesql.Rows {
		r := map[string]*fakesql.Rows{
			queryShowCreateAllTables: {Columns: columnColumns, Values: [][]driver.Value{
				{name, "UserId", nil, "NO", "STRING(36)"},
			}},
			queryShowTableColumnOptions: {Columns: []string{"COLUMN_NAME", "OPTION_NAME", "OPTION_VALUE"}},
			queryShowPrimaryKey: {Columns: primaryKeyColumns, Values: [][]driver.Value{
				{"PRIMARY_KEY", "PRIMARY_KEY", "UserId", "ASC", int64(1)},
			}},
			querySelectTableOptionRowDeletionPolicy: {Columns: []string{"TABLE_NAME", "ROW_DELETION_POLICY_EXPRESSION"}},
			querySelectIndexes:                      {Columns: []string{"INDEX_NAME", "IS_UNIQUE"}},
		}
		if keyColumn != "" {
			r[queryShowCreateAllTables].Values = append(r[queryShowCreateAllTables].Values, []driver.Value{name, keyColumn, nil, "NO", "TIMESTAMP"})
			r[queryShowPrimaryKey].Values = append(r[queryShowPrimaryKey].Values, []driver.Value{"PRIMARY_KEY", "PRIMARY_KEY", keyColumn, "ASC", int64(2)})
		}
		if len(rowDeletionPolicy) > 0 {
			r[querySelectTableOptionRowDeletionPolicy].Values = [][]driver.Value{append([]driver.Value{name}, rowDeletionPolicy...)}
		}
		return r
	}
	fixture := showCreateAllTablesFixture{
		"Users": tableOf("Users", ""),
		// ON DELETE CASCADE と ROW DELETION POLICY を持つ子テーブル
		"Logs": tableOf("Logs", "LoggedAt", "OLDER_THAN(LoggedAt, INTERVAL 30 DAY)"),
		// ON DELETE を省略した (NO ACTION の) 子テーブル
		"Sessions": tableOf("Sessions", "StartedAt"),
		// PARENT なしの INTERLEAVE IN でインターリーブしたテーブル
		"Archives": tableOf("Archives", "ArchivedAt"),
	}
	// INFORMATION_SCHEMA.TABLES は子テーブルを親テーブルより先に返すことがある (Cloud Spanner エミュレータで確認)
	tables := &fakesql.Rows{Columns: tablesColumns, Values: [][]driver.Value{
		{"Logs", "Users", "CASCADE", "IN PARENT"},
		{"Sessions", "Users", "NO ACTION", "IN PARENT"},
		{"Users", nil, nil, "IN PARENT"},
		{"Archives", "Users", "NO ACTION", "IN"},
	}}

	db := fakesql.OpenDB(fixture.queryFuncWithTables(t, tables, "", nil))
	t.Cleanup(func() { _ = db.Close() })

	// INTERLEAVE IN PARENT は PRIMARY KEY と ROW DELETION POLICY の間に置く (Cloud Spanner は逆順の DDL を構文エラーにする)
	const expected = `CREATE TABLE Users (
    UserId STRING(36) NOT NULL
) PRIMARY KEY (UserId);

CREATE TABLE Logs (
    UserId STRING(36) NOT NULL,
    LoggedAt TIMESTAMP NOT NULL
) PRIMARY KEY (UserId, LoggedAt),
INTERLEAVE IN PARENT Users ON DELETE CASCADE,
ROW DELETION POLICY (OLDER_THAN(LoggedAt, INTERVAL 30 DAY));

CREATE TABLE Sessions (
    UserId STRING(36) NOT NULL,
    StartedAt TIMESTAMP NOT NULL
) PRIMARY KEY (UserId, StartedAt),
INTERLEAVE IN PARENT Users;

CREATE TABLE Archives (
    UserId STRING(36) NOT NULL,
    ArchivedAt TIMESTAMP NOT NULL
) PRIMARY KEY (UserId, ArchivedAt);
`
	actual, err := ShowCreateAllTables(context.Background(), db)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestSortParentsFirst(t *testing.T) {
	t.Parallel()

	// 生成するテーブル数の上限 (テストの規模を抑えるため。Cloud Spanner の上限ではない)
	const maxTables = 16

	// 各テーブルの親 (なし / 自分より前に作ったテーブル / 一覧にないテーブル) と、INFORMATION_SCHEMA.TABLES が返す順序を生成し、
	// sortParentsFirst の結果が入力の並べ替えであり、どの子テーブルも一覧にある親テーブルより後ろに来ることを検証する。
	// 親テーブルが子テーブルより後ろにあると、show の出力を空のデータベースに apply したときに、
	// 子テーブルの CREATE TABLE が親テーブルを見つけられずに失敗する。
	property := func(parents []uint8, order []uint8) bool {
		if len(parents) > maxTables {
			parents = parents[:maxTables]
		}

		input := make([]*informationSchemaTable, 0, len(parents))
		for i, p := range parents {
			tbl := &informationSchemaTable{TableName: fmt.Sprintf("T%d", i)}
			switch {
			case p%4 == 3:
				missing := "Missing"
				tbl.ParentTableName = &missing
			case i > 0 && p%2 == 1:
				parent := fmt.Sprintf("T%d", int(p/4)%i)
				tbl.ParentTableName = &parent
			}
			input = append(input, tbl)
		}
		// INFORMATION_SCHEMA.TABLES が返す順序として、order で作成順をシャッフルする
		for i := len(input) - 1; i > 0 && len(order) > 0; i-- {
			j := int(order[i%len(order)]) % (i + 1)
			input[i], input[j] = input[j], input[i]
		}

		sorted := sortParentsFirst(input)

		if len(sorted) != len(input) {
			t.Logf("len(sorted)=%d, len(input)=%d", len(sorted), len(input))
			return false
		}
		position := make(map[string]int, len(sorted))
		for i, tbl := range sorted {
			if _, duplicated := position[tbl.TableName]; duplicated {
				t.Logf("duplicated table: %s", tbl.TableName)
				return false
			}
			position[tbl.TableName] = i
		}
		for _, tbl := range input {
			if _, ok := position[tbl.TableName]; !ok {
				t.Logf("missing table: %s", tbl.TableName)
				return false
			}
		}
		for _, tbl := range sorted {
			if tbl.ParentTableName == nil {
				continue
			}
			if parentPosition, ok := position[*tbl.ParentTableName]; ok && parentPosition > position[tbl.TableName] {
				t.Logf("parent %s comes after child %s", *tbl.ParentTableName, tbl.TableName)
				return false
			}
		}
		return true
	}
	require.NoError(t, quick.Check(property, nil))
}
