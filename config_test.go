//nolint:testpackage // White-box tests exercise the package's execution seam.
package wine

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestConfigWithDefaults(t *testing.T) {
	t.Parallel()

	defaults := Config{}.WithDefaults()
	requireEqual(t, defaults.Binary, wineTool, "binary default")
	requireEqual(t, defaults.PathBinary, winepathTool, "path binary default")
	requireEqual(t, defaults.ServerBinary, wineserverTool, "server binary default")

	beside := Config{Binary: filepath.Join("opt", "wine", "bin", "wine")}.WithDefaults()
	requireEqual(
		t,
		beside.ServerBinary,
		filepath.Join("opt", "wine", "bin", "wineserver"),
		"server binary beside loader",
	)
}

func TestConfigResolveMakesSeparatorPathsAbsolute(t *testing.T) {
	t.Parallel()

	resolved, err := Config{
		Binary:     wineTool,
		PathBinary: filepath.Join(".", "tools", "winepath"),
	}.Resolve()
	requireNoError(t, err)
	requireEqual(t, resolved.Binary, wineTool, "bare binary name unchanged")
	requireTruef(
		t,
		filepath.IsAbs(resolved.PathBinary),
		"path binary with a separator was not made absolute: %q",
		resolved.PathBinary,
	)
}

func TestConfigEnv(t *testing.T) {
	t.Parallel()

	env, err := Config{Prefix: "wineprefix"}.Env([]string{
		"WINEDLLOVERRIDES=mscoree=d",
		"PATH=/usr/bin",
	})
	requireNoError(t, err)
	requireTruef(t, slices.IsSorted(env), "environment is not sorted: %v", env)

	values := envMap(t, env)
	requireEqual(
		t,
		values["WINEDLLOVERRIDES"],
		"mscoree=d;winedbg.exe=",
		"dll overrides preserved and appended",
	)
	requireEqual(t, values["WINEDEBUG"], "-all,err+all", "wine debug default")
	requireEqual(t, values["PATH"], "/usr/bin", "unrelated variables preserved")

	absPrefix, err := filepath.Abs("wineprefix")
	requireNoError(t, err)
	requireEqual(t, values["WINEPREFIX"], absPrefix, "prefix resolved to absolute")
}

func TestConfigEnvKeepsExistingDebug(t *testing.T) {
	t.Parallel()

	env, err := Config{Prefix: testPrefix}.Env([]string{"WINEDEBUG=+relay"})
	requireNoError(t, err)
	requireEqual(t, envMap(t, env)["WINEDEBUG"], "+relay", "existing wine debug retained")
}
