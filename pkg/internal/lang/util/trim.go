package util

import (
	"strings"

	"github.com/hakadoriya/z.go/slicez"
)

func TrimCommentElementHasPrefix(stringSlice []string, prefix string) []string {
	return slicez.Filter(stringSlice, func(_ int, s string) bool {
		return !strings.HasPrefix(s, prefix)
	})
}

func TrimCommentElementTailEmpty(stringSlice []string) []string {
	if len(stringSlice) == 0 {
		return stringSlice
	}

	if stringSlice[len(stringSlice)-1] == "" {
		return stringSlice[:len(stringSlice)-1]
	}

	return stringSlice
}
