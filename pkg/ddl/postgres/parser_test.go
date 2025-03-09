//nolint:testpackage
package postgres

import (
	"testing"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
	require "github.com/hakadoriya/z.go/testingz/requirez"

	"github.com/kunitsucom/ddlctl/pkg/ddl"
	"github.com/kunitsucom/ddlctl/pkg/logs"
)

//nolint:paralleltest,tparallel
func TestParser_Parse(t *testing.T) {
	backup := logs.Trace
	t.Cleanup(func() {
		logs.Trace = backup
	})
	logs.Trace = logs.NewTrace()

	t.Run("success,CREATE_TABLE", func(t *testing.T) {
		t.Parallel()

		input := `CREATE TABLE public.groups ("id" UUID NOT NULL PRIMARY KEY, description TEXT); CREATE TABLE public.users (id UUID NOT NULL, group_id UUID NOT NULL REFERENCES "groups" ("id"), "name" VARCHAR(255) NOT NULL UNIQUE, "age" INT DEFAULT 0 CHECK ("age" >= 0), birthday TIMESTAMP NOT NULL, description TEXT, PRIMARY KEY ("id"));`
		expected := `CREATE TABLE public.groups (
    "id" UUID NOT NULL,
    description TEXT,
    CONSTRAINT groups_pkey PRIMARY KEY ("id")
);
CREATE TABLE public.users (
    id UUID NOT NULL,
    group_id UUID NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "age" INT DEFAULT 0,
    birthday TIMESTAMP NOT NULL,
    description TEXT,
    CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id"),
    CONSTRAINT users_unique_name UNIQUE ("name"),
    CONSTRAINT users_age_check CHECK ("age" >= 0),
    CONSTRAINT users_pkey PRIMARY KEY ("id")
);
`

		l := NewLexer(input)
		p := NewParser(l)
		actual, err := p.Parse()
		require.NoError(t, err)

		if !assert.Equal(t, expected, actual.String()) {
			t.Errorf("❌: %s: stmt: %%#v: \n%#v", t.Name(), actual)
		}

		t.Logf("ℹ️: %s: stmt: %%#v: \n%#v", t.Name(), actual)
		t.Logf("ℹ️: %s: stmt: %%s: \n%s", t.Name(), actual)
	})

	t.Run("success,complex_defaults", func(t *testing.T) {
		t.Parallel()

		input := `-- table: complex_defaults
CREATE TABLE IF NOT EXISTS complex_defaults (
    -- id is the primary key.
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    unique_code TEXT DEFAULT 'CODE-' || TO_CHAR(NOW(), 'YYYYMMDDHH24MISS') || '-' || LPAD(TO_CHAR(NEXTVAL('seq_complex_default')), 5, '0'),
    status TEXT DEFAULT 'pending',
    random_number INTEGER DEFAULT FLOOR(RANDOM() * 100::INTEGER)::INTEGER,
    json_data JSONB DEFAULT '{}',
    calculated_value INTEGER DEFAULT (SELECT COUNT(*) FROM another_table)
);
`
		expected := `CREATE TABLE IF NOT EXISTS complex_defaults (
    id SERIAL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    unique_code TEXT DEFAULT 'CODE-' || TO_CHAR(NOW(), 'YYYYMMDDHH24MISS') || '-' || LPAD(TO_CHAR(NEXTVAL('seq_complex_default')), 5, '0'),
    status TEXT DEFAULT 'pending',
    random_number INTEGER DEFAULT FLOOR(RANDOM() * 100::INTEGER)::INTEGER,
    json_data JSONB DEFAULT '{}',
    calculated_value INTEGER DEFAULT (SELECT COUNT(*) FROM another_table),
    CONSTRAINT complex_defaults_pkey PRIMARY KEY (id)
);
`

		l := NewLexer(input)
		p := NewParser(l)
		actual, err := p.Parse()
		require.NoError(t, err)

		if !assert.Equal(t, expected, actual.String()) {
			t.Errorf("❌: %s: stmt: %%#v: \n%#v", t.Name(), actual)
		}

		t.Logf("ℹ️: %s: stmt: %%#v: \n%#v", t.Name(), actual)
		t.Logf("ℹ️: %s: stmt: %%s: \n%s", t.Name(), actual)
	})

	failureTests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "failure,invalid",
			input:   `)invalid`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_INVALID",
			input:   `CREATE INVALID;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_INVALID",
			input:   `CREATE TABLE;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_IF_INVALID",
			input:   `CREATE TABLE IF;`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_IF_NOT_INVALID",
			input:   `CREATE TABLE IF NOT;`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_INVALID",
			input:   `CREATE TABLE "users";`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID",
			input:   `CREATE TABLE "users" ("id";`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_data_type_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID, CONSTRAINT "invalid" NOT;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_data_type__INVALID",
			input:   `CREATE TABLE "users" ("id" UUID)(;`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_COMMA_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID,(;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_DATA_TYPE_INVALID",
			input:   `CREATE TABLE "users" ("id" VARYING();`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_NOT",
			input:   `CREATE TABLE "users" ("id" UUID NULL NOT;`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_DEFAULT",
			input:   `CREATE TABLE "users" ("id" UUID DEFAULT ("id")`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_DEFAULT_OPEN_PAREN",
			input:   `CREATE TABLE "users" ("id" UUID DEFAULT ("id",`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_PRIMARY_KEY",
			input:   `CREATE TABLE "users" ("id" UUID PRIMARY NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_REFERENCES",
			input:   `CREATE TABLE "users" ("id" UUID REFERENCES NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_REFERENCES_IDENTS",
			input:   `CREATE TABLE "users" ("id" UUID REFERENCES "groups" (NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_CHECK",
			input:   `CREATE TABLE "users" ("id" UUID CHECK NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CHECK_INVALID_IDENTS",
			input:   `CREATE TABLE "users" ("id" UUID CHECK (NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_IDENT",
			input:   `CREATE TABLE "users" ("id" UUID, CONSTRAINT NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_PRIMARY",
			input:   `CREATE TABLE "users" ("id" UUID, PRIMARY NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_PRIMARY_KEY",
			input:   `CREATE TABLE "users" ("id" UUID, PRIMARY KEY NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_PRIMARY_KEY_OPEN_PAREN",
			input:   `CREATE TABLE "users" ("id" UUID, PRIMARY KEY (NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_REFERENCES_ON_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID REFERENCES foo (foo_id) ON`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_REFERENCES_ON_DELETE_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID REFERENCES foo (foo_id) ON DELETE`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_REFERENCES_ON_DELETE_NO_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID REFERENCES foo (foo_id) ON DELETE NO`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_REFERENCES_ON_DELETE_NO_ACTION_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID REFERENCES foo (foo_id) ON DELETE NO ACTION`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_FOREIGN",
			input:   `CREATE TABLE "users" ("id" UUID, FOREIGN NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_FOREIGN_KEY",
			input:   `CREATE TABLE "users" ("id" UUID, FOREIGN KEY NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_FOREIGN_KEY_OPEN_PAREN",
			input:   `CREATE TABLE "users" ("id" UUID, FOREIGN KEY (NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID, FOREIGN KEY ("group_id") NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID, FOREIGN KEY ("group_id") REFERENCES `,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_INVALID_IDENTS",
			input:   `CREATE TABLE "users" ("id" UUID, FOREIGN KEY ("group_id") REFERENCES "groups" NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_INVALID_CLOSE_PAREN",
			input:   `CREATE TABLE "users" ("id" UUID, FOREIGN KEY ("group_id") REFERENCES "groups" ("id")`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_ON_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID, FOREIGN KEY ("group_id") REFERENCES "groups" ("id") ON`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_ON_DELETE_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID, FOREIGN KEY ("group_id") REFERENCES "groups" ("id") ON DELETE`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_ON_DELETE_NO_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID, FOREIGN KEY ("group_id") REFERENCES "groups" ("id") ON DELETE NO`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_ON_DELETE_NO_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID, FOREIGN KEY ("group_id") REFERENCES "groups" ("id") ON DELETE NO ACTION`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_NOT_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID, UNIQUE NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID, UNIQUE;`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_COLUMN_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID, UNIQUE (;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_IDENTS_INVALID",
			input:   `CREATE TABLE "users" ("id" UUID, name TEXT, UNIQUE ("id", name)`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_INDEX_INVALID",
			input:   `CREATE INDEX NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_INDEX_IF_INVALID",
			input:   `CREATE INDEX IF;`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_INDEX_IF_NOT_INVALID",
			input:   `CREATE INDEX IF NOT;`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_INDEX_IF_NOT_EXISTS_INVALID",
			input:   `CREATE INDEX IF NOT EXISTS;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_INDEX_index_name_INVALID",
			input:   `CREATE INDEX users_idx_username NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_INDEX_index_name_ON_INVALID",
			input:   `CREATE INDEX users_idx_username ON NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_INDEX_index_name_ON_table_name_INVALID",
			input:   `CREATE INDEX users_idx_username ON users NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_INDEX_index_name_ON_table_name_USING_INVALID",
			input:   `CREATE INDEX users_idx_username ON users USING NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_INDEX_index_name_ON_table_name_USING_method_INVALID",
			input:   `CREATE INDEX users_idx_username ON users USING btree NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_INDEX_index_name_ON_table_name_USING_method_OPEN_INVALID",
			input:   `CREATE INDEX users_idx_username ON users USING btree (NOT)`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
	}

	for _, tt := range failureTests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewParser(NewLexer(tt.input)).Parse()
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestParser_parseColumn(t *testing.T) {
	t.Parallel()

	t.Run("success,TOKEN_COMMA", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer("( id VARCHAR(36),"))
		p.nextToken()
		p.nextToken()
		p.nextToken()
		_, _, err := p.parseColumn(&Ident{Name: "table_name", QuotationMark: `"`, Raw: `"table_name"`})
		require.NoError(t, err)
	})

	t.Run("failure,invalid", func(t *testing.T) {
		t.Parallel()

		_, _, err := NewParser(NewLexer(`NOT`)).parseColumn(&Ident{Name: "table_name", QuotationMark: `"`, Raw: `"table_name"`})
		require.ErrorIs(t, err, ddl.ErrUnexpectedCurrentToken)
	})

	t.Run("failure,parseDataType", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer("( id VARCHAR("))
		p.nextToken()
		p.nextToken()
		p.nextToken()
		_, _, err := p.parseColumn(&Ident{Name: "table_name", QuotationMark: `"`, Raw: `"table_name"`})
		require.ErrorIs(t, err, ddl.ErrUnexpectedCurrentToken)
	})
}

func TestParser_parseExpr(t *testing.T) {
	t.Parallel()

	t.Run("success,isReservedValue", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer(`(null)`))
		p.nextToken()
		p.nextToken()
		_, err := p.parseExpr()
		require.NoError(t, err)
	})

	t.Run("failure,invalid", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer(`NOT`))
		p.nextToken()
		p.nextToken()
		_, err := p.parseExpr()
		require.ErrorIs(t, err, ddl.ErrUnexpectedCurrentToken)
	})

	t.Run("failure,invalid2", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer(`((NOT`))
		p.nextToken()
		p.nextToken()
		_, err := p.parseExpr()
		require.ErrorIs(t, err, ddl.ErrUnexpectedCurrentToken)
	})
}

