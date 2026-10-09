package fakesql_test

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
	require "github.com/hakadoriya/z.go/testingz/requirez"

	"github.com/hakadoriya/ddlctl/pkg/internal/fakesql"
)

func TestOpenDB(t *testing.T) {
	t.Parallel()

	t.Run("success,QueryContext", func(t *testing.T) {
		t.Parallel()

		db := fakesql.OpenDB(func(query string, args []driver.NamedValue) (*fakesql.Rows, error) {
			// クエリと引数がそのまま QueryFunc に渡ること
			assert.Equal(t, "SELECT name, age FROM users WHERE id = ?", query)
			require.Equal(t, 1, len(args))
			assert.Equal(t, driver.Value("u1"), args[0].Value)
			return &fakesql.Rows{
				Columns: []string{"name", "age"},
				Values:  [][]driver.Value{{"alice", int64(20)}, {"bob", int64(30)}},
			}, nil
		})
		t.Cleanup(func() { _ = db.Close() })

		rows, err := db.QueryContext(context.Background(), "SELECT name, age FROM users WHERE id = ?", "u1")
		require.NoError(t, err)
		t.Cleanup(func() { _ = rows.Close() })

		columns, err := rows.Columns()
		require.NoError(t, err)
		assert.Equal(t, []string{"name", "age"}, columns)

		type user struct {
			name string
			age  int
		}
		var actual []user
		for rows.Next() {
			var u user
			require.NoError(t, rows.Scan(&u.name, &u.age))
			actual = append(actual, u)
		}
		require.NoError(t, rows.Err())
		assert.Equal(t, []user{{name: "alice", age: 20}, {name: "bob", age: 30}}, actual)
	})

	t.Run("failure,QueryFuncError", func(t *testing.T) {
		t.Parallel()

		errQuery := errors.New("query failed")
		db := fakesql.OpenDB(func(string, []driver.NamedValue) (*fakesql.Rows, error) {
			return nil, errQuery
		})
		t.Cleanup(func() { _ = db.Close() })

		_, err := db.QueryContext(context.Background(), "SELECT 1")
		require.ErrorIs(t, err, errQuery)
	})

	t.Run("failure,ErrNoRowsPrepared", func(t *testing.T) {
		t.Parallel()

		db := fakesql.OpenDB(func(string, []driver.NamedValue) (*fakesql.Rows, error) {
			return nil, nil //nolint:nilnil // 結果を用意し忘れた QueryFunc を再現する
		})
		t.Cleanup(func() { _ = db.Close() })

		_, err := db.QueryContext(context.Background(), "SELECT 1")
		require.ErrorIs(t, err, fakesql.ErrNoRowsPrepared)
	})

	t.Run("failure,ErrNotSupported", func(t *testing.T) {
		t.Parallel()

		db := fakesql.OpenDB(func(string, []driver.NamedValue) (*fakesql.Rows, error) {
			return &fakesql.Rows{}, nil
		})
		t.Cleanup(func() { _ = db.Close() })

		// QueryContext 以外の経路 (Prepare / Begin) は対応していないこと
		_, err := db.ExecContext(context.Background(), "DELETE FROM users")
		require.ErrorIs(t, err, fakesql.ErrNotSupported)
		_, err = db.BeginTx(context.Background(), nil)
		require.ErrorIs(t, err, fakesql.ErrNotSupported)
	})
}
