package config

import (
	"context"

	"github.com/hakadoriya/z.go/cliz"

	"github.com/hakadoriya/ddlctl/pkg/internal/consts"
)

func loadPKTagGo(_ context.Context, cmd *cliz.Command) string {
	v, _ := cmd.GetOptionString(consts.OptionGoPKTag)
	return v
}

func PKTagGo() string {
	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	return globalConfig.PKTagGo
}
