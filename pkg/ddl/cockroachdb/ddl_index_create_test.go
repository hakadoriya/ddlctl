package cockroachdb

import (
	"testing"

	"github.com/hakadoriya/z.go/testingz/requirez"
)

func TestCreateIndexStmt_GetNameForDiff(t *testing.T) {
	t.Parallel()

	t.Run("success,", func(t *testing.T) {
		t.Parallel()

		stmt := &CreateIndexStmt{Name: &Ident{Name: "test", QuotationMark: `"`, Raw: `"test"`}}
		expected := "test"
		actual := stmt.GetNameForDiff()

		requirez.Equal(t, expected, actual)
	})
}

func TestCreateIndexStmt_String(t *testing.T) {
	t.Parallel()

	t.Run("success,", func(t *testing.T) {
		t.Parallel()

		stmt := &CreateIndexStmt{
			Comment:         "test comment content",
			IfNotExists:     true,
			Name:            &Ident{Name: "test", QuotationMark: `"`, Raw: `"test"`},
			TableName:       &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			UsingPreColumns: &Using{Value: &Expr{Idents: []*Ident{{Name: "btree", QuotationMark: ``, Raw: `btree`}}}},
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

		requirez.Equal(t, expected, actual)

		t.Logf("✅: %s: stmt: %#v", t.Name(), stmt)
	})

	t.Run("success,UniqueMultiColumnsUsingPostColumns", func(t *testing.T) {
		t.Parallel()

		// 複数行コメント (空行を含む) ・ UNIQUE ・複数カラム・カラム指定後の USING を出力できること
		stmt := &CreateIndexStmt{
			Comment:   "test comment line 1\n\ntest comment line 2",
			Unique:    true,
			Name:      &Ident{Name: "test", QuotationMark: `"`, Raw: `"test"`},
			TableName: &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Columns: []*ColumnIdent{
				{Ident: &Ident{Name: "id", QuotationMark: `"`, Raw: `"id"`}, Order: &Order{Desc: false}},
				{Ident: &Ident{Name: "name", QuotationMark: `"`, Raw: `"name"`}, Order: &Order{Desc: true}},
			},
			UsingPostColumns: &Using{
				Value: &Expr{Idents: []*Ident{{Name: "hash", Raw: `hash`}}},
				With:  &With{Value: &Expr{Idents: []*Ident{{Name: "bucket_count", Raw: `bucket_count`}, {Name: "=", Raw: `=`}, {Name: "8", Raw: `8`}}}},
			},
		}
		expected := `-- test comment line 1
-- test comment line 2
CREATE UNIQUE INDEX "test" ON "users" ("id" ASC, "name" DESC) USING hash WITH bucket_count = 8;
`
		actual := stmt.String()

		requirez.Equal(t, expected, actual)
	})
}

func TestCreateIndexStmt_StringForDiff(t *testing.T) {
	t.Parallel()

	t.Run("success,", func(t *testing.T) {
		t.Parallel()

		stmt := &CreateIndexStmt{
			Name:      &Ident{Name: "test", QuotationMark: `"`, Raw: `"test"`},
			TableName: &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			Columns: []*ColumnIdent{
				{Ident: &Ident{Name: "id", QuotationMark: `"`, Raw: `"id"`}},
			},
		}
		expected := "CREATE INDEX test ON users (id ASC);\n"
		actual := stmt.StringForDiff()

		requirez.Equal(t, expected, actual)
	})

	t.Run("success,UniqueMultiColumnsUsingPreAndPostColumns", func(t *testing.T) {
		t.Parallel()

		// UNIQUE ・複数カラム・カラム指定前後の USING を出力できること
		stmt := &CreateIndexStmt{
			Unique:          true,
			Name:            &Ident{Name: "test", QuotationMark: `"`, Raw: `"test"`},
			TableName:       &ObjectName{Name: &Ident{Name: "users", QuotationMark: `"`, Raw: `"users"`}},
			UsingPreColumns: &Using{Value: &Expr{Idents: []*Ident{{Name: "btree", Raw: `btree`}}}},
			Columns: []*ColumnIdent{
				{Ident: &Ident{Name: "id", QuotationMark: `"`, Raw: `"id"`}},
				{Ident: &Ident{Name: "name", QuotationMark: `"`, Raw: `"name"`}, Order: &Order{Desc: true}},
			},
			UsingPostColumns: &Using{
				Value: &Expr{Idents: []*Ident{{Name: "hash", Raw: `hash`}}},
				With:  &With{Value: &Expr{Idents: []*Ident{{Name: "bucket_count", Raw: `bucket_count`}, {Name: "=", Raw: `=`}, {Name: "8", Raw: `8`}}}},
			},
		}
		expected := "CREATE UNIQUE INDEX test ON users USING btree (id ASC, name DESC) USING hash WITH bucket_count = 8;\n"
		actual := stmt.StringForDiff()

		requirez.Equal(t, expected, actual)
	})
}
