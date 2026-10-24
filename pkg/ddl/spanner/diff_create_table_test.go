package spanner

import (
	"errors"
	"testing"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
	require "github.com/hakadoriya/z.go/testingz/requirez"

	"github.com/hakadoriya/ddlctl/pkg/ddl"
)

//nolint:paralleltest,tparallel
func TestDiffCreateTable(t *testing.T) {
	t.Run("failure,ddl.ErrNoDifference", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, description STRING, CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, description STRING, CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")) PRIMARY KEY ("id");`

		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)

		assert.ErrorIs(t, err, ddl.ErrNoDifference)
		assert.Nil(t, actual)

		t.Logf("✅: %s: actual: %%#v: \n%#v", t.Name(), actual)
		t.Logf("✅: %s: actual: %%s: \n%s", t.Name(), actual)
	})

	t.Run("failure,ddl.ErrNoDifference,SameContent", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, description STRING, CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE users (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, name STRING(255) NOT NULL, description STRING, CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES groups (id)) PRIMARY KEY (id);`

		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)

		assert.ErrorIs(t, err, ddl.ErrNoDifference)
		assert.Nil(t, actual)

		t.Logf("✅: %s: actual: %%#v: \n%#v", t.Name(), actual)
		t.Logf("✅: %s: actual: %%s: \n%s", t.Name(), actual)
	})

	t.Run("success,ADD_COLUMN", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, description STRING, CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING, CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")) PRIMARY KEY ("id");`

		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)

		expectedStr := `-- -
-- +"age" INT64 NOT NULL DEFAULT 0
ALTER TABLE "users" ADD COLUMN "age" INT64 NOT NULL DEFAULT 0;
-- -
-- +CONSTRAINT users_age_check CHECK ("age" >= 0)
ALTER TABLE "users" ADD CONSTRAINT users_age_check CHECK ("age" >= 0);
`

		assert.NoError(t, err)
		assert.Equal(t, expectedStr, actual.String())

		t.Logf("✅: %s: actual: %%#v:\n%#v", t.Name(), actual)
	})

	t.Run("success,DROP_COLUMN", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, description STRING) PRIMARY KEY ("id");`

		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)

		expectedStr := `-- -CONSTRAINT users_age_check CHECK ("age" >= 0)
