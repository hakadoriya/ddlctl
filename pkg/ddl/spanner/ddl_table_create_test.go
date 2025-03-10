package spanner

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
				{Name: &Ident{Name: "id", Raw: "id"}, DataType: &DataType{Name: "INT64"}},
				{Name: &Ident{Name: "name", Raw: "name"}, DataType: &DataType{Name: "STRING", Expr: &Expr{Idents: []*Ident{NewRawIdent("255")}}}},
				{Name: &Ident{Name: "created_at", Raw: "created_at"}, DataType: &DataType{Name: "TIMESTAMP"}, NotNull: true, Options: &Expr{Idents: []*Ident{
					NewRawIdent("("),
					NewRawIdent("allow_commit_timestamp"),
					NewRawIdent("="),
					NewRawIdent("true"),
					NewRawIdent(","),
					NewRawIdent("option_name"),
					NewRawIdent("="),
					NewRawIdent("null"),
					NewRawIdent(")"),
				}}},
			},
			Options: []*Option{
				{Name: "PRIMARY KEY", Value: &Expr{Idents: []*Ident{NewRawIdent("("), NewRawIdent("id"), NewRawIdent(")")}}},
			},
		}
		expected := `-- test comment content
CREATE TABLE test (
    id INT64,
    name STRING(255),
    created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true, option_name = null)
) PRIMARY KEY (id);
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
