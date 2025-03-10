package util

import (
	"path/filepath"

	"github.com/hakadoriya/ddlctl/pkg/logs"
)

func Abs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		logs.Warn.Printf("failed to get absolute path. use path instead: path=%s: %v", path, err)
		return path
	}
	return abs
}
