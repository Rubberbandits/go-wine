//nolint:testpackage // White-box tests exercise the package's execution seam.
package wine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSamePrefixDetectsExistingAlias(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	prefix := filepath.Join(root, "themida")
	alias := filepath.Join(root, "alias")

	requireNoError(t, os.Mkdir(prefix, 0o700))

	err := os.Symlink(prefix, alias)
	if err != nil {
		t.Skipf("cannot create prefix symlink: %v", err)
	}

	same, err := SamePrefix(prefix, alias)
	requireNoError(t, err)
	requireTruef(t, same, "prefix alias not detected")
}

func TestSamePrefixDistinctDirectories(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	same, err := SamePrefix(filepath.Join(root, "a"), filepath.Join(root, "b"))
	requireNoError(t, err)
	requireTruef(t, !same, "distinct prefixes reported as the same")
}
