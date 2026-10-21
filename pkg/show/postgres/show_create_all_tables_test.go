package postgres

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"testing"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
	require "github.com/hakadoriya/z.go/testingz/requirez"

	"github.com/hakadoriya/ddlctl/pkg/internal/fakesql"
)

// newShowCreateAllTablesQueryFunc は schema に対する CREATE TABLE / CREATE INDEX の問い合わせに応答する QueryFunc を返す。
// failQuery に一致するクエリには errQuery を返す。
func newShowCreateAllTablesQueryFunc(t *testing.T, schema, failQuery string, errQuery error) fakesql.QueryFunc {
	t.Helper()

	return func(query string, _ []driver.NamedValue) (*fakesql.Rows, error) {
		if query == failQuery {
			return nil, errQuery
		}
		switch query {
		case fmt.Sprintf(formatShowCreateAllTables, schema):
			return &fakesql.Rows{
				Columns: []string{"create_statement"},
				Values: [][]driver.Value{
					{"CREATE TABLE users (\n  id text NOT NULL,\n  name text\n);"},
					{"CREATE TABLE groups (\n  id text NOT NULL\n);"},
				},
			}, nil
		case fmt.Sprintf(formatShowCreateAllIndexes, schema, schema):
			return &fakesql.Rows{
				Columns: []string{"create_statement"},
				Values: [][]driver.Value{
					{"CREATE INDEX users_idx_by_name ON public.users USING btree (name)"},
					{"CREATE UNIQUE INDEX groups_idx_by_id ON public.groups USING btree (id)"},
				},
			}, nil
		default:
			t.Errorf("unexpected query: %s", query)
			return nil, fakesql.ErrNoRowsPrepared
		}
	}
}

func TestShowCreateAllTables(t *testing.T) {
	t.Parallel()

	const expected = `CREATE TABLE users (
  id text NOT NULL,
  name text
);
CREATE TABLE groups (
  id text NOT NULL
);
CREATE INDEX users_idx_by_name ON public.users USING btree (name);
CREATE UNIQUE INDEX groups_idx_by_id ON public.groups USING btree (id);
`

	t.Run("success,DefaultSchema", func(t *testing.T) {
		t.Parallel()

		db := fakesql.OpenDB(newShowCreateAllTablesQueryFunc(t, "public", "", nil))
		t.Cleanup(func() { _ = db.Close() })

		actual, err := ShowCreateAllTables(context.Background(), db)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("success,WithShowCreateAllTablesOptionSchema", func(t *testing.T) {
		t.Parallel()

		db := fakesql.OpenDB(newShowCreateAllTablesQueryFunc(t, "myschema", "", nil))
		t.Cleanup(func() { _ = db.Close() })

		actual, err := ShowCreateAllTables(context.Background(), db, WithShowCreateAllTablesOptionSchema("myschema"))
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	for name, failQuery := range map[string]string{
		"CreateTable": fmt.Sprintf(formatShowCreateAllTables, "public"),
		"CreateIndex": fmt.Sprintf(formatShowCreateAllIndexes, "public", "public"),
	} {
		t.Run("failure,"+name, func(t *testing.T) {
			t.Parallel()

			errQuery := errors.New("query failed")
			db := fakesql.OpenDB(newShowCreateAllTablesQueryFunc(t, "public", failQuery, errQuery))
			t.Cleanup(func() { _ = db.Close() })

			_, err := ShowCreateAllTables(context.Background(), db)
			require.ErrorIs(t, err, errQuery)
		})
	}
}