-- +
ALTER TABLE "users" DROP CONSTRAINT users_age_check;
-- -"age" INT64 NOT NULL DEFAULT 0
-- +
ALTER TABLE "users" DROP COLUMN "age";
`

		assert.NoError(t, err)
		assert.Equal(t, expectedStr, actual.String())

		t.Logf("✅: %s: actual: %%#v:\n%#v", t.Name(), actual)
	})

	t.Run("success,ALTER_COLUMN_SET_DATA_TYPE", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING NOT NULL, "age" INT64 DEFAULT 0 CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`

		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)

		expectedStr := `-- -"name" STRING(255) NOT NULL
-- +"name" STRING NOT NULL
ALTER TABLE "users" ALTER COLUMN "name" STRING NOT NULL;
`

		assert.NoError(t, err)
		assert.Equal(t, expectedStr, actual.String())

		t.Logf("✅: %s: actual: %%#v:\n%#v", t.Name(), actual)
	})

	t.Run("success,ALTER_COLUMN_DROP_DEFAULT", func(t *testing.T) {
		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 CHECK ("age" >= 0), description STRING, CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, "age" INT64 CHECK ("age" >= 0), description STRING, CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")) PRIMARY KEY ("id");`
		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		expectedStr := `-- -"age" INT64 DEFAULT 0
-- +"age" INT64
ALTER TABLE "users" ALTER COLUMN "age" DROP DEFAULT;
`

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)
		assert.NoError(t, err)
		assert.Equal(t, expectedStr, actual.String())

		t.Logf("✅: %s: actual: %%#v:\n%#v", t.Name(), actual)
	})

	t.Run("success,ALTER_COLUMN_SET_DEFAULT", func(t *testing.T) {
		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, "age" INT64 CHECK ("age" >= 0), description STRING, CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 CHECK ("age" <> 0), description STRING, CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")) PRIMARY KEY (id);`
		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		expectedStr := `-- -"age" INT64
-- +"age" INT64 DEFAULT 0
ALTER TABLE "users" ALTER COLUMN "age" SET DEFAULT 0;
-- -CONSTRAINT users_age_check CHECK ("age" >= 0)
-- +
ALTER TABLE "users" DROP CONSTRAINT users_age_check;
-- -
-- +CONSTRAINT users_age_check CHECK ("age" <> 0)
ALTER TABLE "users" ADD CONSTRAINT users_age_check CHECK ("age" <> 0);
`

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)
		assert.NoError(t, err)
		assert.Equal(t, expectedStr, actual.String())

		t.Logf("✅: %s: actual: %%#v:\n%#v", t.Name(), actual)
	})

	t.Run("success,ALTER_TABLE_RENAME_TO", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "public.users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "app_users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`
		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		expectedStr := `-- -public.users
-- +public.app_users
ALTER TABLE "public.users" RENAME TO "public.app_users";
-- -CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")
-- +
ALTER TABLE "public.app_users" DROP CONSTRAINT users_group_id_fkey;
-- -CONSTRAINT users_age_check CHECK ("age" >= 0)
-- +
ALTER TABLE "public.app_users" DROP CONSTRAINT users_age_check;
-- -
-- +CONSTRAINT app_users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")
ALTER TABLE "public.app_users" ADD CONSTRAINT app_users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id");
-- -
-- +CONSTRAINT app_users_age_check CHECK ("age" >= 0)
ALTER TABLE "public.app_users" ADD CONSTRAINT app_users_age_check CHECK ("age" >= 0);
`

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)
		assert.NoError(t, err)
		assert.Equal(t, expectedStr, actual.String())

		t.Logf("✅: %s: actual: %%#v: \n%#v", t.Name(), actual)
		t.Logf("✅: %s: actual: %%s: \n%s", t.Name(), actual)
	})

	t.Run("success,SET_NOT_NULL", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`
		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		expectedStr := `-- -"age" INT64 DEFAULT 0
-- +"age" INT64 NOT NULL DEFAULT 0
ALTER TABLE "users" ALTER COLUMN "age" INT64 NOT NULL;
`

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)
		assert.NoError(t, err)
		assert.Equal(t, expectedStr, actual.String())

		t.Logf("✅: %s: actual: %%#v:\n%#v", t.Name(), actual)
	})

	t.Run("success,DROP_NOT_NULL", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`
		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		expectedStr := `-- -"age" INT64 NOT NULL DEFAULT 0
-- +"age" INT64 DEFAULT 0
ALTER TABLE "users" ALTER COLUMN "age" INT64;
`

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)
		assert.NoError(t, err)
		assert.Equal(t, expectedStr, actual.String())

		t.Logf("✅: %s: actual: %%#v:\n%#v", t.Name(), actual)
	})

	t.Run("success,ALTER_PRIMARY_KEY", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id", name);`
		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)
		assert.NoError(t, err)
		expected := `-- -PRIMARY KEY ("id")
-- +PRIMARY KEY ("id", name)
DROP TABLE "users";
CREATE TABLE "users" (
    id STRING(36) NOT NULL,
    group_id STRING(36) NOT NULL,
    "name" STRING(255) NOT NULL,
    "age" INT64 NOT NULL DEFAULT 0,
    description STRING,
    CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id"),
    CONSTRAINT users_age_check CHECK ("age" >= 0)
) PRIMARY KEY ("id", name);
`

		assert.Equal(t, expected, actual.String())
	})

	t.Run("success,ALTER_ADD_ROW_DELETION_POLICY", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING, CreatedAt TIMESTAMP) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING, CreatedAt TIMESTAMP) PRIMARY KEY ("id"), ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 7 DAY));`
		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)
		assert.NoError(t, err)
		expected := `-- -
