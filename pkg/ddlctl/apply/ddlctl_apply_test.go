package apply

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"
	"testing"
	"testing/quick"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
	require "github.com/hakadoriya/z.go/testingz/requirez"

	"github.com/hakadoriya/ddlctl/pkg/logs"
)

var errExecFailed = errors.New("exec failed")

// execContextFunc は splitExec に渡す db を関数で差し替えるためのテスト用アダプタ
type execContextFunc func(ctx context.Context, query string, args ...any) (sql.Result, error)

func (f execContextFunc) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return f(ctx, query, args...)
}

func TestReadLine(t *testing.T) {
	t.Parallel()

	// 各行を区切り文字付きでそのまま返す関数を渡すと、入力がそのまま復元されること
	passthrough := func(line string, lineSeparator string, lastLine bool) string {
		if lastLine {
			return line
		}
		return line + lineSeparator
	}
	for _, lineSeparator := range []string{"\n", ";\n", "\r\n"} {
		t.Run("success,RoundTripWithPassthrough,"+strconv.Quote(lineSeparator), func(t *testing.T) {
			t.Parallel()

			// ランダムな文字列だけでは区切り文字がほぼ出現しないため、行の配列を区切り文字で連結して入力を作る
			property := func(lines []string) bool {
				content := strings.Join(lines, lineSeparator)
				return readLine(content, lineSeparator, passthrough) == content
			}
			require.NoError(t, quick.Check(property, nil))
		})
	}

	for _, tt := range []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "success,RemoveCommentLines",
			content:  "-- comment\nCREATE TABLE a (id INT);\n  -- indented comment\nCREATE INDEX i ON a (id);\n",
			expected: "CREATE TABLE a (id INT);\nCREATE INDEX i ON a (id);\n",
		},
		{
			// 末尾に改行がない場合、最終行にも区切り文字が付与されること
			name:     "success,NoTrailingLineSeparator",
			content:  "CREATE TABLE a (id INT);",
			expected: "CREATE TABLE a (id INT);\n",
		},
		{
			name:     "success,Empty",
			content:  "",
			expected: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, readLine(tt.content, "\n", readLineFuncRemoveCommentLine("--")))
		})
	}
}

//nolint:paralleltest // グローバル変数 logs.Warn を差し替えるため並列実行しない
func TestSplitExec(t *testing.T) {
	t.Run("failure,LogErrorContainingPercentVerbatim", func(t *testing.T) {
		// クエリに % を含む DDL が失敗した場合、エラーメッセージがフォーマット文字列として解釈されず、そのままログに出力されること
		buf := new(bytes.Buffer)
		backup := logs.Warn
		t.Cleanup(func() { logs.Warn = backup })
		logs.Warn = &logs.DefaultLogger{Logger: log.New(buf, "", 0)}

		db := execContextFunc(func(context.Context, string, ...any) (sql.Result, error) {
			return nil, errExecFailed
		})
		noop := func(error) bool { return false }

		err := splitExec(context.Background(), db, "ALTER TABLE t ADD CONSTRAINT c CHECK (col LIKE '%d');\n", noop, noop)
		require.ErrorIs(t, err, errExecFailed)

		const expected = "db.ExecContext: q=ALTER TABLE t ADD CONSTRAINT c CHECK (col LIKE '%d'): exec failed"
		lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
		require.True(t, len(lines) > 0 && lines[0] != "", "no log output")
		for _, line := range lines {
			assert.Equal(t, expected, line)
		}
	})
}
