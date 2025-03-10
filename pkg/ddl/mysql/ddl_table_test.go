package mysql

import (
	"testing"

	require "github.com/hakadoriya/z.go/testingz/requirez"
)

func Test_isConstraint(t *testing.T) {
	t.Parallel()

	(&PrimaryKeyConstraint{}).isConstraint()
	(&ForeignKeyConstraint{}).isConstraint()
	(&IndexConstraint{}).isConstraint()
	(&CheckConstraint{}).isConstraint()
}

func TestConstraints_Append(t *testing.T) {
	t.Parallel()

	t.Run("success,Constraints,Append", func(t *testing.T) {
		t.Parallel()

		constraints := Constraints{}
		constraint := &PrimaryKeyConstraint{Name: &Ident{Name: "pk_users", QuotationMark: `"`, Raw: `"pk_users"`}, Columns: []*ColumnIdent{{Ident: &Ident{Name: "id", QuotationMark: `"`, Raw: `"id"`}}}}
		constraints = constraints.Append(constraint)
		constraints = constraints.Append(constraint)
		expected := Constraints{constraint}
		actual := constraints
		require.Equal(t, expected, actual)
	})
}

func TestPrimaryKeyConstraint(t *testing.T) {
	t.Parallel()

	t.Run("success,PrimaryKeyConstraint", func(t *testing.T) {
		t.Parallel()

		primaryKeyConstraint := &PrimaryKeyConstraint{Name: &Ident{Name: "pk_users", QuotationMark: `"`, Raw: `"pk_users"`}, Columns: []*ColumnIdent{{Ident: &Ident{Name: "id", QuotationMark: `"`, Raw: `"id"`}}}}
		expected := "PRIMARY KEY (\"id\")"
		actual := primaryKeyConstraint.String()
		require.Equal(t, expected, actual)

		t.Logf("✅: %s: primaryKeyConstraint: %#v", t.Name(), primaryKeyConstraint)
	})
	t.Run("success,PrimaryKeyConstraint,empty", func(t *testing.T) {
		t.Parallel()

		primaryKeyConstraint := &PrimaryKeyConstraint{}
		expected := "PRIMARY KEY ()"
		actual := primaryKeyConstraint.String()
		require.Equal(t, expected, actual)

		t.Logf("✅: %s: primaryKeyConstraint: %#v", t.Name(), primaryKeyConstraint)
	})
}

func TestForeignKeyConstraint(t *testing.T) {
	t.Parallel()
	t.Run("success,ForeignKeyConstraint", func(t *testing.T) {
		t.Parallel()

		foreignKeyConstraint := &ForeignKeyConstraint{
			Name:       &Ident{Name: "fk_users_groups", QuotationMark: `"`, Raw: `"fk_users_groups"`},
			Columns:    []*ColumnIdent{{Ident: &Ident{Name: "group_id", QuotationMark: `"`, Raw: `"group_id"`}}},
			Ref:        &Ident{Name: "groups", QuotationMark: `"`, Raw: `"groups"`},
			RefColumns: []*ColumnIdent{{Ident: &Ident{Name: "id", QuotationMark: `"`, Raw: `"id"`}}},
		}

		expected := `CONSTRAINT "fk_users_groups" FOREIGN KEY ("group_id") REFERENCES "groups" ("id")`
		actual := foreignKeyConstraint.String()
		require.Equal(t, expected, actual)

		t.Logf("✅: %s: foreignKeyConstraint: %#v", t.Name(), foreignKeyConstraint)
	})
}

func TestUniqueConstraint(t *testing.T) {
	t.Parallel()
	t.Run("success,UniqueConstraint", func(t *testing.T) {
		t.Parallel()

		indexConstraint := &IndexConstraint{
			Unique:  true,
			Name:    &Ident{Name: "uq_users_email", QuotationMark: `"`, Raw: `"uq_users_email"`},
			Columns: []*ColumnIdent{{Ident: &Ident{Name: "email", QuotationMark: `"`, Raw: `"email"`}}},
		}

		expected := `UNIQUE KEY "uq_users_email" ("email")`
		actual := indexConstraint.String()
		require.Equal(t, expected, actual)

		t.Logf("✅: %s: uniqueConstraint: %#v", t.Name(), indexConstraint)
	})
}

