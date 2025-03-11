//nolint:testpackage
package mysql

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

		l := NewLexer("CREATE TABLE `groups` (`group_id` VARCHAR(36) NOT NULL PRIMARY KEY, description TEXT CHARACTER SET utf8mbb4 COMMENT '備考') ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci; CREATE TABLE `users` (user_id VARCHAR(36) NOT NULL, group_id VARCHAR(36) NOT NULL REFERENCES `groups` (`group_id`), `name` VARCHAR(255) COLLATE utf8mb4_bin NOT NULL UNIQUE, `age` INT DEFAULT 0 CHECK (`age` >= 0), birthdate DATE, country char(3), description LONGTEXT CHARACTER SET utf8mbb4 COMMENT '備考', created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, PRIMARY KEY (`user_id`)) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='user 🙆‍';")
		p := NewParser(l)
		actual, err := p.Parse()
		require.NoError(t, err)

		expected := `CREATE TABLE ` + "`" + `groups` + "`" + ` (
    ` + "`" + `group_id` + "`" + ` VARCHAR(36) NOT NULL,
    description TEXT CHARACTER SET utf8mbb4 NULL COMMENT '備考',
    PRIMARY KEY (` + "`" + `group_id` + "`" + `)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
CREATE TABLE ` + "`" + `users` + "`" + ` (
    user_id VARCHAR(36) NOT NULL,
    group_id VARCHAR(36) NOT NULL,
    ` + "`" + `name` + "`" + ` VARCHAR(255) COLLATE utf8mb4_bin NOT NULL,
    ` + "`" + `age` + "`" + ` INT NULL DEFAULT 0,
    birthdate DATE NULL,
    country char(3) NULL,
    description LONGTEXT CHARACTER SET utf8mbb4 NULL COMMENT '備考',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (` + "`" + `user_id` + "`" + `),
    CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES ` + "`" + `groups` + "`" + ` (` + "`" + `group_id` + "`" + `),
    UNIQUE KEY users_unique_name (` + "`" + `name` + "`" + `),
    CONSTRAINT users_age_check CHECK (` + "`" + `age` + "`" + ` >= 0)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='user 🙆‍';
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
    id INT NOT NULL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    age INT DEFAULT 25,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status ENUM('Active', 'Inactive') DEFAULT 'Active',
    salary DECIMAL(10, 2) DEFAULT (10000+1),
    random_number INTEGER DEFAULT (FLOOR(RAND() * 100)),
    notes VARCHAR(1024) DEFAULT 'This is a note for ',
    is_admin BOOLEAN DEFAULT false,
    KEY complex_defaults_idx_on_name (name),
	CONSTRAINT users_age_check CHECK ((age >= 0))
);
`)
		p := NewParser(l)
		actual, err := p.Parse()
		require.NoError(t, err)

		expected := `CREATE TABLE IF NOT EXISTS complex_defaults (
    id INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    age INT NULL DEFAULT 25,
    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    status ENUM('Active', 'Inactive') NULL DEFAULT 'Active',
    salary DECIMAL(10, 2) NULL DEFAULT (10000 + 1),
    random_number INTEGER NULL DEFAULT (FLOOR(RAND() * 100)),
    notes VARCHAR(1024) NULL DEFAULT 'This is a note for ',
    is_admin TINYINT(1) NULL DEFAULT false,
    PRIMARY KEY (id),
    KEY complex_defaults_idx_on_name (name),
    CONSTRAINT users_age_check CHECK ((age >= 0))
);
`
		if !assert.Equal(t, expected, actual.String()) {
			t.Fail()
		}

		t.Logf("✅: %s: actual: %%#v: \n%#v", t.Name(), actual)
		t.Logf("✅: %s: actual: %%s: \n%s", t.Name(), actual)
	})

	t.Run("success,CREATE_TABLE_TYPE_ANNOTATION", func(t *testing.T) {
		// t.Parallel()

		l := NewLexer(`CREATE TABLE IF NOT EXISTS public.users (
    user_id VARCHAR(36) NOT NULL,
    username VARCHAR(256) NOT NULL,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT users_pkey PRIMARY KEY (user_id),
    INDEX users_idx_by_username (username DESC)
);
`)
		p := NewParser(l)
		actual, err := p.Parse()
		require.NoError(t, err)

		expected := `CREATE TABLE IF NOT EXISTS public.users (
    user_id VARCHAR(36) NOT NULL,
    username VARCHAR(256) NOT NULL,
    is_verified TINYINT(1) NOT NULL DEFAULT false,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id),
    KEY users_idx_by_username (username DESC)
);
`
		if !assert.Equal(t, expected, actual.String()) {
			t.Fail()
		}

		t.Logf("✅: %s: actual: %%#v: \n%#v", t.Name(), actual)
		t.Logf("✅: %s: actual: %%s: \n%s", t.Name(), actual)
	})

	t.Run("success,SEMICOLON", func(t *testing.T) {
		// t.Parallel()

		l := NewLexer(`;`)
		p := NewParser(l)
		actual, err := p.Parse()
		require.NoError(t, err)

		expected := ``
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
			input:   `CREATE TABLE "users" ("id" VARCHAR(36);`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_data_type_CHARACTER_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) CHARACTER`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_data_type_CHARACTER_SET_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) CHARACTER SET`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), CONSTRAINT "invalid" NOT;`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_data_type__INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36))(;`,
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
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) NULL NOT;`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_ON_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), ts DATETIME ON`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_ON_UPDATE_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), ts DATETIME ON UPDATE`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_DEFAULT",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) DEFAULT ("id")`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_DEFAULT_OPEN_PAREN",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) DEFAULT ("id",`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_PRIMARY_KEY",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) PRIMARY NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_REFERENCES",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) REFERENCES NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_REFERENCES_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) REFERENCES "groups" (NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_REFERENCES_ON_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) REFERENCES "groups" (id) ON`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_REFERENCES_ON_UPDATE_NO_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) REFERENCES "groups" (id) ON UPDATE NO`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_REFERENCES_ON_UPDATE_NO_ACTION_ON_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) REFERENCES "groups" (id) ON UPDATE NO ACTION ON`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_REFERENCES_ON_UPDATE_NO_ACTION_ON_DELETE_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) REFERENCES "groups" (id) ON UPDATE NO ACTION ON DELETE`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_REFERENCES_ON_UPDATE_NO_ACTION_ON_DELETE_NO_ACTION_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) REFERENCES "groups" (id) ON UPDATE NO ACTION ON DELETE NO ACTION`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_REFERENCESIN_IDENTS",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) REFERENCES "groups" (id) ON UPDATE NO ACTION ON DELETE NO`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_INVALID_CHECK",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) CHECK NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CHECK_INVALID_IDENTS",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36) CHECK (NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_IDENT",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), CONSTRAINT NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_PRIMARY",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), PRIMARY NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_PRIMARY_KEY",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), PRIMARY KEY NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_PRIMARY_KEY_OPEN_PAREN",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), PRIMARY KEY (NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_FOREIGN",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), FOREIGN NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_FOREIGN_KEY",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), FOREIGN KEY NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_INVALID_FOREIGN_KEY_OPEN_PAREN",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), FOREIGN KEY (NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), FOREIGN KEY ("group_id") NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), FOREIGN KEY ("group_id") REFERENCES `,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_INVALID_IDENTS",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), FOREIGN KEY ("group_id") REFERENCES "groups" NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_INVALID_CLOSE_PAREN",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), FOREIGN KEY ("group_id") REFERENCES "groups" ("id")`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_ON_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), FOREIGN KEY ("group_id") REFERENCES "groups" ("id") ON`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_ON_DELETE_NO_ACTION_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), FOREIGN KEY ("group_id") REFERENCES "groups" ("id") ON DELETE NO ACTION`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_ON_DELETE_NO_ACTION_ON_UPDATE_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), FOREIGN KEY ("group_id") REFERENCES "groups" ("id") ON DELETE NO ACTION ON UPDATE`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_FOREIGN_KEY_IDENTS_REFERENCES_ON_DELETE_NO_ACTION_ON_UPDATE_NO_ACTION_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), FOREIGN KEY ("group_id") REFERENCES "groups" ("id") ON DELETE NO ACTION ON UPDATE NO ACTION`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), UNIQUE NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_INDEX_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), UNIQUE INDEX users_idx_name NOT`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_INDEX_COLUMN_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), UNIQUE INDEX users_idx_name (NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_INDEX_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), UNIQUE INDEX NOT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_IDENTS_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, UNIQUE ("id", name)`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_CHECK_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, CHECK INVALID`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_CHECK_OPEN_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, CHECK (INVALID`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_CHECK_OPEN_CLOSE_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, CHECK ("id" NOT NULL)`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_KEY_IDENTS_ENGINE_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, UNIQUE KEY users_idx_on_id_name ("id", name)) ENGINE`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_KEY_IDENTS_ENGINE_EQUAL_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, UNIQUE KEY users_idx_on_id_name ("id", name)) ENGINE=`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_KEY_IDENTS_AUTO_INCREMENT_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, UNIQUE KEY users_idx_on_id_name ("id", name)) AUTO_INCREMENT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_KEY_IDENTS_AUTO_INCREMENT_EQUAL_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, UNIQUE KEY users_idx_on_id_name ("id", name)) AUTO_INCREMENT=`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_KEY_IDENTS_DEFAULT_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, UNIQUE KEY users_idx_on_id_name ("id", name)) DEFAULT=`,
			wantErr: ddl.ErrUnexpectedPeekToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_KEY_IDENTS_DEFAULT_CHARSET_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, UNIQUE KEY users_idx_on_id_name ("id", name)) DEFAULT CHARSET`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_KEY_IDENTS_DEFAULT_CHARSET_EQUAL_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, UNIQUE KEY users_idx_on_id_name ("id", name)) DEFAULT CHARSET=`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_KEY_IDENTS_COLLATE_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, UNIQUE KEY users_idx_on_id_name ("id", name)) COLLATE`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_KEY_IDENTS_COLLATE_EQUAL_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, UNIQUE KEY users_idx_on_id_name ("id", name)) COLLATE=`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_KEY_IDENTS_COMMENT_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, UNIQUE KEY users_idx_on_id_name ("id", name)) COMMENT`,
			wantErr: ddl.ErrUnexpectedCurrentToken,
		},
		{
			name:    "failure,CREATE_TABLE_table_name_column_name_CONSTRAINT_UNIQUE_KEY_IDENTS_COMMENT_EQUAL_INVALID",
			input:   `CREATE TABLE "users" ("id" VARCHAR(36), name TEXT, UNIQUE KEY users_idx_on_id_name ("id", name)) COMMENT=`,
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

func TestParser_parseOnAction(t *testing.T) {
	t.Parallel()

	t.Run("failure,ON", func(t *testing.T) {
		t.Parallel()
		p := NewParser(NewLexer("A ON"))
		p.nextToken()
		p.nextToken()
		_, err := p.parseOnAction()
		require.ErrorIs(t, err, ddl.ErrUnexpectedCurrentToken)
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

	t.Run("success,TIME", func(t *testing.T) {
		t.Parallel()

		p := NewParser(NewLexer(`TIME`))
		p.nextToken()
		p.nextToken()
		_, err := p.parseDataType()
		require.NoError(t, err)
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