-- +ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 7 DAY))
ALTER TABLE "users" ADD ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 7 DAY));
`

		assert.Equal(t, expected, actual.String())
	})

	t.Run("success,ALTER_REPLACE_ROW_DELETION_POLICY", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING, CreatedAt TIMESTAMP) PRIMARY KEY ("id"), ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 30 DAY));`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING, CreatedAt TIMESTAMP) PRIMARY KEY ("id"), ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 7 DAY));`
		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)
		assert.NoError(t, err)
		expected := `-- -ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 30 DAY))
-- +ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 7 DAY))
ALTER TABLE "users" REPLACE ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 7 DAY));
`

		assert.Equal(t, expected, actual.String())
	})

	t.Run("success,ALTER_REPLACE_DROP_DELETION_POLICY", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING, CreatedAt TIMESTAMP) PRIMARY KEY ("id"), ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 30 DAY));`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING, CreatedAt TIMESTAMP) PRIMARY KEY ("id");`
		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)
		assert.NoError(t, err)
		expected := `-- -ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 30 DAY))
-- +
ALTER TABLE "users" DROP ROW DELETION POLICY;
`

		assert.Equal(t, expected, actual.String())
	})

	t.Run("success,DROP_ADD_FOREIGN_KEY", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING, CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING, CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id, name) REFERENCES "groups" ("id", name)) PRIMARY KEY ("id");`
		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		expectedStr := `-- -CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")
-- +
ALTER TABLE "users" DROP CONSTRAINT users_group_id_fkey;
-- -
-- +CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id, name) REFERENCES "groups" ("id", name)
ALTER TABLE "users" ADD CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id, name) REFERENCES "groups" ("id", name);
`

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)
		assert.NoError(t, err)
		assert.Equal(t, expectedStr, actual.String())

		t.Logf("✅: %s: actual: %%#v:\n%#v", t.Name(), actual)
	})

	t.Run("success,DROP_ADD_UNIQUE", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING, CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING, CONSTRAINT users_group_id_fkey_2 FOREIGN KEY (group_id) REFERENCES "groups" ("id")) PRIMARY KEY ("id");`
		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		expectedStr := `-- -CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id")