func TestCheckConstraint(t *testing.T) {
	t.Parallel()
	t.Run("success,CheckConstraint", func(t *testing.T) {
		t.Parallel()

		checkConstraint := &CheckConstraint{
			Name: &Ident{Name: "users_check_age", QuotationMark: `"`, Raw: `"users_check_age"`},
			Expr: &Expr{Idents: []*Ident{{Name: "(", QuotationMark: ``, Raw: `(`}, {Name: "age", QuotationMark: `"`, Raw: `"age"`}, {Name: ">=", QuotationMark: ``, Raw: `>=`}, {Name: "0", QuotationMark: ``, Raw: `0`}}},
		}
		checkConstraint.Expr = checkConstraint.Expr.Append(&Ident{Name: ")", QuotationMark: ``, Raw: `)`})

		expected := `CONSTRAINT "users_check_age" CHECK ("age" >= 0)`
		actual := checkConstraint.String()
		require.Equal(t, expected, actual)

		t.Logf("✅: %s: checkConstraint: %#v", t.Name(), checkConstraint)
	})
}

func TestObjectName_StringForDiff(t *testing.T) {
	t.Parallel()

	t.Run("success,ObjectName", func(t *testing.T) {
		t.Parallel()

		objectName := &ObjectName{Schema: &Ident{Name: "public", QuotationMark: `"`, Raw: `"public"`}, Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}}
		expected := "public.users"
		actual := objectName.StringForDiff()
		require.Equal(t, expected, actual)

		t.Logf("✅: %s: objectName: %#v", t.Name(), objectName)
	})
	t.Run("success,ObjectName,empty", func(t *testing.T) {
		t.Parallel()

		objectName := (*ObjectName)(nil)
		expected := ""
		actual := objectName.StringForDiff()
		require.Equal(t, expected, actual)

		t.Logf("✅: %s: objectName: %#v", t.Name(), objectName)
	})
}

func TestExpr_String(t *testing.T) {
	t.Parallel()

	t.Run("success,String,nil", func(t *testing.T) {
		t.Parallel()

		d := (*Default)(nil)
		expected := ""
		actual := d.String()
		require.Equal(t, expected, actual)
	})
	t.Run("success,String,nilnil", func(t *testing.T) {
		t.Parallel()

		d := &Default{}
		expected := ""
		actual := d.String()
		require.Equal(t, expected, actual)
	})
	t.Run("success,PlainString,nilnil", func(t *testing.T) {
		t.Parallel()

		d := &Default{}
		expected := ""
		actual := d.StringForDiff()
		require.Equal(t, expected, actual)
	})
	t.Run("success,DEFAULT_VALUE", func(t *testing.T) {
		t.Parallel()

		d := &Default{Value: &Expr{[]*Ident{{Name: "now()", Raw: "now()"}}}}
		expected := "DEFAULT now()"
		actual := d.String()
		require.Equal(t, expected, actual)

		t.Logf("✅: %s: d: %#v", t.Name(), d)
	})
	t.Run("success,DEFAULT_VALUE,empty", func(t *testing.T) {
		t.Parallel()

		d := (*Expr)(nil)
		expected := ""
		actual := d.String()
		require.Equal(t, expected, actual)
	})
	t.Run("success,DEFAULT_EXPR", func(t *testing.T) {
		t.Parallel()

		d := &Default{Value: &Expr{[]*Ident{{Name: "(", Raw: "("}, {Name: "age", Raw: "age"}, {Name: ">=", Raw: ">="}, {Name: "0", Raw: "0"}, {Name: ")", Raw: ")"}}}}
		expected := "DEFAULT (age >= 0)"
		actual := d.String()
		require.Equal(t, expected, actual)

		t.Logf("✅: %s: d: %#v", t.Name(), d)
	})
}

