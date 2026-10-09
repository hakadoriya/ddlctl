package cockroachdb

import (
	"context"
	"database/sql/driver"
	"errors"
	"testing"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
	require "github.com/hakadoriya/z.go/testingz/requirez"

	"github.com/hakadoriya/ddlctl/pkg/internal/fakesql"
)

func TestShowCreateAllTables(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		db := fakesql.OpenDB(func(query string, _ []driver.NamedValue) (*fakesql.Rows, error) {
			assert.Equal(t, queryShowCreateAllTables, query)
			return &fakesql.Rows{
				Columns: []string{"create_statement"},
				Values: [][]driver.Value{
					{"CREATE TABLE public.users (\n\tid STRING NOT NULL,\n\tCONSTRAINT users_pkey PRIMARY KEY (id ASC)\n);"},
					{"CREATE TABLE public.groups (\n\tid STRING NOT NULL,\n\tCONSTRAINT groups_pkey PRIMARY KEY (id ASC)\n);"},
				},
			}, nil
		})
		t.Cleanup(func() { _ = db.Close() })

		const expected = `CREATE TABLE public.users (
	id STRING NOT NULL,
	CONSTRAINT users_pkey PRIMARY KEY (id ASC)
);
CREATE TABLE public.groups (
	id STRING NOT NULL,
	CONSTRAINT groups_pkey PRIMARY KEY (id ASC)
);
`
		actual, err := ShowCreateAllTables(context.Background(), db)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("success,NoTables", func(t *testing.T) {
		t.Parallel()

		db := fakesql.OpenDB(func(string, []driver.NamedValue) (*fakesql.Rows, error) {
			return &fakesql.Rows{Columns: []string{"create_statement"}}, nil
		})
		t.Cleanup(func() { _ = db.Close() })

		actual, err := ShowCreateAllTables(context.Background(), db)
		require.NoError(t, err)
		assert.Equal(t, "", actual)
	})

	t.Run("failure,QueryContext", func(t *testing.T) {
		t.Parallel()

		errQuery := errors.New("query failed")
		db := fakesql.OpenDB(func(string, []driver.NamedValue) (*fakesql.Rows, error) {
			return nil, errQuery
		})
		t.Cleanup(func() { _ = db.Close() })

		_, err := ShowCreateAllTables(context.Background(), db)
		require.ErrorIs(t, err, errQuery)
	})
}
