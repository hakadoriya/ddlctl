package config

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/hakadoriya/z.go/cliz"
	"github.com/hakadoriya/z.go/errorz"

	"github.com/hakadoriya/ddlctl/pkg/apperr"
	"github.com/hakadoriya/ddlctl/pkg/logs"
)

// Use a structure so that settings can be backed up.
//
//nolint:tagliatelle
type config struct {
	Version     bool   `json:"version"`
	Trace       bool   `json:"trace"`
	Debug       bool   `json:"debug"`
	Language    string `json:"language"`
	Dialect     string `json:"dialect"`
	AutoApprove bool   `json:"auto_approve"`
	// Golang
	ColumnTagGo string `json:"column_tag_go"`
	DDLTagGo    string `json:"ddl_tag_go"`
	PKTagGo     string `json:"pk_tag_go"`
}

//nolint:gochecknoglobals
var (
	globalConfig   *config
	globalConfigMu sync.RWMutex
)

func MustLoad(ctx context.Context) (rollback func()) {
	rollback, err := Load(ctx)
	if err != nil {
		err = apperr.Errorf("Load: %w", err)
		panic(err)
	}
	return rollback
}

func Load(ctx context.Context) (rollback func(), err error) {
	globalConfigMu.Lock()
	defer globalConfigMu.Unlock()
	backup := globalConfig

	cfg, err := load(ctx)
	if err != nil {
		return nil, apperr.Errorf("load: %w", err)
	}

	globalConfig = cfg

	rollback = func() {
		globalConfigMu.Lock()
		defer globalConfigMu.Unlock()
		globalConfig = backup
	}

	return rollback, nil
}

// MEMO: Since there is a possibility of returning some kind of error in the future, the signature is made to return an error.
//
//nolint:funlen
func load(ctx context.Context) (cfg *config, err error) { //nolint:unparam
	cmd := cliz.MustFromContext(ctx)

	c := &config{
		Trace:       loadTrace(ctx, cmd),
		Debug:       loadDebug(ctx, cmd),
		Language:    loadLanguage(ctx, cmd),
		Dialect:     loadDialect(ctx, cmd),
		AutoApprove: loadAutoApprove(ctx, cmd),
		ColumnTagGo: loadColumnTagGo(ctx, cmd),
		DDLTagGo:    loadDDLTagGo(ctx, cmd),
		PKTagGo:     loadPKTagGo(ctx, cmd),
	}

	switch {
	case c.Trace:
		apperr.Errorf = errorz.Errorf //nolint:reassign
		logs.Trace = logs.NewTrace()
		logs.Debug = logs.NewDebug()
		logs.Trace.Print("trace mode enabled")
	case c.Debug:
		apperr.Errorf = errorz.Errorf //nolint:reassign
		logs.Debug = logs.NewDebug()
		logs.Debug.Print("debug mode enabled")
	}

	if err := json.NewEncoder(logs.Debug).Encode(c); err != nil {
		logs.Debug.Printf("config: %#v", c)
	}

	return c, nil
}
