package spanner

import (
	"testing"

	require "github.com/hakadoriya/z.go/testingz/requirez"
)

func Test_isConstraint(t *testing.T) {
	t.Parallel()

	(&ForeignKeyConstraint{}).isConstraint()
	(&CheckConstraint{}).isConstraint()
}

func TestConstraints_Append(t *testing.T) {
	t.Parallel()

	t.Run("success,Constraints,Append", func(t *testing.T) {
		t.Parallel()

		constraints := Constraints{}
		constraint := &CheckConstraint{
			Name: NewRawIdent(`"users_age_check"`),
			Expr: &Expr{Idents: []*Ident{
				{Name: "(", QuotationMark: ``, Raw: `(`},
				{Name: "age", QuotationMark: `"`, Raw: `"age"`},
				{Name: ">=", QuotationMark: ``, Raw: `>=`},
				{Name: "0", QuotationMark: ``, Raw: `0`},
				{Name: ")", QuotationMark: ``, Raw: `)`},
			}},
		}
		constraints = constraints.Append(constraint)
		constraints = constraints.Append(constraint)
		expected := Constraints{constraint}
		actual := constraints
		require.Equal(t, expected, actual)
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

func TestCheckConstraint(t *testing.T) {
	t.Parallel()
	t.Run("success,CheckConstraint", func(t *testing.T) {
		t.Parallel()

		checkConstraint := &CheckConstraint{
			Name: &Ident{Name: "users_check_age", QuotationMark: `"`, Raw: `"users_check_age"`},
			Expr: &Expr{Idents: []*Ident{{Name: "(", QuotationMark: ``, Raw: `(`}, {Name: "age", QuotationMark: `"`, Raw: `"age"`}, {Name: ">=", QuotationMark: ``, Raw: `>=`}, {Name: "0", QuotationMark: ``, Raw: `0`}, {Name: ")", QuotationMark: ``, Raw: `)`}}},
		}

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

		expected := `"id" INTEGER`
		actual := column.String()
		require.Equal(t, expected, actual)

		t.Logf("✅: %s: column: %#v", t.Name(), column)
	})
}

func TestOption(t *testing.T) {
	t.Parallel()

	t.Run("success,Option", func(t *testing.T) {
		t.Parallel()

		option := &Option{Name: "PRIMARY KEY", Value: &Expr{Idents: []*Ident{NewRawIdent("("), NewRawIdent(`"id1"`), NewRawIdent(`,`), NewRawIdent(`"id2"`), NewRawIdent(")")}}}

		expected := `PRIMARY KEY ("id1", "id2")`
		actual := option.String()
		require.Equal(t, expected, actual)

		expectedForDiff := `PRIMARY KEY ( id1 , id2 )`
		actualForDiff := option.StringForDiff()
		require.Equal(t, expectedForDiff, actualForDiff)

		t.Logf("✅: %s: option: %#v", t.Name(), option)
	})

	t.Run("success,Options", func(t *testing.T) {
		t.Parallel()

		options := Options{
			&Option{Name: "PRIMARY KEY", Value: &Expr{Idents: []*Ident{NewRawIdent("("), NewRawIdent(`"id1"`), NewRawIdent(`,`), NewRawIdent(`"id2"`), NewRawIdent(")")}}},
			&Option{Name: "PRIMARY KEY", Value: &Expr{Idents: []*Ident{NewRawIdent("("), NewRawIdent(`"id1"`), NewRawIdent(`,`), NewRawIdent(`"id2"`), NewRawIdent(")")}}},
		}

		expected := `PRIMARY KEY ("id1", "id2"),
PRIMARY KEY ("id1", "id2")`
		actual := options.String()
		require.Equal(t, expected, actual)

		expectedForDiff := `PRIMARY KEY ( id1 , id2 ), PRIMARY KEY ( id1 , id2 )`
		actualForDiff := options.StringForDiff()
		require.Equal(t, expectedForDiff, actualForDiff)

		t.Logf("✅: %s: option: %#v", t.Name(), options)
	})

	t.Run("success,Option,empty", func(t *testing.T) {
		t.Parallel()

		option := &Option{}
		expectedString := ""
		actualString := option.String()
		require.Equal(t, expectedString, actualString)

		expectedStringForDiff := ""
		actualStringForDiff := option.StringForDiff()
		require.Equal(t, expectedStringForDiff, actualStringForDiff)

		t.Logf("✅: %s: option: %#v", t.Name(), option)
	})
}
