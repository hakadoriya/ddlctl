package mysql

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
	require "github.com/hakadoriya/z.go/testingz/requirez"

	"github.com/hakadoriya/ddlctl/pkg/internal/fakesql"
)

// newShowCreateAllTablesQueryFunc は tableNamesQuery に対してテーブル users / groups を返し、
// 各テーブルの SHOW CREATE TABLE に応答する QueryFunc を返す。
// failQuery に一致するクエリには errQuery を返す。
func newShowCreateAllTablesQueryFunc(t *testing.T, tableNamesQuery, failQuery string, errQuery error) fakesql.QueryFunc {
	t.Helper()

	return func(query string, _ []driver.NamedValue) (*fakesql.Rows, error) {
		if query == failQuery {
			return nil, errQuery
		}
		switch query {
		case tableNamesQuery:
			return &fakesql.Rows{
				Columns: []string{"TABLE_NAME"},
				Values:  [][]driver.Value{{"users"}, {"groups"}},
			}, nil
		case "SHOW CREATE TABLE `users`":
			return &fakesql.Rows{
				Columns: []string{"Table", "Create Table"},
				Values:  [][]driver.Value{{"users", "CREATE TABLE `users` (\n  `id` varchar(36) NOT NULL,\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB"}},
			}, nil
		case "SHOW CREATE TABLE `groups`":
			return &fakesql.Rows{
				Columns: []string{"Table", "Create Table"},
				Values:  [][]driver.Value{{"groups", "CREATE TABLE `groups` (\n  `id` varchar(36) NOT NULL,\n  PRIMARY KEY (`id`)\n) ENGINE=InnoDB"}},
			}, nil
		default:
			t.Errorf("unexpected query: %s", query)
			return nil, fakesql.ErrNoRowsPrepared
		}
	}
}

func TestShowCreateAllTables(t *testing.T) {
	t.Parallel()

	const expected = "CREATE TABLE `users` (\n" +
		"  `id` varchar(36) NOT NULL,\n" +
		"  PRIMARY KEY (`id`)\n" +
		") ENGINE=InnoDB;\n" +
		"CREATE TABLE `groups` (\n" +
		"  `id` varchar(36) NOT NULL,\n" +
		"  PRIMARY KEY (`id`)\n" +
		") ENGINE=InnoDB;\n"

	t.Run("success,CurrentDatabase", func(t *testing.T) {
		t.Parallel()

		const tableNamesQuery = "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = database()"
		db := fakesql.OpenDB(newShowCreateAllTablesQueryFunc(t, tableNamesQuery, "", nil))
		t.Cleanup(func() { _ = db.Close() })

		actual, err := ShowCreateAllTables(context.Background(), db)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("success,WithShowCreateAllTablesOptionSchema", func(t *testing.T) {
		t.Parallel()

		const tableNamesQuery = "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = `mydb`"
		db := fakesql.OpenDB(newShowCreateAllTablesQueryFunc(t, tableNamesQuery, "", nil))
		t.Cleanup(func() { _ = db.Close() })

		actual, err := ShowCreateAllTables(context.Background(), db, WithShowCreateAllTablesOptionSchema("mydb"))
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	const tableNamesQuery = "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = database()"
	for _, failQuery := range []string{
		tableNamesQuery,
		"SHOW CREATE TABLE `groups`",
	} {
		t.Run("failure,"+failQuery, func(t *testing.T) {
			t.Parallel()

			errQuery := errors.New("query failed")
			db := fakesql.OpenDB(newShowCreateAllTablesQueryFunc(t, tableNamesQuery, failQuery, errQuery))
			t.Cleanup(func() { _ = db.Close() })

			_, err := ShowCreateAllTables(context.Background(), db)
			require.ErrorIs(t, err, errQuery)
		})
	}
}
