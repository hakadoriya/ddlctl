package diff

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/hakadoriya/z.go/cliz"

	"github.com/hakadoriya/ddlctl/pkg/apperr"
	"github.com/hakadoriya/ddlctl/pkg/ddl"
	ddlcrdb "github.com/hakadoriya/ddlctl/pkg/ddl/cockroachdb"
	ddlmysql "github.com/hakadoriya/ddlctl/pkg/ddl/mysql"
	ddlpg "github.com/hakadoriya/ddlctl/pkg/ddl/postgres"
	ddlspanner "github.com/hakadoriya/ddlctl/pkg/ddl/spanner"
	"github.com/hakadoriya/ddlctl/pkg/ddlctl/generate"
	"github.com/hakadoriya/ddlctl/pkg/ddlctl/show"
	"github.com/hakadoriya/ddlctl/pkg/internal/config"
	"github.com/hakadoriya/ddlctl/pkg/logs"
)

func Command(c *cliz.Command, args []string) error {
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

	if err := Diff(ctx, os.Stdout, dialect, language, leftArg, rightArg); err != nil {
		if errors.Is(err, ddl.ErrNoDifference) {
			logs.Debug.Print(ddl.ErrNoDifference.Error())
			return nil
		}
		return apperr.Errorf("diff: %w", err)
	}

	return nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

//nolint:cyclop
func resolve(ctx context.Context, language, dialect, arg string) (ddl string, err error) {
	switch {
	case isFile(arg): // NOTE: expect SQL file
		ddlBytes, err := os.ReadFile(arg)
		if err != nil {
			return "", apperr.Errorf("os.ReadFile: %w", err)
		}
		ddl = string(ddlBytes)
	case exists(arg): // NOTE: expect ddlctl generate format
		b := new(strings.Builder)
		if err := generate.Generate(ctx, b, arg, dialect, language); err != nil {
			return "", apperr.Errorf("Generate: %w", err)
		}
		ddl = b.String()
	default: // NOTE: expect DSN
		genDDL, err := show.Show(ctx, dialect, arg)
		if err != nil {
			return "", apperr.Errorf("Show: %w", err)
		}
		ddl = genDDL
	}

	return ddl, nil
}

//nolint:cyclop,funlen,gocognit
func Diff(ctx context.Context, out io.Writer, dialect, language, src string, dst string) error {
	srcDDL, err := resolve(ctx, language, dialect, src)
	if err != nil {
		return apperr.Errorf("resolve: %w", err)
	}

	dstDDL, err := resolve(ctx, language, dialect, dst)
	if err != nil {
		return apperr.Errorf("resolve: %w", err)
	}

	logs.Trace.Printf("srcDDL: %q", srcDDL)
	logs.Trace.Printf("dstDDL: %q", dstDDL)

	switch dialect {
	case ddlmysql.Dialect:
		leftDDL, err := ddlmysql.NewParser(ddlmysql.NewLexer(srcDDL)).Parse()
		if err != nil {
			return apperr.Errorf("myddl.NewParser: %w", err)
		}
		rightDDL, err := ddlmysql.NewParser(ddlmysql.NewLexer(dstDDL)).Parse()
		if err != nil {
			return apperr.Errorf("myddl.NewParser: %w", err)
		}

		result, err := ddlmysql.Diff(leftDDL, rightDDL)
		if err != nil {
			return apperr.Errorf("myddl.Diff: %w", err)
		}

		if _, err := io.WriteString(out, result.String()); err != nil {
			return apperr.Errorf("io.WriteString: %w", err)
		}

		return nil
	case ddlpg.Dialect:
		leftDDL, err := ddlpg.NewParser(ddlpg.NewLexer(srcDDL)).Parse()
		if err != nil {
			return apperr.Errorf("pgddl.NewParser: %w", err)
		}
		rightDDL, err := ddlpg.NewParser(ddlpg.NewLexer(dstDDL)).Parse()
		if err != nil {
			return apperr.Errorf("pgddl.NewParser: %w", err)
		}

		result, err := ddlpg.Diff(leftDDL, rightDDL)
		if err != nil {
			return apperr.Errorf("pgddl.Diff: %w", err)
		}

		if _, err := io.WriteString(out, result.String()); err != nil {
			return apperr.Errorf("io.WriteString: %w", err)
		}

		return nil
	case ddlcrdb.Dialect:
		leftDDL, err := ddlcrdb.NewParser(ddlcrdb.NewLexer(srcDDL)).Parse()
		if err != nil {
			return apperr.Errorf("pgddl.NewParser: %w", err)
		}
		rightDDL, err := ddlcrdb.NewParser(ddlcrdb.NewLexer(dstDDL)).Parse()
		if err != nil {
			return apperr.Errorf("pgddl.NewParser: %w", err)
		}

		result, err := ddlcrdb.Diff(leftDDL, rightDDL)
		if err != nil {
			return apperr.Errorf("pgddl.Diff: %w", err)
		}

		if _, err := io.WriteString(out, result.String()); err != nil {
			return apperr.Errorf("io.WriteString: %w", err)
		}

		return nil
	case ddlspanner.Dialect:
		leftDDL, err := ddlspanner.NewParser(ddlspanner.NewLexer(srcDDL)).Parse()
		if err != nil {
			return apperr.Errorf("spanddl.NewParser: %w", err)
		}
		rightDDL, err := ddlspanner.NewParser(ddlspanner.NewLexer(dstDDL)).Parse()
		if err != nil {
			return apperr.Errorf("spanddl.NewParser: %w", err)
		}

		result, err := ddlspanner.Diff(leftDDL, rightDDL)
		if err != nil {
			return apperr.Errorf("spanddl.Diff: %w", err)
		}

		if _, err := io.WriteString(out, result.String()); err != nil {
			return apperr.Errorf("io.WriteString: %w", err)
		}

		return nil
	case "":
		return apperr.Errorf("dialect=%s: %w", dialect, apperr.ErrDialectIsEmpty)
	default:
		return apperr.Errorf("dialect=%s: %w", dialect, apperr.ErrNotSupported)
	}
}
