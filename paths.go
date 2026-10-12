package wine

import (
	"fmt"
	"path/filepath"
	"strings"
)

// hostPath converts a possibly Windows-style path (with backslash separators) to
// the host's native separators, so paths authored on Windows resolve correctly
// when a program is driven through Wine from another OS.
func hostPath(path string) string { return filepath.FromSlash(strings.ReplaceAll(path, "\\", "/")) }

// absolutePath resolves path against the current working directory after
// normalizing its separators.
func absolutePath(path string) (string, error) {
	resolved, err := filepath.Abs(hostPath(path))
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", path, err)
	}

	return resolved, nil
}
