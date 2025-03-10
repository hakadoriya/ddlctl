package postgres

import (
	"fmt"
	"testing"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
	require "github.com/hakadoriya/z.go/testingz/requirez"
)

func Test_isAlterTableAction(t *testing.T) {
	t.Parallel()

	(&RenameTable{}).isAlterTableAction()
	(&RenameConstraint{}).isAlterTableAction()
	(&RenameColumn{}).isAlterTableAction()
	(&AddColumn{}).isAlterTableAction()
	(&DropColumn{}).isAlterTableAction()
	(&AlterColumnSetDataType{}).isAlterTableAction()
	(&AlterColumnSetDefault{}).isAlterTableAction()
	(&AlterColumnDropDefault{}).isAlterTableAction()
	(&AlterColumnSetNotNull{}).isAlterTableAction()
	(&AlterColumnDropNotNull{}).isAlterTableAction()
	(&AddConstraint{}).isAlterTableAction()
	(&DropConstraint{}).isAlterTableAction()
	(&AlterConstraint{}).isAlterTableAction()
}

func TestAlterTableStmt_String(t *testing.T) {
	t.Parallel()

	t.Run("success,RenameTable", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Comment: "test comment content",
			Name:    &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Action:  &RenameTable{NewName: &ObjectName{Name: &Ident{Name: "accounts", QuotationMark: `"`, Raw: `"accounts"`}}},
		}

		expected := `-- test comment content
ALTER TABLE "users" RENAME TO "accounts";
`
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,RenameColumn", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name:   &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Action: &RenameColumn{Name: &Ident{Name: "name", QuotationMark: `"`, Raw: `"name"`}, NewName: &Ident{Name: "username", QuotationMark: `"`, Raw: `"username"`}},
		}

		expected := `ALTER TABLE "users" RENAME COLUMN "name" TO "username";` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,RenameConstraint", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name:   &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Action: &RenameConstraint{Name: &Ident{Name: "users_pkey", QuotationMark: `"`, Raw: `"users_pkey"`}, NewName: &Ident{Name: "users_id_pkey", QuotationMark: `"`, Raw: `"users_id_pkey"`}},
		}

		expected := `ALTER TABLE "users" RENAME CONSTRAINT "users_pkey" TO "users_id_pkey";` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,AddColumn", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name: &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Action: &AddColumn{
				Column: &Column{
					Name:     &Ident{Name: "age", QuotationMark: `"`, Raw: `"age"`},
					DataType: &DataType{Name: "INTEGER"},
				},
			},
		}

		expected := `ALTER TABLE "users" ADD COLUMN "age" INTEGER;` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,DropColumn", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name:   &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Action: &DropColumn{Name: &Ident{Name: "age", QuotationMark: `"`, Raw: `"age"`}},
		}

		expected := `ALTER TABLE "users" DROP COLUMN "age";` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,AlterColumnSetDataType", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name: &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Action: &AlterColumnSetDataType{
				Name:     &Ident{Name: "age", QuotationMark: `"`, Raw: `"age"`},
				DataType: &DataType{Name: "INTEGER"},
			},
		}

		expected := `ALTER TABLE "users" ALTER COLUMN "age" SET DATA TYPE INTEGER;` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,AlterColumnSetDefault", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name: &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Action: &AlterColumnSetDefault{
				Name:    &Ident{Name: "age", QuotationMark: `"`, Raw: `"age"`},
				Default: &Default{Value: &Expr{[]*Ident{{Name: "0", Raw: "0"}}}},
			},
		}

		expected := `ALTER TABLE "users" ALTER COLUMN "age" SET DEFAULT 0;` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,AlterColumnDropDefault", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name: &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Action: &AlterColumnDropDefault{
				Name: &Ident{Name: "age", QuotationMark: `"`, Raw: `"age"`},
			},
		}

		expected := `ALTER TABLE "users" ALTER COLUMN "age" DROP DEFAULT;` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,AlterColumnSetNotNull", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name: &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Action: &AlterColumnSetNotNull{
				Name: &Ident{Name: "age", QuotationMark: `"`, Raw: `"age"`},
			},
		}

		expected := `ALTER TABLE "users" ALTER COLUMN "age" SET NOT NULL;` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,AlterColumnDropNotNull", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name: &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Action: &AlterColumnDropNotNull{
				Name: &Ident{Name: "age", QuotationMark: `"`, Raw: `"age"`},
			},
		}

		expected := `ALTER TABLE "users" ALTER COLUMN "age" DROP NOT NULL;` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,AddConstraint", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name: &ObjectName{Name: &Ident{Name: "groups", QuotationMark: `"`, Raw: `"groups"`}},
			Action: &AddConstraint{
				Constraint: &PrimaryKeyConstraint{
					Name: &Ident{Name: "groups_pkey", QuotationMark: `"`, Raw: `"groups_pkey"`},
					Columns: []*ColumnIdent{
						{Ident: &Ident{Name: "id", QuotationMark: `"`, Raw: `"id"`}},
					},
				},
			},
		}

		expected := `ALTER TABLE "groups" ADD CONSTRAINT "groups_pkey" PRIMARY KEY ("id");` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,DropConstraint", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name:   &ObjectName{Name: &Ident{Name: "groups", QuotationMark: `"`, Raw: `"groups"`}},
			Action: &DropConstraint{Name: &Ident{Name: "groups_pkey", QuotationMark: `"`, Raw: `"groups_pkey"`}},
		}

		expected := `ALTER TABLE "groups" DROP CONSTRAINT "groups_pkey";` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,AlterConstraint,DEFERRABLE", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name: &ObjectName{Name: &Ident{Name: "groups", QuotationMark: `"`, Raw: `"groups"`}},
			Action: &AlterConstraint{
				Name:              &Ident{Name: "groups_pkey", QuotationMark: `"`, Raw: `"groups_pkey"`},
				Deferrable:        true,
				InitiallyDeferred: true,
			},
		}

		expected := `ALTER TABLE "groups" ALTER CONSTRAINT "groups_pkey" DEFERRABLE INITIALLY DEFERRED;` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,AlterConstraint,NOT_DEFERRABLE", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name: &ObjectName{Name: &Ident{Name: "groups", QuotationMark: `"`, Raw: `"groups"`}},
			Action: &AlterConstraint{
				Name:              &Ident{Name: "groups_pkey", QuotationMark: `"`, Raw: `"groups_pkey"`},
				Deferrable:        false,
				InitiallyDeferred: false,
			},
		}

		expected := `ALTER TABLE "groups" ALTER CONSTRAINT "groups_pkey" NOT DEFERRABLE INITIALLY IMMEDIATE;` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})
}

func TestAlterTableStmt_GetNameForDiff(t *testing.T) {
	t.Parallel()

	stmt := &AlterTableStmt{Name: &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}}}

	expected := `users`
	actual := stmt.GetNameForDiff()

	require.Equal(t, expected, actual)
}
