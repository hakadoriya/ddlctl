//nolint:testpackage
package spanner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
	require "github.com/hakadoriya/z.go/testingz/requirez"

	"github.com/kunitsucom/ddlctl/pkg/internal/config"
	"github.com/kunitsucom/ddlctl/pkg/internal/fixture"
	ddlctlgo "github.com/kunitsucom/ddlctl/pkg/internal/lang/go"
)

func Test_integrationtest_go_spanner(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		cmd := fixture.Cmd()
		args, err := cmd.Parse(context.Background(), []string{
			"ddlctl",
			"--lang=go",
			"--dialect=spanner",
			"--go-column-tag=dbtest",
			"--go-ddl-tag=spanddl",
			"--go-pk-tag=pkey",
			"integrationtest_go_001.source",
			"dummy",
		})
		require.NoError(t, err)

		ctx := cmd.Context()

		{
			_, err := config.Load(ctx)
			require.NoError(t, err)
		}

		ddl, err := ddlctlgo.Parse(ctx, args[1])
		require.NoError(t, err)

		buf := bytes.NewBuffer(nil)

		require.NoError(t, Fprint(buf, ddl))

		golden, err := os.ReadFile("integrationtest_go_001.golden")
		require.NoError(t, err)

		if !assert.Equal(t, string(golden), buf.String()) {
			fmt.Println(buf.String()) //nolint:forbidigo
		}
	})
}
