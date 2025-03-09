package config

import (
	"context"

	"github.com/hakadoriya/z.go/cliz"

	"github.com/kunitsucom/ddlctl/pkg/internal/consts"
)

func loadTrace(_ context.Context, cmd *cliz.Command) bool {
	v, _ := cmd.GetOptionBool(consts.OptionTrace)
	return v
}

func Trace() bool {
	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	return globalConfig.Trace
}
