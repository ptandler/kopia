package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kopia/kopia/internal/testutil"
	"github.com/kopia/kopia/tests/testenv"
)

func TestSnapshotRunScheduled(t *testing.T) {
	t.Parallel()

	runner := testenv.NewInProcRunner(t)
	e := testenv.NewCLITest(t, testenv.RepoFormatNotImportant, runner)

	defer e.RunAndExpectSuccess(t, "repo", "disconnect")

	e.RunAndExpectSuccess(t, "repo", "create", "filesystem", "--path", e.RepoDir)

	srcdir := testutil.TempDirectory(t)
	require.NoError(t, os.WriteFile(filepath.Join(srcdir, "some-file"), []byte{1, 2, 3}, 0o755))

	e.RunAndExpectSuccess(t, "policy", "set", srcdir,
		"--snapshot-interval=1h",
		"--keep-latest=4",
	)

	e.RunAndExpectSuccess(t, "snapshot", "create", srcdir)

	e.RunAndExpectSuccess(t, "snapshot", "run-scheduled", "--dry-run")

	e.RunAndExpectSuccess(t, "snapshot", "run-scheduled", "--parallel=2")

	e.RunAndExpectSuccess(t, "snapshot", "list", srcdir)
}