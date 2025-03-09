package spanner

import (
	"testing"

	"github.com/kunitsucom/util.go/testing/assert"
	"github.com/kunitsucom/util.go/testing/require"

	"github.com/kunitsucom/ddlctl/pkg/ddl"
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
ALTER TABLE complex_defaults ALTER COLUMN created_at DROP OPTIONS;
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
