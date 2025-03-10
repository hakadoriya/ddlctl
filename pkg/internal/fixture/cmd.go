package fixture

import (
	"github.com/hakadoriya/z.go/cliz"

	"github.com/hakadoriya/ddlctl/pkg/internal/consts"
)

func Cmd() *cliz.Command {
	return &cliz.Command{
		Options: []cliz.Option{
			&cliz.StringOption{
				Name:        consts.OptionLanguage,
				Env:         consts.EnvKeyLanguage,
				Description: "programming language to generate DDL",
				Default:     "go",
			},
			&cliz.StringOption{
				Name:        consts.OptionDialect,
				Env:         consts.EnvKeyDialect,
				Description: "SQL dialect to generate DDL",
				Default:     "",
			},
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
		},
	}
}
