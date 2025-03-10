package mysql

import (
	"testing"

	require "github.com/hakadoriya/z.go/testingz/requirez"
)

func TestDropIndexStmt_GetNameForDiff(t *testing.T) {
	t.Parallel()

	t.Run("success,", func(t *testing.T) {
		t.Parallel()

		stmt := &DropIndexStmt{Name: &ObjectName{Name: &Ident{Name: "test", QuotationMark: `"`, Raw: `"test"`}}}
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
			IfExists: true,
			Name:     &ObjectName{Name: &Ident{Name: "test", QuotationMark: `"`, Raw: `"test"`}},
		}
		expected := `DROP INDEX IF EXISTS "test";` + "\n"
		actual := stmt.String()

		require.Equal(t, expected, actual)

		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})
}
