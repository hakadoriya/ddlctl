//nolint:testpackage
package apply

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"testing"

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
