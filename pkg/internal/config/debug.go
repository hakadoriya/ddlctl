package config

import (
	"context"

	"github.com/hakadoriya/z.go/cliz"

	"github.com/hakadoriya/ddlctl/pkg/internal/consts"
)

func loadDebug(_ context.Context, cmd *cliz.Command) bool {
	v, _ := cmd.GetOptionBool(consts.OptionDebug)
	return v
}

func Debug() bool {
	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	return globalConfig.Debug
}
