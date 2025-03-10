package show

import (
	"context"
	"database/sql/driver"
	"errors"
	"io"
	"os"

	"github.com/hakadoriya/z.go/cliz"
	"github.com/hakadoriya/z.go/databasez/sqlz"

	apperr "github.com/hakadoriya/ddlctl/pkg/apperr"
	crdbddl "github.com/hakadoriya/ddlctl/pkg/ddl/cockroachdb"
	myddl "github.com/hakadoriya/ddlctl/pkg/ddl/mysql"
	pgddl "github.com/hakadoriya/ddlctl/pkg/ddl/postgres"
	spanddl "github.com/hakadoriya/ddlctl/pkg/ddl/spanner"
	"github.com/hakadoriya/ddlctl/pkg/internal/config"
	crdbshow "github.com/hakadoriya/ddlctl/pkg/show/cockroachdb"
	myshow "github.com/hakadoriya/ddlctl/pkg/show/mysql"
	pgshow "github.com/hakadoriya/ddlctl/pkg/show/postgres"
	spanshow "github.com/hakadoriya/ddlctl/pkg/show/spanner"
)

func Command(c *cliz.Command, args []string) error {
	ctx := c.Context()
	if _, err := config.Load(ctx); err != nil {
		return apperr.Errorf("config.Load: %w", err)
	}

	ddl, err := Show(ctx, config.Dialect(), args[0])
	if err != nil {
		return apperr.Errorf("diff: %w", err)
	}

	if _, err := io.WriteString(os.Stdout, ddl); err != nil {
		return apperr.Errorf("io.WriteString: %w", err)
	}

	return nil
}

//nolint:cyclop,funlen,gocognit
func Show(ctx context.Context, dialect string, dsn string) (ddl string, err error) {
	driverName := func() string {
		switch dialect {
		case crdbddl.Dialect:
			return crdbddl.DriverName
		default:
			return dialect
		}
	}()

	db, err := sqlz.OpenContext(ctx, driverName, dsn)
	if err != nil {
		if dialect == spanddl.Dialect && errors.Is(err, driver.ErrBadConn) {
			err = apperr.Errorf("error such as 'Instance not found' or 'Database not found' might have occurred: %w", err)
		}
		return "", apperr.Errorf("sqlz.OpenContext: %w", err)
	}
	defer func() {
		if cerr := db.Close(); err == nil && cerr != nil {
			err = apperr.Errorf("db.Close: %w", cerr)
		}
	}()

	switch dialect {
	case myddl.Dialect:
		ddl, err := myshow.ShowCreateAllTables(ctx, db)
		if err != nil {
			return "", apperr.Errorf("pgutil.ShowCreateAllTables: %w", err)
		}
		return ddl, nil
	case pgddl.Dialect:
		ddl, err := pgshow.ShowCreateAllTables(ctx, db)
		if err != nil {
			return "", apperr.Errorf("pgutil.ShowCreateAllTables: %w", err)
		}
		return ddl, nil
	case crdbddl.Dialect:
		ddl, err := crdbshow.ShowCreateAllTables(ctx, db)
		if err != nil {
			return "", apperr.Errorf("crdbutil.ShowCreateAllTables: %w", err)
		}
		return ddl, nil
	case spanddl.Dialect:
		ddl, err := spanshow.ShowCreateAllTables(ctx, db)
		if err != nil {
			return "", apperr.Errorf("spanshow.ShowCreateAllTables: %w", err)
		}
		return ddl, nil
	default:
		return "", apperr.Errorf("dialect=%s: %w", dialect, apperr.ErrNotSupported)
	}
}
