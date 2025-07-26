package postgres

import (
	"testing"

	require "github.com/hakadoriya/z.go/testingz/requirez"
)

func Test_lookupIdent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  TokenType
	}{
		{name: "success,EQUAL", input: "EQUAL", want: TOKEN_EQUAL},
		{name: "success,GREATER", input: "GREATER", want: TOKEN_GREATER},
		{name: "success,LESS", input: "LESS", want: TOKEN_LESS},
		{name: "success,CREATE", input: "CREATE", want: TOKEN_CREATE},
		{name: "success,ALTER", input: "ALTER", want: TOKEN_ALTER},
		{name: "success,DROP", input: "DROP", want: TOKEN_DROP},
		{name: "success,RENAME", input: "RENAME", want: TOKEN_RENAME},
		{name: "success,CREATE", input: "CREATE", want: TOKEN_CREATE},
		{name: "success,ALTER", input: "ALTER", want: TOKEN_ALTER},
		{name: "success,DROP", input: "DROP", want: TOKEN_DROP},
		{name: "success,RENAME", input: "RENAME", want: TOKEN_RENAME},
		{name: "success,TRUNCATE", input: "TRUNCATE", want: TOKEN_TRUNCATE},
		{name: "success,DELETE", input: "DELETE", want: TOKEN_DELETE},
		{name: "success,UPDATE", input: "UPDATE", want: TOKEN_UPDATE},
		{name: "success,TABLE", input: "TABLE", want: TOKEN_TABLE},
		{name: "success,INDEX", input: "INDEX", want: TOKEN_INDEX},
		{name: "success,VIEW", input: "VIEW", want: TOKEN_VIEW},
		{name: "success,IF", input: "IF", want: TOKEN_IF},
		{name: "success,EXISTS", input: "EXISTS", want: TOKEN_EXISTS},
		{name: "success,ON", input: "ON", want: TOKEN_ON},
		{name: "success,TO", input: "TO", want: TOKEN_TO},
		{name: "success,BOOLEAN", input: "BOOLEAN", want: TOKEN_BOOLEAN},
		{name: "success,SMALLINT", input: "SMALLINT", want: TOKEN_SMALLINT},
		{name: "success,INTEGER", input: "INTEGER", want: TOKEN_INTEGER},
		{name: "success,INT", input: "INT", want: TOKEN_INTEGER},
		{name: "success,BIGINT", input: "BIGINT", want: TOKEN_BIGINT},
		{name: "success,DECIMAL", input: "DECIMAL", want: TOKEN_DECIMAL},
		{name: "success,NUMERIC", input: "NUMERIC", want: TOKEN_NUMERIC},
		{name: "success,REAL", input: "REAL", want: TOKEN_REAL},
		{name: "success,DOUBLE", input: "DOUBLE", want: TOKEN_DOUBLE},
		{name: "success,PRECISION", input: "PRECISION", want: TOKEN_PRECISION},
		{name: "success,SMALLSERIAL", input: "SMALLSERIAL", want: TOKEN_SMALLSERIAL},
		{name: "success,SERIAL", input: "SERIAL", want: TOKEN_SERIAL},
		{name: "success,TOKEN_BIGSERIAL", input: "BIGSERIAL", want: TOKEN_BIGSERIAL},
		{name: "success,UUID", input: "UUID", want: TOKEN_UUID},
		{name: "success,JSONB", input: "JSONB", want: TOKEN_JSONB},
		{name: "success,CHARACTER", input: "CHARACTER", want: TOKEN_CHARACTER},
		{name: "success,VARYING", input: "VARYING", want: TOKEN_VARYING},
		{name: "success,VARCHAR", input: "VARCHAR", want: TOKEN_VARCHAR},
		{name: "success,TEXT", input: "TEXT", want: TOKEN_TEXT},
		{name: "success,TIMESTAMP", input: "TIMESTAMP", want: TOKEN_TIMESTAMP},
		{name: "success,TIMESTAMPTZ", input: "TIMESTAMPTZ", want: TOKEN_TIMESTAMPTZ},
		{name: "success,WITH", input: "WITH", want: TOKEN_WITH},
		{name: "success,TIME", input: "TIME", want: TOKEN_TIME},
		{name: "success,ZONE", input: "ZONE", want: TOKEN_ZONE},
		{name: "success,DEFAULT", input: "DEFAULT", want: TOKEN_DEFAULT},
		{name: "success,NOT", input: "NOT", want: TOKEN_NOT},
		{name: "success,NULL", input: "NULL", want: TOKEN_NULL},
		{name: "success,ASC", input: "ASC", want: TOKEN_ASC},
		{name: "success,DESC", input: "DESC", want: TOKEN_DESC},
		{name: "success,CASCADE", input: "CASCADE", want: TOKEN_CASCADE},
		{name: "success,NO", input: "NO", want: TOKEN_NO},
		{name: "success,ACTION", input: "ACTION", want: TOKEN_ACTION},
		{name: "success,CONSTRAINT", input: "CONSTRAINT", want: TOKEN_CONSTRAINT},
		{name: "success,PRIMARY", input: "PRIMARY", want: TOKEN_PRIMARY},
		{name: "success,KEY", input: "KEY", want: TOKEN_KEY},
		{name: "success,FOREIGN", input: "FOREIGN", want: TOKEN_FOREIGN},
		{name: "success,REFERENCES", input: "REFERENCES", want: TOKEN_REFERENCES},
		{name: "success,UNIQUE", input: "UNIQUE", want: TOKEN_UNIQUE},
		{name: "success,CHECK", input: "CHECK", want: TOKEN_CHECK},
		{name: "success,NULLIF", input: "NULLIF", want: TOKEN_NULLIF},
		{name: "success,IDENT", input: "users", want: TOKEN_IDENT},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := lookupIdent(tt.input)

			if !require.Equal(t, tt.want, got) {
				t.FailNow()
			}
		})
	}
}

