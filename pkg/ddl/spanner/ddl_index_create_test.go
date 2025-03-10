package spanner

import (
	"testing"

	require "github.com/hakadoriya/z.go/testingz/requirez"
)

func TestCreateIndexStmt_GetNameForDiff(t *testing.T) {
	t.Parallel()

	t.Run("success,", func(t *testing.T) {
		t.Parallel()

		stmt := &CreateIndexStmt{Name: &ObjectName{Name: &Ident{Name: "test", QuotationMark: `"`, Raw: `"test"`}}}
		expected := "test"
		actual := stmt.GetNameForDiff()

		require.Equal(t, expected, actual)
	})
}

func TestCreateIndexStmt_String(t *testing.T) {
	t.Parallel()

	t.Run("success,", func(t *testing.T) {
		t.Parallel()

		stmt := &CreateIndexStmt{
			Comment:     "test comment content",
			IfNotExists: true,
			Name:        &ObjectName{Name: &Ident{Name: "test", QuotationMark: `"`, Raw: `"test"`}},
			TableName:   &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Using:       []*Ident{{Name: "btree", QuotationMark: ``, Raw: `btree`}},
			Columns: []*ColumnIdent{
				{
					Ident: &Ident{Name: "id", QuotationMark: `"`, Raw: `"id"`},
					Order: &Order{Desc: false},
				},
			},
		}
		expected := `-- test comment content
CREATE INDEX IF NOT EXISTS "test" ON "users" USING btree ("id" ASC);
`
		actual := stmt.String()

		require.Equal(t, expected, actual)

		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})
}
