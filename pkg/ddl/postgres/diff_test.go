package postgres

import (
	"fmt"
	"testing"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
	require "github.com/hakadoriya/z.go/testingz/requirez"

	"github.com/hakadoriya/ddlctl/pkg/ddl"
)

func TestDiff(t *testing.T) {
	t.Parallel()

	t.Run("failure,ddl.ErrNoDifference", func(t *testing.T) {
		t.Parallel()

		before := &DDL{}
		after := &DDL{}
		_, err := Diff(before, after)
		require.ErrorIs(t, err, ddl.ErrNoDifference)
	})

	t.Run("failure,ddl.ErrNotSupported,DropTableStmt", func(t *testing.T) {
		t.Parallel()

		{
			before := &DDL{
				Stmts: []Stmt{
					&DropTableStmt{Name: &ObjectName{Name: &Ident{Name: "table_name", Raw: "table_name"}}},
				},
			}
			after := (*DDL)(nil)
			_, err := Diff(before, after)
			require.ErrorIs(t, err, ddl.ErrNotSupported)
		}
		{
			before := &DDL{
				Stmts: []Stmt{
					&DropTableStmt{Name: &ObjectName{Name: &Ident{Name: "table_name", Raw: "table_name"}}},
				},
			}
			after := &DDL{}
			_, err := Diff(before, after)
			require.ErrorIs(t, err, ddl.ErrNotSupported)
		}
		{
			before := &DDL{}
			after := &DDL{
				Stmts: []Stmt{
					&DropTableStmt{Name: &ObjectName{Name: &Ident{Name: "table_name", Raw: "table_name"}}},
				},
			}
			_, err := Diff(before, after)
			require.ErrorIs(t, err, ddl.ErrNotSupported)
		}
	})

	t.Run("success,after", func(t *testing.T) {
		t.Parallel()

		before := (*DDL)(nil)
		after := &DDL{
			Stmts: []Stmt{
				&CreateTableStmt{
					Name: &ObjectName{Name: &Ident{Name: "table_name", Raw: "table_name"}},
					Columns: []*Column{
						{
							Name: &Ident{Name: "column_name", Raw: "column_name"},
							DataType: &DataType{
								Name: "STRING",
							},
							NotNull: true,
						},
					},
					Constraints: []Constraint{
						&PrimaryKeyConstraint{
							Columns: []*ColumnIdent{
								{
									Ident: &Ident{Name: "column_name", Raw: "column_name"},
								},
							},
						},
					},
				},
			},
		}
		actual, err := Diff(before, after)
		require.NoError(t, err)
		if !assert.Equal(t, after, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", after), fmt.Sprintf("%#v", actual))
		}
		assert.Equal(t, `CREATE TABLE table_name (
    column_name STRING NOT NULL,
    PRIMARY KEY (column_name)
);
`, actual.String())
	})

	t.Run("success,before,nil,Table", func(t *testing.T) {
		t.Parallel()

		before := &DDL{
			Stmts: []Stmt{
				&CreateTableStmt{
					Name: &ObjectName{Schema: &Ident{Name: "public", Raw: "public"}, Name: &Ident{Name: "table_name", Raw: "table_name"}},
					Columns: []*Column{
						{
							Name: &Ident{Name: "column_name", Raw: "column_name"},
						},
					},
				},
			},
		}
		after := (*DDL)(nil)
		actual, err := Diff(before, after)
		require.NoError(t, err)
		expected := &DDL{
			Stmts: []Stmt{
				&DropTableStmt{
					Name: &ObjectName{Schema: &Ident{Name: "public", Raw: "public"}, Name: &Ident{Name: "table_name", Raw: "table_name"}},
				},
			},
		}
		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		assert.Equal(t, `DROP TABLE public.table_name;
`, actual.String())
	})

	t.Run("success,before,Table", func(t *testing.T) {
		t.Parallel()

		before := &DDL{
			Stmts: []Stmt{
				&CreateTableStmt{
					Name: &ObjectName{Name: &Ident{Name: "table_name", Raw: "table_name"}},
					Columns: []*Column{
						{
							Name: &Ident{Name: "column_name", Raw: "column_name"},
						},
					},
				},
			},
		}
		after := &DDL{}
		actual, err := Diff(before, after)
		require.NoError(t, err)
		expected := &DDL{
			Stmts: []Stmt{
				&DropTableStmt{
					Name: &ObjectName{Name: &Ident{Name: "table_name", Raw: "table_name"}},
				},
			},
		}
		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		assert.Equal(t, `DROP TABLE table_name;
`, actual.String())
	})

	t.Run("success,before,nil,Index", func(t *testing.T) {
		t.Parallel()

		before := &DDL{
			Stmts: []Stmt{
				&CreateIndexStmt{
					Name: &Ident{Name: "table_name_idx_column_name", Raw: "table_name_idx_column_name"},
					Columns: []*ColumnIdent{
						{
							Ident: &Ident{Name: "column_name", Raw: "column_name"},
						},
					},
				},
			},
		}
		after := (*DDL)(nil)
		actual, err := Diff(before, after)
		require.NoError(t, err)
		expected := &DDL{
			Stmts: []Stmt{
				&DropIndexStmt{
					Name: &Ident{Name: "table_name_idx_column_name", Raw: "table_name_idx_column_name"},
				},
			},
		}
		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		assert.Equal(t, `DROP INDEX table_name_idx_column_name;
`, actual.String())

		t.Logf("✅: %s: actual: %%#v: \n%#v", t.Name(), actual)
		t.Logf("✅: %s: actual: %%s: \n%s", t.Name(), actual)
	})

	t.Run("success,before,Index", func(t *testing.T) {
		t.Parallel()

		before := &DDL{
			Stmts: []Stmt{
				&CreateIndexStmt{
					Name: &Ident{Name: "table_name_idx_column_name", Raw: "table_name_idx_column_name"},
					Columns: []*ColumnIdent{
						{
							Ident: &Ident{Name: "column_name", Raw: "column_name"},
						},
					},
				},
			},
		}
		after := &DDL{}
		actual, err := Diff(before, after)
		require.NoError(t, err)
		expected := &DDL{
			Stmts: []Stmt{
				&DropIndexStmt{
					Name: &Ident{Name: "table_name_idx_column_name", Raw: "table_name_idx_column_name"},
				},
			},
		}
		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		assert.Equal(t, `DROP INDEX table_name_idx_column_name;
`, actual.String())

		t.Logf("✅: %s: actual: %%#v: \n%#v", t.Name(), actual)
		t.Logf("✅: %s: actual: %%s: \n%s", t.Name(), actual)
	})

	t.Run("success,before,Table", func(t *testing.T) {
		t.Parallel()

		before := &DDL{}
		after := &DDL{
			Stmts: []Stmt{
				&CreateTableStmt{
					Name: &ObjectName{Schema: &Ident{Name: "public", Raw: "public"}, Name: &Ident{Name: "table_name", Raw: "table_name"}},
					Columns: []*Column{
						{
							Name: &Ident{Name: "column_name", Raw: "column_name"},
							DataType: &DataType{
								Name: "STRING",
							},
							NotNull: true,
						},
					},
					Constraints: []Constraint{
						&PrimaryKeyConstraint{
							Columns: []*ColumnIdent{
								{
									Ident: &Ident{Name: "column_name", Raw: "column_name"},
								},
							},
						},
					},
				},
			},
		}
		actual, err := Diff(before, after)
		require.NoError(t, err)
		if !assert.Equal(t, after, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", after), fmt.Sprintf("%#v", actual))
		}
		assert.Equal(t, `CREATE TABLE public.table_name (
    column_name STRING NOT NULL,
    PRIMARY KEY (column_name)
);
`, actual.String())
	})

	t.Run("success,before,Index", func(t *testing.T) {
		t.Parallel()

		before, err := NewParser(NewLexer(``)).Parse()
		require.NoError(t, err)

		after, err := NewParser(NewLexer(`CREATE INDEX table_name_idx_column_name ON table_name (column_name);
`)).Parse()
		require.NoError(t, err)

		expected := `CREATE INDEX table_name_idx_column_name ON table_name (column_name);
`

		actual, err := Diff(before, after)
		require.NoError(t, err)

		if !assert.Equal(t, expected, actual.String()) {
			t.Errorf("❌: %s: stmt: %%#v: \n%#v", t.Name(), actual)
		}
	})

	t.Run("success,before,after,Table", func(t *testing.T) {
		t.Parallel()

		before, err := NewParser(NewLexer(`CREATE TABLE public.groups (
    id UUID NOT NULL,
);
CREATE TABLE public.users (
    user_id UUID NOT NULL,
    username VARCHAR(256) NOT NULL,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    CONSTRAINT users_pkey PRIMARY KEY (user_id),
);
`)).Parse()
		require.NoError(t, err)

		after, err := NewParser(NewLexer(`CREATE TABLE public.groups (
    id UUID NOT NULL,
);CREATE TABLE public.users (
    user_id UUID NOT NULL,
    username VARCHAR(256) NOT NULL,
    is_verified BOOLEAN NOT NULL DEFAULT false,
	description TEXT NOT NULL,
    CONSTRAINT users_pkey PRIMARY KEY (user_id),
);
`)).Parse()
		require.NoError(t, err)

		expected := `-- -
-- +description TEXT NOT NULL
ALTER TABLE public.users ADD COLUMN description TEXT NOT NULL;
`
		actual, err := Diff(before, after)
		require.NoError(t, err)

		if !assert.Equal(t, expected, actual.String()) {
			t.Errorf("❌: %s: stmt: %%#v: \n%#v", t.Name(), actual)
		}
	})

	t.Run("success,before,after,Index", func(t *testing.T) {
		t.Parallel()

		before, err := NewParser(NewLexer(`CREATE UNIQUE INDEX IF NOT EXISTS public.users_idx_by_username ON users (username);`)).Parse()
		require.NoError(t, err)

		after, err := NewParser(NewLexer(`CREATE UNIQUE INDEX IF NOT EXISTS public.users_idx_by_username ON users (username, age);`)).Parse()
		require.NoError(t, err)

		expected := `-- -CREATE UNIQUE INDEX public.users_idx_by_username ON users (username);
-- +CREATE UNIQUE INDEX public.users_idx_by_username ON users (username, age);
--  
DROP INDEX public.users_idx_by_username;
CREATE UNIQUE INDEX IF NOT EXISTS public.users_idx_by_username ON users (username, age);
`
		actual, err := Diff(before, after)
		require.NoError(t, err)
		if !assert.Equal(t, expected, actual.String()) {
			t.Errorf("❌: %s: stmt: %%#v: \n%#v", t.Name(), actual)
		}
	})

	t.Run("success,VARCHAR(10)->VARCHAR(11)", func(t *testing.T) {
		t.Parallel()

		before, err := NewParser(NewLexer(`CREATE TABLE public.users ( username VARCHAR(10) NOT NULL );`)).Parse()
		require.NoError(t, err)

		after, err := NewParser(NewLexer(`CREATE TABLE public.users ( username VARCHAR(11) NOT NULL );`)).Parse()
		require.NoError(t, err)

		expected := `-- -username VARCHAR(10) NOT NULL
-- +username VARCHAR(11) NOT NULL
ALTER TABLE public.users ALTER COLUMN username SET DATA TYPE VARCHAR(11);` + "\n"
		actual, err := Diff(before, after)
		require.NoError(t, err)

		if !assert.Equal(t, expected, actual.String()) {
			t.Errorf("❌: %s: stmt: %%#v: \n%#v", t.Name(), actual)
		}
	})

	t.Run("success,SET_DEFAULT_TRUE_FALSE", func(t *testing.T) {
		t.Parallel()

		before, err := NewParser(NewLexer(`CREATE TABLE public.passwords ( user_id UUID NOT NULL, password TEXT NOT NULL, is_verified BOOLEAN NOT NULL DEFAULT false, is_expired BOOLEAN NOT NULL DEFAULT true );`)).Parse()
		require.NoError(t, err)

		after, err := NewParser(NewLexer(`CREATE TABLE public.passwords ( user_id UUID NOT NULL, password TEXT NOT NULL, is_verified BOOLEAN NOT NULL DEFAULT FALSE, is_expired BOOLEAN NOT NULL DEFAULT TRUE );`)).Parse()
		require.NoError(t, err)

		expected := ``
		actual, err := Diff(before, after)
		assert.ErrorIs(t, err, ddl.ErrNoDifference)

		if !assert.Equal(t, expected, actual.String()) {
			t.Errorf("❌: %s: stmt: %%#v: \n%#v", t.Name(), actual)
		}
	})
}
