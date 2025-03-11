//nolint:testpackage
package spanner

import (
	"testing"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
	require "github.com/hakadoriya/z.go/testingz/requirez"

	"github.com/hakadoriya/ddlctl/pkg/ddl"
	"github.com/hakadoriya/ddlctl/pkg/logs"
)

//nolint:paralleltest,tparallel
func TestParser_Parse(t *testing.T) {
	backup := logs.Trace
	t.Cleanup(func() {
		logs.Trace = backup
	})
	logs.Trace = logs.NewTrace()

	t.Run("success,CREATE_TABLE", func(t *testing.T) {
		// t.Parallel()

		l := NewLexer(`CREATE TABLE "groups" ("id" STRING(36) NOT NULL, description STRING) PRIMARY KEY ("id"); CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0, ExpiredDate TIMESTAMP, description STRING, CONSTRAINT users_age_check CHECK ("age" >= 0), CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")) PRIMARY KEY ("id"), INTERLEAVE IN PARENT names ON DELETE NO ACTION, ROW DELETION POLICY (OLDER_THAN(ExpiredDate, INTERVAL 0 DAY));`)
		p := NewParser(l)
		actual, err := p.Parse()
		require.NoError(t, err)

		const expected = `CREATE TABLE "groups" (
    "id" STRING(36) NOT NULL,
    description STRING
) PRIMARY KEY ("id");
CREATE TABLE "users" (
    id STRING(36) NOT NULL,
    group_id STRING(36) NOT NULL,
    "name" STRING(255) NOT NULL,
    "age" INT64 DEFAULT 0,
    ExpiredDate TIMESTAMP,
    description STRING,
    CONSTRAINT users_age_check CHECK ("age" >= 0),
    CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")
) PRIMARY KEY ("id"),
INTERLEAVE IN PARENT names ON DELETE NO ACTION,
ROW DELETION POLICY (OLDER_THAN(ExpiredDate, INTERVAL 0 DAY));
`

		if !assert.Equal(t, expected, actual.String()) {
			t.Fail()
		}

		t.Logf("✅: %s: actual: %%#v: \n%#v", t.Name(), actual)
		t.Logf("✅: %s: actual: %%s: \n%s", t.Name(), actual)
	})

	t.Run("success,complex_defaults", func(t *testing.T) {
		// t.Parallel()

		l := NewLexer(`-- table: complex_defaults
CREATE TABLE IF NOT EXISTS complex_defaults (
    -- id is the primary key.
    id INT64,
    created_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP()),
    updated_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP()),
    unique_code STRING DEFAULT (GENERATE_UUID()),
    status STRING DEFAULT ('pending'),
    random_number INT64 DEFAULT (FLOOR(RANDOM() * 100)),
    json_data JSON DEFAULT ('{}'),
    calculated_value INT64 DEFAULT (SELECT COUNT(*) FROM another_table)
) PRIMARY KEY (id);
`)
		p := NewParser(l)
		actual, err := p.Parse()
		require.NoError(t, err)

		const expected = `CREATE TABLE IF NOT EXISTS complex_defaults (
    id INT64,
    created_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP()),
    updated_at TIMESTAMP DEFAULT (CURRENT_TIMESTAMP()),
    unique_code STRING DEFAULT (GENERATE_UUID()),
    status STRING DEFAULT ('pending'),
    random_number INT64 DEFAULT (FLOOR(RANDOM() * 100)),
    json_data JSON DEFAULT ('{}'),
    calculated_value INT64 DEFAULT (SELECT COUNT(*) FROM another_table)
) PRIMARY KEY (id);
`

		if !assert.Equal(t, expected, actual.String()) {
			t.Fail()
		}

		t.Logf("✅: %s: actual: %%#v: \n%#v", t.Name(), actual)
		t.Logf("✅: %s: actual: %%s: \n%s", t.Name(), actual)
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
			input:   `CREATE TABLE "users" ("id" STRING(36);`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_data_type_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36), enabled BOOL DEFAULT (FALSE);`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_data_type_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36), enabled BOOL DEFAULT (TRUE AND FALSE);`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_data_type_OPTIONS_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36), created_at TIMESTAMP OPTIONS (allow_commit_timestamp = true;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36), CONSTRAINT "invalid" NOT;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_data_type__INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36))(;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_COMMA_INVALID",
			input:   `CREATE TABLE "users" ("id" TIMESTAMP CREATE`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_DATA_TYPE_INVALID",
			input:   `CREATE TABLE "users" ("id" VARYING();`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_NOT",
			input:   `CREATE TABLE "users" ("id" STRING(36) NULL NOT;`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_DEFAULT",
			input:   `CREATE TABLE "users" ("id" STRING(36) DEFAULT ("id")`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_DEFAULT_OPEN_PAREN",
			input:   `CREATE TABLE "users" ("id" STRING(36) DEFAULT ("id",`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_PRIMARY_KEY",
			input:   `CREATE TABLE "users" ("id" STRING(36) PRIMARY NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_REFERENCES",
			input:   `CREATE TABLE "users" ("id" STRING(36) REFERENCES NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_REFERENCES_IDENTS",
			input:   `CREATE TABLE "users" ("id" STRING(36) REFERENCES "groups" (NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_CHECK",
			input:   `CREATE TABLE "users" ("id" STRING(36) CHECK NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CHECK_INVALID_IDENTS",
			input:   `CREATE TABLE "users" ("id" STRING(36) CHECK (NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_IDENT",
			input:   `CREATE TABLE "users" ("id" STRING(36), CONSTRAINT NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_CHECK_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36), CONSTRAINT constraint_name CHECK`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_CHECK_OPEN_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36), CONSTRAINT constraint_name CHECK (`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_PRIMARY",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_OPEN_PAREN_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_OPEN_PAREN_column_name_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_INTERLEAVE_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), INTERLEAVE;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_INTERLEAVE_IN_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), INTERLEAVE IN;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_INTERLEAVE_IN_PARENT_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), INTERLEAVE IN PARENT;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_INTERLEAVE_IN_PARENT_ON_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), INTERLEAVE IN PARENT table_name ON;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_INTERLEAVE_IN_PARENT_ON_DELETE_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), INTERLEAVE IN PARENT table_name ON DELETE;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_INTERLEAVE_IN_PARENT_ON_DELETE_CASCADE_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), INTERLEAVE IN PARENT table_name ON DELETE CASCADE NOT;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_INTERLEAVE_IN_PARENT_ON_DELETE_NO_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), INTERLEAVE IN PARENT table_name ON DELETE NO;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		// ROW DELETION POLICY (OLDER_THAN(ExpiredDate, INTERVAL 0 DAY));
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_ROW_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), ROW;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_ROW_DELETION_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), ROW DELETION;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_ROW_DELETION_POLICY_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), ROW DELETION POLICY;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_ROW_DELETION_POLICY_(_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), ROW DELETION POLICY (;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_ROW_DELETION_POLICY_(_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), ROW DELETION POLICY (;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_ROW_DELETION_POLICY_OLDER_THAN_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), ROW DELETION POLICY (OLDER_THAN;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_ROW_DELETION_POLICY_OLDER_THAN_OPEN_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), ROW DELETION POLICY (OLDER_THAN(;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_ROW_DELETION_POLICY_OLDER_THAN_OPEN_column_name_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), ROW DELETION POLICY (OLDER_THAN(ExpiredDate;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_ROW_DELETION_POLICY_OLDER_THAN_OPEN_column_name_COMMA_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), ROW DELETION POLICY (OLDER_THAN(ExpiredDate,;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_ROW_DELETION_POLICY_OLDER_THAN_OPEN_column_name_COMMA_INTERVAL_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), ROW DELETION POLICY (OLDER_THAN(ExpiredDate, INTERVAL;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_ROW_DELETION_POLICY_OLDER_THAN_OPEN_column_name_COMMA_INTERVAL_NUMBER_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), ROW DELETION POLICY (OLDER_THAN(ExpiredDate, INTERVAL 0;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_ROW_DELETION_POLICY_OLDER_THAN_OPEN_column_name_COMMA_INTERVAL_NUMBER_DAY_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), ROW DELETION POLICY (OLDER_THAN(ExpiredDate, INTERVAL 0 DAY;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_OPTION_PRIMARY_KEY_ROW_DELETION_POLICY_OLDER_THAN_OPEN_column_name_COMMA_INTERVAL_NUMBER_DAY_CLOSE_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36)) PRIMARY KEY (id), ROW DELETION POLICY (OLDER_THAN(ExpiredDate, INTERVAL 0 DAY);`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_FOREIGN",
			input:   `CREATE TABLE "users" ("id" STRING(36), FOREIGN NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_FOREIGN_KEY",
			input:   `CREATE TABLE "users" ("id" STRING(36), FOREIGN KEY NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_FOREIGN_KEY_OPEN_PAREN",
			input:   `CREATE TABLE "users" ("id" STRING(36), FOREIGN KEY (NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36), FOREIGN KEY ("group_id") NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36), FOREIGN KEY ("group_id") REFERENCES `,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_INVALID_IDENTS",
			input:   `CREATE TABLE "users" ("id" STRING(36), FOREIGN KEY ("group_id") REFERENCES "groups" NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_INVALID_CLOSE_PAREN",
			input:   `CREATE TABLE "users" ("id" STRING(36), FOREIGN KEY ("group_id") REFERENCES "groups" ("id")`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36), UNIQUE NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_INDEX_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36), UNIQUE INDEX users_idx_name NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_INDEX_COLUMN_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36), UNIQUE INDEX users_idx_name (NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_INDEX_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36), UNIQUE INDEX NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_IDENTS_INVALID",
			input:   `CREATE TABLE "users" ("id" STRING(36), name STRING, UNIQUE ("id", name)`,
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

	t.Run("success,TOKEN_SEMICOLON", func(t *testing.T) {
		_, err := NewParser(NewLexer(`;`)).Parse()
		require.NoError(t, err)
	})
}

func TestParser_parseColumn(t *testing.T) {
	t.Parallel()

	t.Run("success,TOKEN_COMMA", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer("( id STRING(36),"))
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

		p := NewParser(NewLexer("( id STRING("))
		p.nextToken()
		p.nextToken()
		p.nextToken()
		_, _, err := p.parseColumn(&Ident{Name: "table_name", QuotationMark: `"`, Raw: `"table_name"`})
		require.ErrorIs(t, err, ddl.ErrUnexpectedCurrentToken)
	})
}

func TestParser_parseColumnDefault(t *testing.T) {
	t.Parallel()

	t.Run("success,isReservedValue", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer(`DEFAULT TRUE,`))
		p.nextToken()
		p.nextToken()
		p.nextToken()
		_, err := p.parseColumnDefault()
		require.NoError(t, err)
	})
}

func TestParser_parseExpr(t *testing.T) {
	t.Parallel()

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

	t.Run("failure,invalid_paren_content", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer(`STRING(`))
		p.nextToken()
		p.nextToken()
		_, err := p.parseDataType()
		require.ErrorIs(t, err, ddl.ErrUnexpectedCurrentToken)
	})
}
