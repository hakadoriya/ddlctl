package config

import (
	"context"

	"github.com/hakadoriya/z.go/cliz"

	"github.com/kunitsucom/ddlctl/pkg/internal/consts"
)

func loadLanguage(_ context.Context, cmd *cliz.Command) string {
	v, _ := cmd.GetOptionString(consts.OptionLanguage)
	return v
}

func Language() string {
	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	return globalConfig.Language
}
