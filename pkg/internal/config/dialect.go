package config

import (
	"context"

	"github.com/hakadoriya/z.go/cliz"

	"github.com/kunitsucom/ddlctl/pkg/internal/consts"
)

func loadDialect(_ context.Context, cmd *cliz.Command) string {
	v, _ := cmd.GetOptionString(consts.OptionDialect)
	return v
}

func Dialect() string {
	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	return globalConfig.Dialect
}
