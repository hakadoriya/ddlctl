package integrationtest_test

import (
	"context"
	"os"
	"testing"

	"github.com/hakadoriya/z.go/testingz"
	"github.com/hakadoriya/z.go/testingz/assertz"
	"github.com/hakadoriya/z.go/testingz/requirez"

	"github.com/kunitsucom/ddlctl/pkg/ddlctl/diff"
	"github.com/kunitsucom/ddlctl/pkg/internal/fixture"
)

//nolint:paralleltest
func Test_ddlctl_diff(t *testing.T) {
	t.Run("success,go,postgres", func(t *testing.T) {
		cmd := fixture.Cmd()
		args, err := cmd.Parse(context.Background(), []string{
			"--lang=go",
			"--dialect=postgres",
			"postgres_before.sql",
			"postgres_after.sql",
		})
		requirez.NoError(t, err)

		backup := os.Stdout
		t.Cleanup(func() { os.Stdout = backup })

		w, closeFunc, err := testingz.NewFileWriter(t)
		requirez.NoError(t, err)

		os.Stdout = w
		{
			err := diff.Command(cmd, args)
			requirez.NoError(t, err)
		}
		result := closeFunc()

		const expected = `-- -
-- +description TEXT NOT NULL
ALTER TABLE public.test_groups ADD COLUMN description TEXT NOT NULL;
-- -name TEXT NOT NULL
-- +
ALTER TABLE public.test_users DROP COLUMN name;
-- -
-- +username TEXT NOT NULL
ALTER TABLE public.test_users ADD COLUMN username TEXT NOT NULL;
`

		actual := result.String()

		assertz.Equal(t, expected, actual)
	})
}
