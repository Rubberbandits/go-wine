package wine

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Config names the host Wine tools and the prefix a run uses. The zero value is
// usable after [Config.WithDefaults] supplies the standard tool names; only
// Prefix has no default, because it must be chosen deliberately.
//
// The prefix must be owned exclusively by the work driven through it: [Config.Stop]
// shuts the whole Wine server down, so a prefix shared with other Wine processes
// would disrupt them.
type Config struct {
	// Binary is the wine loader used to run a Windows executable. Empty defaults
	// to "wine" resolved on PATH.
	Binary string `json:"binary,omitempty"`
	// PathBinary is the winepath tool used to convert host paths to Windows
	// paths. Empty defaults to "winepath".
	PathBinary string `json:"path_binary,omitempty"`
	// ServerBinary is the wineserver tool used to shut the prefix down. Empty
	// defaults to a "wineserver" beside Binary.
	ServerBinary string `json:"server_binary,omitempty"`
	// Prefix is the WINEPREFIX directory. It has no default and must be set
	// before the configuration is used to run anything.
	Prefix string `json:"prefix,omitempty"`
}

// WithDefaults returns a copy of the configuration with any unset tool paths
// filled in: "wine", "winepath", and a "wineserver" located beside Binary.
// Prefix is left untouched.
func (c Config) WithDefaults() Config {
	if c.Binary == "" {
		c.Binary = "wine"
	}

	if c.PathBinary == "" {
		c.PathBinary = "winepath"
	}

	if c.ServerBinary == "" {
		c.ServerBinary = filepath.Join(filepath.Dir(c.Binary), "wineserver")
	}

	return c
}

// WithBase returns a copy of the configuration with each unset field filled in
// from base, so a specific configuration can inherit a shared installation while
// overriding individual fields. The receiver's set fields always win. It is
// typically chained before [Config.WithDefaults]:
//
//	effective := job.WithBase(service).WithDefaults()
func (c Config) WithBase(base Config) Config {
	if c.Binary == "" {
		c.Binary = base.Binary
	}

	if c.PathBinary == "" {
		c.PathBinary = base.PathBinary
	}

	if c.ServerBinary == "" {
		c.ServerBinary = base.ServerBinary
	}

	if c.Prefix == "" {
		c.Prefix = base.Prefix
	}

	return c
}

// Resolve returns a copy of the configuration with every tool path that contains
// a separator made absolute, so a relative tool path is not re-resolved against
// a different working directory once the caller changes into one. Bare command
// names (resolved on PATH) are left unchanged.
func (c Config) Resolve() (Config, error) {
	for _, tool := range []*string{&c.Binary, &c.PathBinary, &c.ServerBinary} {
		if strings.ContainsAny(*tool, "/\\") {
			resolved, err := absolutePath(*tool)
			if err != nil {
				return Config{}, fmt.Errorf("resolve Wine tool path: %w", err)
			}

			*tool = resolved
		}
	}

	return c, nil
}

// Env derives the subprocess environment from base. It selects this prefix via
// WINEPREFIX, appends winedbg.exe= to WINEDLLOVERRIDES so Wine's interactive
// crash debugger never blocks an unattended run (preserving any existing
// overrides), and defaults WINEDEBUG to errors only. The returned slice is
// sorted for deterministic output.
func (c Config) Env(base []string) ([]string, error) {
	values := make(map[string]string)

	for _, entry := range base {
		if key, value, ok := strings.Cut(entry, "="); ok {
			values[key] = value
		}
	}

	overrides := strings.TrimRight(values["WINEDLLOVERRIDES"], ";")
	if !strings.HasSuffix(";"+overrides, ";winedbg.exe=") {
		if overrides != "" {
			overrides += ";"
		}

		overrides += "winedbg.exe="
	}

	values["WINEDLLOVERRIDES"] = overrides
	if values["WINEDEBUG"] == "" {
		values["WINEDEBUG"] = "-all,err+all"
	}

	if c.Prefix != "" {
		prefix, err := filepath.Abs(HostPath(c.Prefix))
		if err != nil {
			return nil, fmt.Errorf("resolve Wine prefix: %w", err)
		}

		values["WINEPREFIX"] = prefix
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	env := make([]string, 0, len(keys))
	for _, key := range keys {
		env = append(env, key+"="+values[key])
	}

	return env, nil
}