func TestParser_parseDataType(t *testing.T) {
	t.Parallel()

	t.Run("failure,TIMESTAMP_WITH_NOT", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer(`TIMESTAMP WITH NOT`))
		p.nextToken()
		p.nextToken()
		_, err := p.parseDataType()
		require.ErrorIs(t, err, ddl.ErrUnexpectedPeekToken)
	})

	t.Run("failure,TIMESTAMP_WITH_TIME_NOT", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer(`TIMESTAMP WITH TIME NOT`))
		p.nextToken()
		p.nextToken()
		_, err := p.parseDataType()
		require.ErrorIs(t, err, ddl.ErrUnexpectedPeekToken)
	})

	t.Run("failure,DOUBLE_NOT", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer(`DOUBLE NOT`))
		p.nextToken()
		p.nextToken()
		_, err := p.parseDataType()
		require.ErrorIs(t, err, ddl.ErrUnexpectedPeekToken)
	})

	t.Run("failure,DOUBLE_PRECISION", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer(`DOUBLE PRECISION(NOT`))
		p.nextToken()
		p.nextToken()
		_, err := p.parseDataType()
		require.ErrorIs(t, err, ddl.ErrUnexpectedCurrentToken)
	})

	t.Run("failure,CHARACTER_NOT", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer(`CHARACTER NOT`))
		p.nextToken()
		p.nextToken()
		_, err := p.parseDataType()
		require.ErrorIs(t, err, ddl.ErrUnexpectedPeekToken)
	})

	t.Run("failure,CHARACTER_VARYING_NOT", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer(`CHARACTER VARYING(NOT`))
		p.nextToken()
		p.nextToken()
		_, err := p.parseDataType()
		require.ErrorIs(t, err, ddl.ErrUnexpectedCurrentToken)
	})
}