func TestColumn(t *testing.T) {
	t.Parallel()

	t.Run("success,Column", func(t *testing.T) {
		t.Parallel()

		column := &Column{
			Name:     &Ident{Name: "id", QuotationMark: `"`, Raw: `"id"`},
			DataType: &DataType{Name: "INTEGER"},
		}

		expected := `"id" INTEGER NULL`
		actual := column.String()
		require.Equal(t, expected, actual)

		t.Logf("✅: %s: column: %#v", t.Name(), column)
	})
}

func TestOption(t *testing.T) {
	t.Parallel()

	t.Run("success,Option", func(t *testing.T) {
		t.Parallel()

		option := &Option{Name: "ENGINE", Value: &Expr{[]*Ident{NewRawIdent("InnoDB")}}}

		expected := `ENGINE=InnoDB`
		actual := option.String()
		require.Equal(t, expected, actual)

		t.Logf("✅: %s: option: %#v", t.Name(), option)
	})
	t.Run("success,Option", func(t *testing.T) {
		t.Parallel()

		option := &Option{Name: "KEY", Value: &Expr{[]*Ident{NewRawIdent("("), NewRawIdent("sample"), NewRawIdent("value"), NewRawIdent(")")}}}

		expected := `KEY=(sample value)`
		actual := option.String()
		require.Equal(t, expected, actual)

		expectedDiff := "KEY=( sample value )"
		actualDiff := option.StringForDiff()
		require.Equal(t, expectedDiff, actualDiff)

		t.Logf("✅: %s: option: %#v", t.Name(), option)
	})

	t.Run("success,Option,empty,1", func(t *testing.T) {
		t.Parallel()

		option := &Option{}
		expected := ""
		actual := option.String()
		require.Equal(t, expected, actual)

		actualDiff := option.StringForDiff()
		require.Equal(t, expected, actualDiff)

		t.Logf("✅: %s: option: %#v", t.Name(), option)
	})
}

func TestOptions(t *testing.T) {
	t.Parallel()

	t.Run("success,Option", func(t *testing.T) {
		t.Parallel()

		option := Options{
			{Name: "ENGINE", Value: &Expr{[]*Ident{NewRawIdent("InnoDB")}}},
			{Name: "CHARSET", Value: &Expr{[]*Ident{NewRawIdent("utf8mb4")}}},
		}

		expected := `ENGINE=InnoDB CHARSET=utf8mb4`
		actual := option.String()
		require.Equal(t, expected, actual)

		expectedDiff := "ENGINE=InnoDB CHARSET=utf8mb4"
		actualDiff := option.StringForDiff()
		require.Equal(t, expectedDiff, actualDiff)

		t.Logf("✅: %s: option: %#v", t.Name(), option)
	})
	t.Run("success,Option", func(t *testing.T) {
		t.Parallel()

		option := Options{{Name: "KEY", Value: &Expr{[]*Ident{NewRawIdent("("), NewRawIdent("sample"), NewRawIdent("value"), NewRawIdent(")")}}}}

		expected := `KEY=(sample value)`
		actual := option.String()
		require.Equal(t, expected, actual)

		expectedDiff := "KEY=( sample value )"
		actualDiff := option.StringForDiff()
		require.Equal(t, expectedDiff, actualDiff)

		t.Logf("✅: %s: option: %#v", t.Name(), option)
	})

	t.Run("success,Option,empty,1", func(t *testing.T) {
		t.Parallel()

		option := Options{{}}
		expected := ""
		actual := option.String()
		require.Equal(t, expected, actual)

		actualDiff := option.StringForDiff()
		require.Equal(t, expected, actualDiff)

		t.Logf("✅: %s: option: %#v", t.Name(), option)
	})
}
