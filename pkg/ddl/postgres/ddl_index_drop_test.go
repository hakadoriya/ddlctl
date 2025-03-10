package postgres

import (
	"testing"

	require "github.com/hakadoriya/z.go/testingz/requirez"
)

func TestDropIndexStmt_GetNameForDiff(t *testing.T) {
	t.Parallel()

	t.Run("success,", func(t *testing.T) {
		t.Parallel()

		stmt := &DropIndexStmt{Name: &Ident{Name: "test", QuotationMark: `"`, Raw: `"test"`}}
		expected := "test"
		actual := stmt.GetNameForDiff()

		require.Equal(t, expected, actual)
	})
}

func TestDropIndexStmt_String(t *testing.T) {
	t.Parallel()

	t.Run("success,", func(t *testing.T) {
		t.Parallel()

		stmt := &DropIndexStmt{
			Comment:  "test comment content",
			IfExists: true,
			Name:     &Ident{Name: "test", QuotationMark: `"`, Raw: `"test"`},
		}
		expected := `-- test comment content
DROP INDEX IF EXISTS "test";
`
		actual := stmt.String()

		require.Equal(t, expected, actual)

		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})
}
