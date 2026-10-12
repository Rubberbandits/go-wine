package wine

import "errors"

// These sentinel errors classify the package's failures and are matched with
// errors.Is.
var (
	// ErrInvalidPath indicates that winepath returned no usable Windows path for
	// a host path.
	ErrInvalidPath = errors.New("invalid Wine path")
	// ErrNoPrefix indicates that an [Environment] was used without a prefix,
	// which must be set because the prefix is shut down after each run.
	ErrNoPrefix = errors.New("no Wine prefix configured")
)