-- +
ALTER TABLE "users" DROP CONSTRAINT users_group_id_fkey;
-- -
-- +CONSTRAINT users_group_id_fkey_2 FOREIGN KEY (group_id) REFERENCES "groups" ("id")
ALTER TABLE "users" ADD CONSTRAINT users_group_id_fkey_2 FOREIGN KEY (group_id) REFERENCES "groups" ("id");
`

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)
		assert.NoError(t, err)
		assert.Equal(t, expectedStr, actual.String())

		t.Logf("✅: %s: actual: %%#v:\n%#v", t.Name(), actual)
	})

	t.Run("success,ALTER_COLUMN_SET_DEFAULT_OVERWRITE", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 NOT NULL CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL, "name" STRING(255) NOT NULL, "age" INT64 DEFAULT ( (0 + 3) - 1 * 4 / 2 ) NOT NULL CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`
		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		expectedStr := `-- -"age" INT64 NOT NULL DEFAULT 0
-- +"age" INT64 NOT NULL DEFAULT ((0 + 3) - 1 * 4 / 2)
ALTER TABLE "users" ALTER COLUMN "age" SET DEFAULT ((0 + 3) - 1 * 4 / 2);
`

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)
		assert.NoError(t, err)
		assert.Equal(t, expectedStr, actual.String())

		t.Logf("✅: %s: actual: %%#v:\n%#v", t.Name(), actual)
	})

	t.Run("success,ALTER_COLUMN_SET_DEFAULT_complex", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE complex_defaults (
    id INT64,
    created_at TIMESTAMP OPTIONS (allow_commit_timestamp=true, option_name=null),
    updated_at TIMESTAMP,
    unique_code STRING,
    status STRING DEFAULT ('pending'),
    random_number INT64 DEFAULT (FLOOR(RANDOM() * 100)),
    json_data JSON DEFAULT ('{}'),
    calculated_value INT64 DEFAULT (SELECT COUNT(*) FROM another_table)
)  PRIMARY KEY (id);
`
		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		after := `CREATE TABLE complex_defaults (
    id INT64,
    created_at TIMESTAMP,
    updated_at TIMESTAMP OPTIONS (allow_commit_timestamp=true, option_name=null),
    unique_code STRING DEFAULT (GENERATE_UUID()),
    status STRING DEFAULT ('pending'),
    random_number INT64 DEFAULT (FLOOR(RANDOM() * 100)),
    json_data JSON DEFAULT ('{}'),
    calculated_value INT64 DEFAULT (SELECT COUNT(*) FROM another_table)
) PRIMARY KEY (id);
`
		afterDDL, err := NewParser(NewLexer(after)).Parse()
		require.NoError(t, err)

		expectedStr := `-- -created_at TIMESTAMP OPTIONS (allow_commit_timestamp = TRUE, option_name = NULL)
-- +created_at TIMESTAMP
ALTER TABLE complex_defaults ALTER COLUMN created_at SET OPTIONS (allow_commit_timestamp = NULL, option_name = NULL);
-- -updated_at TIMESTAMP
-- +updated_at TIMESTAMP OPTIONS (allow_commit_timestamp = TRUE, option_name = NULL)
ALTER TABLE complex_defaults ALTER COLUMN updated_at SET OPTIONS (allow_commit_timestamp = TRUE, option_name = NULL);
-- -unique_code STRING
-- +unique_code STRING DEFAULT (GENERATE_UUID())
ALTER TABLE complex_defaults ALTER COLUMN unique_code SET DEFAULT (GENERATE_UUID());
`

		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)
		assert.NoError(t, err)
		assert.Equal(t, expectedStr, actual.String())

		t.Logf("✅: %s: actual: %%#v:\n%#v", t.Name(), actual)
	})

	t.Run("success,DiffCreateTableUseAlterTableAddConstraintNotValid", func(t *testing.T) {
		t.Parallel()

		beforeDDL, err := NewParser(NewLexer(`CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0, description STRING) PRIMARY KEY ("id");`)).Parse()
		require.NoError(t, err)

		afterDDL, err := NewParser(NewLexer(`CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`)).Parse()
		require.NoError(t, err)

		expected := `-- -
