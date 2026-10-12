package wine

import (
	"fmt"
	"path/filepath"
	"strings"
)

// HostPath converts a possibly Windows-style path (with backslash separators) to
// the host's native separators, so a path authored on Windows resolves correctly
// when a program is driven through Wine from another OS.
func HostPath(path string) string { return filepath.FromSlash(strings.ReplaceAll(path, "\\", "/")) }

// absolutePath resolves path against the current working directory after
// normalizing its separators.
func absolutePath(path string) (string, error) {
	resolved, err := filepath.Abs(HostPath(path))
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", path, err)
	}

	return resolved, nil
}
