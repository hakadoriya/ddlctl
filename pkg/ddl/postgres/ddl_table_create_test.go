package postgres

import (
	"testing"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
)

func TestCreateTableStmt_String(t *testing.T) {
	t.Parallel()

	t.Run("success,", func(t *testing.T) {
		t.Parallel()

		stmt := &CreateTableStmt{
			Comment: "test comment content",
			Indent:  "  ",
			Name:    &ObjectName{Name: &Ident{Name: "test", QuotationMark: `"`, Raw: `"test"`}},
			Columns: []*Column{
				{Name: &Ident{Name: "id", Raw: "id"}, DataType: &DataType{Name: "INTEGER"}},
				{Name: &Ident{Name: "name", Raw: "name"}, DataType: &DataType{Name: "VARYING", Expr: &Expr{Idents: []*Ident{NewRawIdent("255")}}}},
			},
			Options: []*Option{
				{Name: "TABLESPACE", Value: &Ident{Name: "default_tablespace", Raw: "default_tablespace"}},
				{Name: "LIKE", Value: &Ident{Name: "parent_test", Raw: "parent_test"}},
			},
		}

		expected := `-- test comment content
CREATE TABLE "test" (
    id INTEGER,
    name VARYING(255)
)
TABLESPACE default_tablespace,
LIKE parent_test;
`
		actual := stmt.String()
		assert.Equal(t, expected, actual)

		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})
}

func TestCreateTableStmt_GetNameForDiff(t *testing.T) {
	t.Parallel()

	t.Run("success,", func(t *testing.T) {
		t.Parallel()

		stmt := &CreateTableStmt{Name: &ObjectName{Name: &Ident{Name: "test", QuotationMark: `"`, Raw: `"test"`}}}
		expected := "test"
		actual := stmt.GetNameForDiff()

		assert.Equal(t, expected, actual)

		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})
}