-- +CONSTRAINT users_age_check CHECK ("age" >= 0)
ALTER TABLE "users" ADD CONSTRAINT users_age_check CHECK ("age" >= 0) NOT VALID;
`
		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(true),
		)

		assert.NoError(t, err)
		assert.Equal(t, expected, actual.String())

		t.Logf("✅: %s: actual: %%#v: \n%#v", t.Name(), actual)
		t.Logf("✅: %s: actual: %%s: \n%s", t.Name(), actual)
	})

	t.Run("success,CREATE_TABLE", func(t *testing.T) {
		t.Parallel()

		afterDDL, err := NewParser(NewLexer(`CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`)).Parse()
		require.NoError(t, err)

		expected := `CREATE TABLE "users" (
    id STRING(36) NOT NULL,
    group_id STRING(36) NOT NULL,
    "name" STRING(255) NOT NULL,
    "age" INT64 DEFAULT 0,
    description STRING,
    CONSTRAINT users_group_id_fkey FOREIGN KEY (group_id) REFERENCES "groups" ("id"),
    CONSTRAINT users_age_check CHECK ("age" >= 0)
) PRIMARY KEY ("id");
`
		//nolint:forcetypeassert
		actual, err := DiffCreateTable(
			nil,
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(true),
		)

		assert.NoError(t, err)
		assert.Equal(t, expected, actual.String())

		t.Logf("✅: %s: actual: %%#v: \n%#v", t.Name(), actual)
		t.Logf("✅: %s: actual: %%s: \n%s", t.Name(), actual)
	})

	t.Run("success,DROP_TABLE", func(t *testing.T) {
		t.Parallel()

		before := `CREATE TABLE "users" (id STRING(36) NOT NULL, group_id STRING(36) NOT NULL REFERENCES "groups" ("id"), "name" STRING(255) NOT NULL, "age" INT64 DEFAULT 0 CHECK ("age" >= 0), description STRING) PRIMARY KEY ("id");`

		beforeDDL, err := NewParser(NewLexer(before)).Parse()
		require.NoError(t, err)

		//nolint:forcetypeassert
		ddls, err := DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			nil,
			DiffCreateTableUseAlterTableAddConstraintNotValid(true),
		)

		assert.NoError(t, err)
		assert.Equal(t, &DDL{
			Stmts: []Stmt{
				&DropTableStmt{
					Name: &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
				},
			},
		}, ddls)

		t.Logf("✅: %s:\n%s", t.Name(), ddls)
	})
}

// TestDiffCreateTable_PrimaryKeyOrder は、主キーの並び順の書き方のゆれ (ASC の明示・省略、大文字・小文字) を
// 同じものとして比べ、並び順そのものが違うときだけ表を作り直すことを確かめる。
func TestDiffCreateTable_PrimaryKeyOrder(t *testing.T) {
	t.Parallel()

	const columns = `CREATE TABLE T (A STRING(36) NOT NULL, B STRING(36) NOT NULL) `

	diff := func(t *testing.T, beforePK, afterPK string) (*DDL, error) {
		t.Helper()

		beforeDDL, err := NewParser(NewLexer(columns + beforePK + ";")).Parse()
		require.NoError(t, err)
		afterDDL, err := NewParser(NewLexer(columns + afterPK + ";")).Parse()
		require.NoError(t, err)

		//nolint:forcetypeassert
		return DiffCreateTable(
			beforeDDL.Stmts[0].(*CreateTableStmt),
			afterDDL.Stmts[0].(*CreateTableStmt),
			DiffCreateTableUseAlterTableAddConstraintNotValid(false),
		)
	}

	noDifferenceCases := []struct {
		name     string
		beforePK string
		afterPK  string
	}{
		{name: "ASC を明示しても省略と同じ", beforePK: "PRIMARY KEY (A ASC, B DESC)", afterPK: "PRIMARY KEY (A, B DESC)"},
		{name: "小文字の desc は DESC と同じ", beforePK: "PRIMARY KEY (A, B desc)", afterPK: "PRIMARY KEY (A, B DESC)"},
		{name: "小文字の asc は省略と同じ", beforePK: "PRIMARY KEY (A asc, B)", afterPK: "PRIMARY KEY (A, B)"},
		{name: "引用符の有無は区別しない", beforePK: "PRIMARY KEY (`A`, `B` DESC)", afterPK: "PRIMARY KEY (A, B DESC)"},
	}
	for _, tc := range noDifferenceCases {
		t.Run("failure,ddl.ErrNoDifference,"+tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := diff(t, tc.beforePK, tc.afterPK)
			assert.ErrorIs(t, err, ddl.ErrNoDifference)
			assert.Nil(t, actual)
		})
	}

	recreateCases := []struct {
		name     string
		beforePK string
		afterPK  string
	}{
		{name: "ASC から DESC", beforePK: "PRIMARY KEY (A, B)", afterPK: "PRIMARY KEY (A, B DESC)"},
		{name: "DESC から ASC の明示", beforePK: "PRIMARY KEY (A, B DESC)", afterPK: "PRIMARY KEY (A, B ASC)"},
		{name: "列の順番", beforePK: "PRIMARY KEY (A, B)", afterPK: "PRIMARY KEY (B, A)"},
	}
	for _, tc := range recreateCases {
		t.Run("success,DropAndCreateTable,"+tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := diff(t, tc.beforePK, tc.afterPK)
			require.NoError(t, err)
			require.Equal(t, 2, len(actual.Stmts))
			_, isDrop := actual.Stmts[0].(*DropTableStmt)
			assert.True(t, isDrop)
			_, isCreate := actual.Stmts[1].(*CreateTableStmt)
			assert.True(t, isCreate)
		})
	}
}

func TestDiffCreateTable_InterleaveInParent(t *testing.T) {
	t.Parallel()

	// INTERLEAVE IN PARENT 句の書き方と、その意味 (親テーブル名と ON DELETE の動作)
	type interleave struct {
		clause  string
		meaning string
	}
	parents := []struct{ raw, name string }{{"P", "P"}, {"`P`", "P"}, {"Q", "Q"}}
	onDeletes := []struct{ raw, action string }{
		// ON DELETE を省略したときの動作は NO ACTION
		{"", "NO ACTION"},
		{" ON DELETE NO ACTION", "NO ACTION"},
		{" on delete no action", "NO ACTION"},
		{" ON DELETE CASCADE", "CASCADE"},
		{" on Delete Cascade", "CASCADE"},
	}
	interleaves := make([]interleave, 0, 1+len(parents)*len(onDeletes))
	interleaves = append(interleaves, interleave{clause: "", meaning: "not interleaved"})
	for _, parent := range parents {
		for _, onDelete := range onDeletes {
			interleaves = append(interleaves, interleave{
				clause:  ",\nINTERLEAVE IN PARENT " + parent.raw + onDelete.raw,
				meaning: parent.name + " ON DELETE " + onDelete.action,
			})
		}
	}

	parse := func(t *testing.T, clause string) *CreateTableStmt {
		t.Helper()

		parsed, err := NewParser(NewLexer("CREATE TABLE T (Id STRING(36) NOT NULL) PRIMARY KEY (Id)" + clause + ";")).Parse()
		require.NoError(t, err)
		//nolint:forcetypeassert
		return parsed.Stmts[0].(*CreateTableStmt)
	}

	// 全ての書き方の組について、意味が同じなら差分なし、意味が違えば DROP TABLE + CREATE TABLE になることを検証する。
	// 書き方の違いだけで差分が出ると、ddlctl diff / apply が毎回 DROP TABLE + CREATE TABLE を生成し、テーブルのデータが消える。
	for _, before := range interleaves {
		for _, after := range interleaves {
			actual, err := DiffCreateTable(parse(t, before.clause), parse(t, after.clause), DiffCreateTableUseAlterTableAddConstraintNotValid(false))
			if before.meaning == after.meaning {
				if !errors.Is(err, ddl.ErrNoDifference) {
					t.Errorf("expected no difference: before=%q, after=%q: err=%v\n%s", before.clause, after.clause, err, actual)
				}
				continue
			}
			if err != nil {
				t.Errorf("expected DROP TABLE + CREATE TABLE: before=%q, after=%q: err=%v", before.clause, after.clause, err)
				continue
			}
			if len(actual.Stmts) != 2 {
				t.Errorf("expected DROP TABLE + CREATE TABLE: before=%q, after=%q:\n%s", before.clause, after.clause, actual)
				continue
			}
			_, isDrop := actual.Stmts[0].(*DropTableStmt)
			_, isCreate := actual.Stmts[1].(*CreateTableStmt)
			if !isDrop || !isCreate {
				t.Errorf("expected DROP TABLE + CREATE TABLE: before=%q, after=%q:\n%s", before.clause, after.clause, actual)
			}
		}
	}
}
