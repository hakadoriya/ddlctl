package config

import (
	"context"

	"github.com/hakadoriya/z.go/cliz"

	"github.com/kunitsucom/ddlctl/pkg/internal/consts"
)

func loadDDLTagGo(_ context.Context, cmd *cliz.Command) string {
	v, _ := cmd.GetOptionString(consts.OptionGoDDLTag)
	return v
}

func DDLTagGo() string {
	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	return globalConfig.DDLTagGo
}
