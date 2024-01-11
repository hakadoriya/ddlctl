package config

import (
	"context"

	cliz "github.com/kunitsucom/util.go/exp/cli"

	"github.com/kunitsucom/ddlctl/pkg/internal/consts"
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
