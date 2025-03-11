package ddlctl

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/hakadoriya/z.go/buildinfoz"
	"github.com/hakadoriya/z.go/cliz"

	"github.com/hakadoriya/ddlctl/pkg/apperr"
	"github.com/hakadoriya/ddlctl/pkg/ddlctl/apply"
	"github.com/hakadoriya/ddlctl/pkg/ddlctl/diff"
	"github.com/hakadoriya/ddlctl/pkg/ddlctl/generate"
	"github.com/hakadoriya/ddlctl/pkg/ddlctl/show"
	"github.com/hakadoriya/ddlctl/pkg/internal/consts"
)

//nolint:gochecknoglobals
var (
	optLanguage = &cliz.StringOption{
		Name:        consts.OptionLanguage,
		Env:         consts.EnvKeyLanguage,
		Description: "programming language to generate DDL",
		Default:     "go",
	}
	optDialect = &cliz.StringOption{
		Name:        consts.OptionDialect,
		Env:         consts.EnvKeyDialect,
		Description: "SQL dialect to generate DDL",
		Default:     "",
	}
	opts = []cliz.Option{
		optLanguage,
		optDialect,
		// Golang
		&cliz.StringOption{
			Name:        consts.OptionGoColumnTag,
			Env:         consts.EnvKeyGoColumnTag,
			Description: "column annotation key for Go struct tag",
			Default:     "db",
		},
		&cliz.StringOption{
			Name:        consts.OptionGoDDLTag,
			Env:         consts.EnvKeyGoDDLTag,
			Description: "DDL annotation key for Go struct tag",
			Default:     "ddlctl",
		},
		&cliz.StringOption{
			Name:        consts.OptionGoPKTag,
			Env:         consts.EnvKeyGoPKTag,
			Description: "primary key annotation key for Go struct tag",
			Default:     "pk",
		},
	}
)

//nolint:cyclop,funlen
func DDLCtl(ctx context.Context) error {
	cmd := cliz.Command{
		Name:        "ddlctl",
		Usage:       "ddlctl [options]",
		Description: "ddlctl is a tool for control RDBMS DDL.",
		SubCommands: []*cliz.Command{
			{
				Name:        "version",
				Usage:       "ddlctl version",
				Description: "show version",
				ExecFunc: func(_ *cliz.Command, _ []string) error {
					fmt.Printf("version: %s\n", buildinfoz.BuildVersion())           //nolint:forbidigo
					fmt.Printf("revision: %s\n", buildinfoz.BuildRevision())         //nolint:forbidigo
					fmt.Printf("build branch: %s\n", buildinfoz.BuildBranch())       //nolint:forbidigo
					fmt.Printf("build timestamp: %s\n", buildinfoz.BuildTimestamp()) //nolint:forbidigo
					return nil
				},
			},
			{
				Name:        "generate",
				Aliases:     []string{"gen"},
				Usage:       "ddlctl generate [options] --dialect <DDL dialect> <source> <destination>",
				Description: "generate DDL from source (file or directory) to destination (file or directory).",
				Options:     opts,
				ExecFunc:    generate.Command,
			},
			{
				Name:        "show",
				Usage:       "ddlctl show --dialect <DDL dialect> <DSN>",
				Description: "show DDL from DSN like `SHOW CREATE TABLE`.",
				Options:     []cliz.Option{optDialect},
				ExecFunc:    show.Command,
			},
			{
				Name:        "diff",
				Usage:       "ddlctl diff [options] --dialect <DDL dialect> <before DDL source> <after DDL source>",
				Description: "diff DDL from <before DDL source> to <after DDL source>.",
				Options:     opts,
				ExecFunc:    diff.Command,
			},
			{
				Name:        "apply",
				Usage:       "ddlctl apply [options] --dialect <DDL dialect> <DSN to apply> <DDL source>",
				Description: "apply DDL from <DDL source> to <DSN to apply>.",
				Options: append(opts,
					&cliz.BoolOption{
						Name:        consts.OptionAutoApprove,
						Env:         consts.EnvKeyAutoApprove,
						Description: "auto approve",
						Default:     false,
					},
				),
				ExecFunc: apply.Command,
			},
		},
		Options: []cliz.Option{
			&cliz.BoolOption{
				Name:        consts.OptionTrace,
				Env:         consts.EnvKeyTrace,
				Description: "trace mode enabled",
				Default:     false,
			},
			&cliz.BoolOption{
				Name:        consts.OptionDebug,
				Env:         consts.EnvKeyDebug,
				Description: "debug mode",
				Default:     false,
			},
		},
	}

	if err := cmd.Exec(ctx, os.Args); err != nil {
		if errors.Is(err, cliz.ErrHelp) {
			return nil
		}

		return apperr.Errorf("cmd.Run: %w", err)
	}

	return nil
}
