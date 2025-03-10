package spanner

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
	(&AlterColumnDataType{}).isAlterTableAction()
	(&AlterColumnSetDefault{}).isAlterTableAction()
	(&AlterColumnDropDefault{}).isAlterTableAction()
	(&AlterColumnSetOptions{}).isAlterTableAction()
	(&AlterColumnDropOptions{}).isAlterTableAction()
	(&AddConstraint{}).isAlterTableAction()
	(&DropConstraint{}).isAlterTableAction()
	(&AlterConstraint{}).isAlterTableAction()
	(&AddRowDeletionPolicy{}).isAlterTableAction()
	(&ReplaceRowDeletionPolicy{}).isAlterTableAction()
	(&DropRowDeletionPolicy{}).isAlterTableAction()
}

func TestAlterTableStmt_String(t *testing.T) {
	t.Parallel()

	t.Run("success,RenameTable", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name: &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Action: &RenameTable{
				NewName: &ObjectName{Name: &Ident{Name: "accounts", QuotationMark: `"`, Raw: `"accounts"`}},
			},
		}

		expected := `ALTER TABLE "users" RENAME TO "accounts";` + "\n"
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

	t.Run("success,AlterColumnDataType", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name: &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Action: &AlterColumnDataType{
				Name:     &Ident{Name: "age", QuotationMark: `"`, Raw: `"age"`},
				DataType: &DataType{Name: "INT64"},
			},
		}

		expected := `ALTER TABLE "users" ALTER COLUMN "age" INT64;` + "\n"
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

	t.Run("success,AddConstraint", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name: &ObjectName{Name: &Ident{Name: "groups", QuotationMark: `"`, Raw: `"groups"`}},
			Action: &AddConstraint{
				Constraint: &CheckConstraint{
					Name: &Ident{Name: "groups_yyyymmdd_chk", QuotationMark: `"`, Raw: `"groups_yyyymmdd_chk"`},
					Expr: &Expr{Idents: []*Ident{
						NewRawIdent("("),
						NewRawIdent(`"yyyymmdd"`),
						NewRawIdent(">"),
						NewRawIdent("0"),
						NewRawIdent(")"),
					}},
				},
			},
		}

		expected := `ALTER TABLE "groups" ADD CONSTRAINT "groups_yyyymmdd_chk" CHECK ("yyyymmdd" > 0);` + "\n"
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

	t.Run("success,AddRowDeletionPolicy", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name: &ObjectName{Name: &Ident{Name: "groups", QuotationMark: `"`, Raw: `"groups"`}},
			Action: &AddRowDeletionPolicy{RowDeletionPolicy: &Option{Name: "ROW DELETION POLICY", Value: &Expr{Idents: []*Ident{
				NewRawIdent("("),
				NewRawIdent("OLDER_THAN"),
				NewRawIdent("("),
				NewRawIdent("CreatedAt"),
				NewRawIdent(","),
				NewRawIdent("INTERVAL"),
				NewRawIdent("30"),
				NewRawIdent("DAY"),
				NewRawIdent(")"),
				NewRawIdent(")"),
			}}}},
		}

		expected := `ALTER TABLE "groups" ADD ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 30 DAY));` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,ReplaceRowDeletionPolicy", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name: &ObjectName{Name: &Ident{Name: "groups", QuotationMark: `"`, Raw: `"groups"`}},
			Action: &ReplaceRowDeletionPolicy{RowDeletionPolicy: &Option{Name: "ROW DELETION POLICY", Value: &Expr{Idents: []*Ident{
				NewRawIdent("("),
				NewRawIdent("OLDER_THAN"),
				NewRawIdent("("),
				NewRawIdent("CreatedAt"),
				NewRawIdent(","),
				NewRawIdent("INTERVAL"),
				NewRawIdent("30"),
				NewRawIdent("DAY"),
				NewRawIdent(")"),
				NewRawIdent(")"),
			}}}},
		}

		expected := `ALTER TABLE "groups" REPLACE ROW DELETION POLICY (OLDER_THAN(CreatedAt, INTERVAL 30 DAY));` + "\n"
		actual := stmt.String()

		if !assert.Equal(t, expected, actual) {
			assert.Equal(t, fmt.Sprintf("%#v", expected), fmt.Sprintf("%#v", actual))
		}
		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,DropRowDeletionPolicy", func(t *testing.T) {
		t.Parallel()

		stmt := &AlterTableStmt{
			Name:   &ObjectName{Name: &Ident{Name: "groups", QuotationMark: `"`, Raw: `"groups"`}},
			Action: &DropRowDeletionPolicy{},
		}

		expected := `ALTER TABLE "groups" DROP ROW DELETION POLICY;` + "\n"
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
