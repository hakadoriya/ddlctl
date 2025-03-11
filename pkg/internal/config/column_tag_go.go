package config

import (
	"context"

	"github.com/hakadoriya/z.go/cliz"

	"github.com/hakadoriya/ddlctl/pkg/internal/consts"
)

func loadColumnTagGo(_ context.Context, cmd *cliz.Command) string {
	v, _ := cmd.GetOptionString(consts.OptionGoColumnTag)
	return v
}

func ColumnTagGo() string {
	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	return globalConfig.ColumnTagGo
}
