package apply

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hakadoriya/z.go/cliz"
	"github.com/hakadoriya/z.go/databasez/sqlz"
	"github.com/hakadoriya/z.go/errorz"
	retry "github.com/hakadoriya/z.go/retryz"

	"github.com/hakadoriya/ddlctl/pkg/apperr"
	"github.com/hakadoriya/ddlctl/pkg/ddl"
	ddlcrdb "github.com/hakadoriya/ddlctl/pkg/ddl/cockroachdb"
	ddlmysql "github.com/hakadoriya/ddlctl/pkg/ddl/mysql"
	ddlpostgres "github.com/hakadoriya/ddlctl/pkg/ddl/postgres"
	ddlspanner "github.com/hakadoriya/ddlctl/pkg/ddl/spanner"
	"github.com/hakadoriya/ddlctl/pkg/ddlctl/diff"
	"github.com/hakadoriya/ddlctl/pkg/internal/config"
	"github.com/hakadoriya/ddlctl/pkg/internal/consts"
	"github.com/hakadoriya/ddlctl/pkg/internal/util"
	"github.com/hakadoriya/ddlctl/pkg/logs"
)

//nolint:cyclop,funlen,gocognit,gocyclo
func Command(c *cliz.Command, args []string) (err error) {
	ctx := c.Context()
	if _, err := config.Load(ctx); err != nil {
		return apperr.Errorf("config.Load: %w", err)
	}

	const beforeAndAfterForDiff = 2
	if len(args) != beforeAndAfterForDiff {
		return apperr.Errorf("args=%v: %w", args, apperr.ErrTwoArgumentsRequired)
	}

	dialect := config.Dialect()
	language := config.Language()
	leftArg, rightArg := args[0], args[1]

	buf := new(strings.Builder)
	if err := diff.Diff(ctx, buf, dialect, language, leftArg, rightArg); err != nil {
		if errors.Is(err, ddl.ErrNoDifference) {
			_, _ = fmt.Fprintln(os.Stdout, ddl.ErrNoDifference.Error())
			return nil
		}
		return apperr.Errorf("diff: %w", err)
	}
	ddlStr := buf.String()

	msg := `
ddlctl will exec the following DDL queries:

-- 8< --

` + ddlStr + `

-- >8 --

Do you want to apply these DDL queries?
  ddlctl will exec the DDL queries described above.
  Only 'yes' will be accepted to approve.

Enter a value: `

	if _, err := os.Stdout.WriteString(msg); err != nil {
		return apperr.Errorf("os.Stdout.WriteString: %w", err)
	}

	if config.AutoApprove() {
		if _, err := fmt.Fprintf(os.Stdout, "yes (via --%s option)\n", consts.OptionAutoApprove); err != nil {
			return apperr.Errorf("os.Stdout.WriteString: %w", err)
		}
	} else {
		if err := prompt(); err != nil {
			return apperr.Errorf("prompt: %w", err)
		}
	}

	os.Stdout.WriteString("\nexecuting...\n")

	if err := Apply(ctx, dialect, leftArg, ddlStr); err != nil {
		return apperr.Errorf("Apply: %w", err)
	}

	os.Stdout.WriteString("done\n")

	return nil
}

