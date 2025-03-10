package generate

import (
	"context"
	"io"
	"os"
	"path/filepath"

	apperr "github.com/hakadoriya/ddlctl/pkg/apperr"
	crdbddl "github.com/hakadoriya/ddlctl/pkg/ddl/cockroachdb"
	"github.com/hakadoriya/ddlctl/pkg/internal/config"
	"github.com/hakadoriya/ddlctl/pkg/internal/generator"
	"github.com/hakadoriya/ddlctl/pkg/internal/generator/dialect/mysql"
	"github.com/hakadoriya/ddlctl/pkg/internal/generator/dialect/postgres"
	"github.com/hakadoriya/ddlctl/pkg/internal/generator/dialect/spanner"
	ddlctlgo "github.com/hakadoriya/ddlctl/pkg/internal/lang/go"
	"github.com/hakadoriya/ddlctl/pkg/logs"
	"github.com/hakadoriya/z.go/cliz"
)

func Command(c *cliz.Command, args []string) error {
	ctx := c.Context()
	if _, err := config.Load(ctx); err != nil {
		return apperr.Errorf("config.Load: %w", err)
	}

	dialect := config.Dialect()
	language := config.Language()
	src := args[0]
	dst := args[1]

	logs.Info.Printf("dialect: %s", dialect)
	logs.Info.Printf("language: %s", language)
	logs.Info.Printf("source: %s", src)
	logs.Info.Printf("destination: %s", dst)

	if info, err := os.Stat(dst); err == nil && info.IsDir() {
		dst = filepath.Join(dst, "ddlctl.gen.sql")
	}

	const rw_r__r__ = 0o644 //nolint:revive,stylecheck
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, rw_r__r__)
	if err != nil {
		return apperr.Errorf("os.OpenFile: %w", err)
	}

	if err := Generate(ctx, dstFile, src, dialect, language); err != nil {
		return apperr.Errorf("fprint: %w", err)
	}
	return nil
}

func Generate(ctx context.Context, dst io.Writer, src, dialect, language string) error {
	ddl, err := Parse(ctx, language, src)
	if err != nil {
		return apperr.Errorf("parse: %w", err)
	}

	if err := Fprint(dst, dialect, ddl); err != nil {
		return apperr.Errorf("fprint: %w", err)
	}
	return nil
}

func Parse(ctx context.Context, language string, src string) (*generator.DDL, error) {
	switch language {
	case ddlctlgo.Language:
		ddl, err := ddlctlgo.Parse(ctx, src)
		if err != nil {
			return nil, apperr.Errorf("ddlctlgo.Parse: %w", err)
		}
		return ddl, nil
	default:
		return nil, apperr.Errorf("language=%s: %w", language, apperr.ErrNotSupported)
	}
}

func Fprint(w io.Writer, dialect string, ddl *generator.DDL) error {
	switch dialect {
	case spanner.Dialect:
		if err := spanner.Fprint(w, ddl); err != nil {
			return apperr.Errorf("spanner.Fprint: %w", err)
		}
		return nil
	case postgres.Dialect, crdbddl.Dialect:
		if err := postgres.Fprint(w, ddl); err != nil {
			return apperr.Errorf("postgres.Fprint: %w", err)
		}
		return nil
	case mysql.Dialect:
		if err := mysql.Fprint(w, ddl); err != nil {
			return apperr.Errorf("mysql.Fprint: %w", err)
		}
		return nil
	case "":
		return apperr.Errorf("dialect=%s: %w", dialect, apperr.ErrDialectIsEmpty)
	default:
		return apperr.Errorf("dialect=%s: %w", dialect, apperr.ErrNotSupported)
	}
}
