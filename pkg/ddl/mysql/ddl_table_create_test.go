package mysql

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
			Name:    &ObjectName{Name: &Ident{Name: "test", Raw: "test"}},
			Columns: []*Column{
				{Name: &Ident{Name: "id", Raw: "id"}, DataType: &DataType{Name: "INTEGER"}},
				{Name: &Ident{Name: "name", Raw: "name"}, DataType: &DataType{Name: "VARYING", Expr: &Expr{[]*Ident{{Name: "255", Raw: "255"}}}}},
			},
			Options: []*Option{
				{Name: "ENGINE", Value: &Expr{[]*Ident{NewRawIdent("InnoDB")}}},
				{Name: "DEFAULT CHARSET", Value: &Expr{[]*Ident{NewRawIdent("utf8mb4")}}},
				{Name: "COLLATE", Value: &Expr{[]*Ident{NewRawIdent("utf8mb4_0900_ai_ci")}}},
			},
		}
		expected := `-- test comment content
CREATE TABLE test (
    id INTEGER NULL,
    name VARYING(255) NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
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
