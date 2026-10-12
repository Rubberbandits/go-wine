package wine

import (
	"fmt"
	"os"
	"path/filepath"
)

// SamePrefix reports whether two WINEPREFIX values resolve to the same prefix,
// also recognizing an existing symlink that points at the same directory. An
// empty value resolves like Wine itself: to the WINEPREFIX environment variable
// if set, otherwise to ~/.wine. It is used to reject sharing a prefix that must
// be dedicated to one task.
func SamePrefix(first, second string) (bool, error) {
	resolve := func(prefix string) (string, error) {
		if prefix == "" {
			prefix = os.Getenv("WINEPREFIX")
		}

		if prefix == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("resolve default Wine prefix: %w", err)
			}

			prefix = filepath.Join(home, ".wine")
		}

		resolved, err := filepath.Abs(HostPath(prefix))
		if err != nil {
			return "", fmt.Errorf("resolve Wine prefix %q: %w", prefix, err)
		}

		return resolved, nil
	}

	first, err := resolve(first)
	if err != nil {
		return false, err
	}

	second, err = resolve(second)
	if err != nil {
		return false, err
	}

	if first == second {
		return true, nil
	}

	firstInfo, firstErr := os.Stat(first)
	secondInfo, secondErr := os.Stat(second)

	return firstErr == nil && secondErr == nil && os.SameFile(firstInfo, secondInfo), nil
}