//nolint:cyclop,funlen,gocognit,gocyclo
func Apply(ctx context.Context, dialect, dsn, ddlStr string) error {
	driverName := func() string {
		switch dialect {
		case ddlcrdb.Dialect:
			return ddlcrdb.DriverName
		case ddlpostgres.Dialect:
			return ddlpostgres.DriverName
		default:
			return dialect
		}
	}()

	db, err := sqlz.OpenContext(ctx, driverName, dsn)
	if err != nil {
		return apperr.Errorf("sqlz.OpenContext: %w", err)
	}
	defer func() {
		if err2 := db.Close(); err2 != nil && err == nil {
			err = apperr.Errorf("db.Close: %w", err2)
		}
	}()

	switch dialect {
	case ddlcrdb.Dialect:
		if err := splitExec(
			ctx,
			db,
			ddlStr,
			func(err error) bool { return errorz.Contains(err, "already exists") },
			func(_ error) bool { return false }, // TODO: handle error
		); err != nil {
			return apperr.Errorf("splitExec: %w", err)
		}
	case ddlmysql.Dialect:
		if err := splitExec(
			ctx,
			db,
			ddlStr,
			func(err error) bool {
				return errorz.Contains(err, "already exists") || errorz.Contains(err, "Duplicate column name")
			},
			func(err error) bool { return errorz.Contains(err, "Cannot add foreign key constraint") },
		); err != nil {
			return apperr.Errorf("splitExec: %w", err)
		}
	case ddlpostgres.Dialect:
		if err := splitExec(
			ctx,
			db,
			ddlStr,
			func(err error) bool {
				return errorz.Contains(err, "already exists") || errorz.Contains(err, "does not exist")
			},
			func(_ error) bool { return false }, // TODO: handle error
		); err != nil {
			return apperr.Errorf("splitExec: %w", err)
		}
	case ddlspanner.Dialect:
		conn, err := db.Conn(ctx)
		if err != nil {
			return apperr.Errorf("db.Conn: %w", err)
		}
		defer func() {
			if err2 := conn.Close(); err == nil && err2 != nil {
				err = apperr.Errorf("conn.Close: %w", err2)
			}
		}()
		{
			q := "START BATCH DDL"
			if _, err := conn.ExecContext(ctx, q); err != nil {
				return apperr.Errorf("conn.ExecContext: q=%s: %w", q, err)
			}
		}
		commentTrimmedDDL := readLine(ddlStr, "\n", readLineFuncRemoveCommentLine("--"))
		for _, q := range strings.Split(commentTrimmedDDL, ";\n") {
			if len(q) == 0 {
				// skip empty query
				continue
			}
			if _, err := conn.ExecContext(ctx, q); err != nil {
				return apperr.Errorf("conn.ExecContext: q=%s: %w", q, err)
			}
		}

		{
			q := "RUN BATCH"
			if _, err := conn.ExecContext(ctx, q); err != nil {
				return apperr.Errorf("conn.ExecContext: q=%s: %w", q, err)
			}
		}
	default:
		if _, err := db.ExecContext(ctx, ddlStr); err != nil {
			return apperr.Errorf("db.ExecContext: q=%s: %w", ddlStr, err)
		}
	}

	return nil
}

func readLine(content string, lineSeparator string, f func(line string, lineSeparator string, lastLine bool) (treated string)) string {
	var result string
	lines := strings.Split(content, lineSeparator)
	lastLine := len(lines) - 1
	for i, line := range lines {
		treated := f(line, lineSeparator, i == lastLine)
		result += treated
	}
	return result
}

func readLineFuncRemoveCommentLine(commentPrefix string) func(line string, lineSeparator string, lastLine bool) (treated string) {
	return func(line string, lineSeparator string, lastLine bool) (treated string) {
		trimmed := strings.TrimSpace(line)

		if lastLine && trimmed == "" {
			return ""
		}

		if strings.HasPrefix(trimmed, commentPrefix) {
			return ""
		}

		return line + lineSeparator
	}
}

func prompt() error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	switch input := scanner.Text(); input {
	case "yes":
		return nil
	default:
		return apperr.Errorf("input=%q: %w", input, apperr.ErrCanceled)
	}
}

func splitExec(
	ctx context.Context,
	db interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	},
	ddlStr string,
	notErrorNotLogFunc func(err error) bool,
	errorNotLogFunc func(err error) bool,
) error {
	ddls := strings.Split(util.RemoveCommentsAndEmptyLines("--", ddlStr), ";\n")
	const interval = 500 * time.Millisecond
	retryer := retry.New(ctx, retry.NewConfig(interval, interval, retry.WithMaxRetries(len(ddls))))
	if err := retryer.Do(func(ctx context.Context) error {
		var outerErr error
		for _, q := range ddls {
			if len(q) == 0 {
				// skip empty query
				continue
			}
			if _, err := db.ExecContext(ctx, q); err != nil {
				// If the error is one of the following, do not error and not log. go to the next DDL;
				if notErrorNotLogFunc(err) {
					continue
				}

				err = apperr.Errorf("db.ExecContext: q=%s: %w", q, err)
				outerErr = err
				// If the error is one of the following, error but not log. go to the next DDL;
				if errorNotLogFunc(err) {
					continue
				}

				// If the error is not one of the above, error and log. go to the next DDL;
				logs.Warn.Printf(err.Error())
			}
		}
		if outerErr != nil {
			return outerErr
		}
		return nil
	}); err != nil {
		return apperr.Errorf("retry.Do: %w", err)
	}

	return nil
}