func TestLex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []Token
	}{
		{
			name: "success,CREATE_TABLE",
			input: `CREATE TABLE IF NOT EXISTS "users" (
    "user_id"    UUID         NOT NULL,
    "name"       VARCHAR(255) NOT NULL,
    "email"      VARCHAR(255) NOT NULL,
    "password"   VARCHAR(255) NOT NULL,
    "created_at" TIMESTAMPTZ   NOT NULL,
    "updated_at" TIMESTAMPTZ   NOT NULL,
    PRIMARY KEY ("user_id"),
    UNIQUE ("email")
);`,
			want: []Token{
				{Type: TOKEN_CREATE, Literal: Literal{Str: "CREATE"}},
				{Type: TOKEN_TABLE, Literal: Literal{Str: "TABLE"}},
				{Type: TOKEN_IF, Literal: Literal{Str: "IF"}},
				{Type: TOKEN_NOT, Literal: Literal{Str: "NOT"}},
				{Type: TOKEN_EXISTS, Literal: Literal{Str: "EXISTS"}},
				{Type: TOKEN_IDENT, Literal: Literal{Str: `"users"`}},
				{Type: TOKEN_OPEN_PAREN, Literal: Literal{Str: "("}},
				{Type: TOKEN_IDENT, Literal: Literal{Str: `"user_id"`}},
				{Type: TOKEN_UUID, Literal: Literal{Str: "UUID"}},
				{Type: TOKEN_NOT, Literal: Literal{Str: "NOT"}},
				{Type: TOKEN_NULL, Literal: Literal{Str: "NULL"}},
				{Type: TOKEN_COMMA, Literal: Literal{Str: ","}},
				{Type: TOKEN_IDENT, Literal: Literal{Str: `"name"`}},
				{Type: TOKEN_VARCHAR, Literal: Literal{Str: "VARCHAR"}},
				{Type: TOKEN_OPEN_PAREN, Literal: Literal{Str: "("}},
				{Type: TOKEN_IDENT, Literal: Literal{Str: "255"}},
				{Type: TOKEN_CLOSE_PAREN, Literal: Literal{Str: ")"}},
				{Type: TOKEN_NOT, Literal: Literal{Str: "NOT"}},
				{Type: TOKEN_NULL, Literal: Literal{Str: "NULL"}},
				{Type: TOKEN_COMMA, Literal: Literal{Str: ","}},
				{Type: TOKEN_IDENT, Literal: Literal{Str: `"email"`}},
				{Type: TOKEN_VARCHAR, Literal: Literal{Str: "VARCHAR"}},
				{Type: TOKEN_OPEN_PAREN, Literal: Literal{Str: "("}},
				{Type: TOKEN_IDENT, Literal: Literal{Str: "255"}},
				{Type: TOKEN_CLOSE_PAREN, Literal: Literal{Str: ")"}},
				{Type: TOKEN_NOT, Literal: Literal{Str: "NOT"}},
				{Type: TOKEN_NULL, Literal: Literal{Str: "NULL"}},
				{Type: TOKEN_COMMA, Literal: Literal{Str: ","}},
				{Type: TOKEN_IDENT, Literal: Literal{Str: `"password"`}},
				{Type: TOKEN_VARCHAR, Literal: Literal{Str: "VARCHAR"}},
				{Type: TOKEN_OPEN_PAREN, Literal: Literal{Str: "("}},
				{Type: TOKEN_IDENT, Literal: Literal{Str: "255"}},
				{Type: TOKEN_CLOSE_PAREN, Literal: Literal{Str: ")"}},
				{Type: TOKEN_NOT, Literal: Literal{Str: "NOT"}},
				{Type: TOKEN_NULL, Literal: Literal{Str: "NULL"}},
				{Type: TOKEN_COMMA, Literal: Literal{Str: ","}},
				{Type: TOKEN_IDENT, Literal: Literal{Str: `"created_at"`}},
				{Type: TOKEN_TIMESTAMPTZ, Literal: Literal{Str: "TIMESTAMPTZ"}},
				{Type: TOKEN_NOT, Literal: Literal{Str: "NOT"}},
				{Type: TOKEN_NULL, Literal: Literal{Str: "NULL"}},
				{Type: TOKEN_COMMA, Literal: Literal{Str: ","}},
				{Type: TOKEN_IDENT, Literal: Literal{Str: `"updated_at"`}},
				{Type: TOKEN_TIMESTAMPTZ, Literal: Literal{Str: "TIMESTAMPTZ"}},
				{Type: TOKEN_NOT, Literal: Literal{Str: "NOT"}},
				{Type: TOKEN_NULL, Literal: Literal{Str: "NULL"}},
				{Type: TOKEN_COMMA, Literal: Literal{Str: ","}},
				{Type: TOKEN_PRIMARY, Literal: Literal{Str: "PRIMARY"}},
				{Type: TOKEN_KEY, Literal: Literal{Str: "KEY"}},
				{Type: TOKEN_OPEN_PAREN, Literal: Literal{Str: "("}},
				{Type: TOKEN_IDENT, Literal: Literal{Str: `"user_id"`}},
				{Type: TOKEN_CLOSE_PAREN, Literal: Literal{Str: ")"}},
				{Type: TOKEN_COMMA, Literal: Literal{Str: ","}},
				{Type: TOKEN_UNIQUE, Literal: Literal{Str: "UNIQUE"}},
				{Type: TOKEN_OPEN_PAREN, Literal: Literal{Str: "("}},
				{Type: TOKEN_IDENT, Literal: Literal{Str: `"email"`}},
				{Type: TOKEN_CLOSE_PAREN, Literal: Literal{Str: ")"}},
				{Type: TOKEN_CLOSE_PAREN, Literal: Literal{Str: ")"}},
				{Type: TOKEN_SEMICOLON, Literal: Literal{Str: ";"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := NewLexer(tt.input)
			got := make([]Token, 0)
			for {
				tok := l.NextToken()
				if tok.Type == TOKEN_EOF {
					break
				}
				got = append(got, tok)
			}

			if !require.Equal(t, tt.want, got) {
				t.FailNow()
			}

			for i := range got {
				if !require.Equal(t, got[i].Type, tt.want[i].Type) {
					t.Fail()
				}

				if !require.Equal(t, got[i].Literal, tt.want[i].Literal) {
					t.Fail()
				}
			}
		})
	}
}

func TestLexer_NextToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  Token
	}{
		{
			name:  "failure,|",
			input: `|`,
			want: Token{
				Type:    TOKEN_ILLEGAL,
				Literal: Literal{Str: "|"},
			},
		},
		{
			name:  "failure,:",
			input: `:`,
			want: Token{
				Type:    TOKEN_ILLEGAL,
				Literal: Literal{Str: ":"},
			},
		},
		{
			name:  "failure,!",
			input: `!`,
			want: Token{
				Type:    TOKEN_ILLEGAL,
				Literal: Literal{Str: "!"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			l := NewLexer(tt.input)
			got := l.NextToken()

			if !require.Equal(t, tt.want, got) {
				t.FailNow()
			}
		})
	}
}

func TestLiteral(t *testing.T) {
	t.Parallel()

	t.Run("success,String", func(t *testing.T) {
		t.Parallel()

		literal := Literal{Str: "users"}
		expected := literal.Str
		actual := literal.String()

		require.Equal(t, expected, actual)
	})

	t.Run("success,PlainString", func(t *testing.T) {
		t.Parallel()

		literal := Literal{Str: "users"}
		expected := literal.Str
		actual := literal.StringForDiff()

		require.Equal(t, expected, actual)
	})
}
