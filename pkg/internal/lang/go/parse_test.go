//nolint:testpackage
package ddlctlgo

import (
	"context"
	"go/ast"
	"os"
	"path/filepath"
	"testing"

	assert "github.com/hakadoriya/z.go/testingz/assertz"
	require "github.com/hakadoriya/z.go/testingz/requirez"

	"github.com/hakadoriya/ddlctl/pkg/apperr"
	"github.com/hakadoriya/ddlctl/pkg/internal/config"
	"github.com/hakadoriya/ddlctl/pkg/internal/fixture"
	"github.com/hakadoriya/ddlctl/pkg/internal/generator"
)

//nolint:paralleltest
func TestParse(t *testing.T) {
	t.Run("success,common.source", func(t *testing.T) {
		cmd := fixture.Cmd()
		args, err := cmd.Parse(context.Background(), []string{
			"ddlctl",
			"--lang=go",
			"--dialect=spanner",
			"--go-column-tag=dbtest",
			"--go-ddl-tag=spanddl",
			"--go-pk-tag=pkey",
			"tests/common.source",
			"dummy",
		})
		require.NoError(t, err)
		ctx := cmd.Context()

		{
			_, err := config.Load(ctx)
			require.NoError(t, err)
		}

		ddl, err := Parse(ctx, args[1])
		require.NoError(t, err)
		if !assert.Equal(t, 6, len(ddl.Stmts)) { // TODO: 確認
			for _, stmt := range ddl.Stmts {
				t.Logf("🚧: ddl.Stmts: %#v", stmt)
			}
		}
	})

	t.Run("success,info.IsDir", func(t *testing.T) {
		cmd := fixture.Cmd()
		args, err := cmd.Parse(context.Background(), []string{
			"ddlctl",
			"--lang=go",
			"--dialect=spanner",
			"--go-column-tag=dbtest",
			"--go-ddl-tag=spanddl",
			"--go-pk-tag=pkey",
			"tests",
			"dummy",
		})
		require.NoError(t, err)
		ctx := cmd.Context()

		{
			_, err := config.Load(ctx)
			require.NoError(t, err)
		}

		backup := fileSuffix
		t.Cleanup(func() { fileSuffix = backup })
		fileSuffix = ".source"

		{
			ddl, err := Parse(ctx, args[1])
			require.NoError(t, err)
			if !assert.Equal(t, 6, len(ddl.Stmts)) { // TODO: 確認
				for _, stmt := range ddl.Stmts {
					t.Logf("🚧: ddl.Stmts: %#v", stmt)
				}
			}
		}
	})

	t.Run("failure,info.IsDir", func(t *testing.T) {
		tempDir := t.TempDir()
		{
			f, err := os.Create(filepath.Join(tempDir, "error.go"))
			require.NoError(t, err)
			_ = f.Close()
		}

		cmd := fixture.Cmd()
		args, err := cmd.Parse(context.Background(), []string{
			"ddlctl",
			"--lang=go",
			"--dialect=spanner",
			"--go-column-tag=dbtest",
			"--go-ddl-tag=spanddl",
			"--go-pk-tag=pkey",
			tempDir,
			"dummy",
		})
		require.NoError(t, err)

		ctx := cmd.Context()

		{
			_, err := config.Load(ctx)
			require.NoError(t, err)
		}

		{
			_, err := Parse(ctx, args[1])
			require.ErrorContains(t, err, "expected 'package', found 'EOF'")
		}
	})

	t.Run("failure,os.ErrNotExist", func(t *testing.T) {
		cmd := fixture.Cmd()
		args, err := cmd.Parse(context.Background(), []string{
			"ddlctl",
			"--lang=go",
			"--dialect=spanner",
			"--go-column-tag=dbtest",
			"--go-ddl-tag=spanddl",
			"--go-pk-tag=pkey",
			"tests/no-such-file.source",
			"dummy",
		})
		require.NoError(t, err)
		ctx := cmd.Context()

		{
			_, err := config.Load(ctx)
			require.NoError(t, err)
		}

		{
			t.Setenv("PWD", "\\")
			_, err := Parse(ctx, args[1])
			require.Error(t, err)
			assert.ErrorIs(t, err, os.ErrNotExist)
		}
	})

	t.Run("failure,parser.ParseFile", func(t *testing.T) {
		cmd := fixture.Cmd()
		args, err := cmd.Parse(context.Background(), []string{
			"ddlctl",
			"--lang=go",
			"--dialect=spanner",
			"--go-column-tag=dbtest",
			"--go-ddl-tag=spanddl",
			"--go-pk-tag=pkey",
			"tests/no.errsource",
			"dummy",
		})
		require.NoError(t, err)
		ctx := cmd.Context()

		{
			_, err := config.Load(ctx)
			require.NoError(t, err)
		}

		{
			_, err := Parse(ctx, args[1])
			require.Error(t, err)
			assert.ErrorContains(t, err, "expected 'package', found 'EOF'")
		}
	})

	t.Run("failure,extractDDLSource", func(t *testing.T) {
		cmd := fixture.Cmd()
		args, err := cmd.Parse(context.Background(), []string{
			"ddlctl",
			"--lang=go",
			"--dialect=spanner",
			"--go-column-tag=dbtest",
			"--go-ddl-tag=spanddl",
			"--go-pk-tag=pkey",
			"tests/no-go-ddl-tag.source",
			"dummy",
		})
		require.NoError(t, err)
		ctx := cmd.Context()

		{
			_, err := config.Load(ctx)
			require.NoError(t, err)
		}

		{
			_, err := Parse(ctx, args[1])
			require.Error(t, err)
			assert.ErrorIs(t, err, apperr.ErrDDLTagGoAnnotationNotFoundInSource)
		}
	})
}

func Test_walkDirFn(t *testing.T) {
	t.Parallel()

	t.Run("failure,err", func(t *testing.T) {
		t.Parallel()

		cmd := fixture.Cmd()
		_, err := cmd.Parse(context.Background(), []string{
			"ddlctl",
			"--lang=go",
			"--dialect=spanner",
			"--go-column-tag=dbtest",
			"--go-ddl-tag=spanddl",
			"--go-pk-tag=pkey",
			"tests",
			"dummy",
		})
		require.NoError(t, err)
		ctx := cmd.Context()

		{
			_, err := config.Load(ctx)
			require.NoError(t, err)
		}

		ddl := generator.NewDDL(ctx)
		fn := walkDirFn(ctx, ddl)
		{
			err := fn("", nil, os.ErrPermission)
			require.Error(t, err)
		}
	})
}

func Test_extractContainingCommentFromCommentGroup(t *testing.T) {
	t.Parallel()

	t.Run("failure,no-such-string", func(t *testing.T) {
		t.Parallel()

		actual := extractContainingCommentFromCommentGroup(&ast.CommentGroup{
			List: []*ast.Comment{
				{
					Text: "// spanddl: index: CREATE INDEX `idx_users_name` ON `users` (`name`)",
				},
			},
		}, "no-such-string")
		assert.Nil(t, actual)
	})
}
