package mysql

import (
	"testing"

	require "github.com/hakadoriya/z.go/testingz/requirez"
)

func Test_isStmt(t *testing.T) {
	t.Parallel()

	(&CreateTableStmt{}).isStmt()
	(&DropTableStmt{}).isStmt()
	(&AlterTableStmt{}).isStmt()
	(&CreateIndexStmt{}).isStmt()
	(&DropIndexStmt{}).isStmt()
}

func TestIdent_String(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ident := &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}
		expected := ident.Raw
		actual := ident.String()

		require.Equal(t, expected, actual)

		t.Logf("✅: %s: ident: %#v", t.Name(), ident)
	})

	t.Run("success,empty", func(t *testing.T) {
		t.Parallel()

		ident := (*Ident)(nil)
		expected := ""
		actual := ident.String()

		require.Equal(t, expected, actual)

		t.Logf("✅: %s: ident: %#v", t.Name(), ident)
	})
}

func TestIdent_StringForDiff(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ident := &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}
		expected := ident.Name
		actual := ident.StringForDiff()

		require.Equal(t, expected, actual)
	})

	t.Run("success,empty", func(t *testing.T) {
		t.Parallel()
		ident := (*Ident)(nil)
		expected := ""
		actual := ident.StringForDiff()

		require.Equal(t, expected, actual)
	})
}

func TestDataType_StringForDiff(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		dataType := &DataType{Name: "integer", Type: TOKEN_INTEGER}
		expected := string(TOKEN_INTEGER)
		actual := dataType.StringForDiff()

		require.Equal(t, expected, actual)
	})

	t.Run("success,nil", func(t *testing.T) {
		t.Parallel()
		dataType := (*DataType)(nil)
		expected := ""
		actual := dataType.StringForDiff()

		require.Equal(t, expected, actual)
	})

	t.Run("success,TOKEN_ILLEGAL", func(t *testing.T) {
		t.Parallel()
		dataType := &DataType{Name: "unknown", Type: TOKEN_ILLEGAL}
		expected := string(TOKEN_ILLEGAL)
		actual := dataType.StringForDiff()

		require.Equal(t, expected, actual)
	})

	t.Run("success,empty", func(t *testing.T) {
		t.Parallel()
		dataType := &DataType{Name: "unknown", Type: ""}
		expected := string(TOKEN_ILLEGAL)
		actual := dataType.StringForDiff()

		require.Equal(t, expected, actual)
	})
}
