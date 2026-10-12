//nolint:testpackage // White-box tests exercise the package's execution seam.
package wine

import (
	"context"
	"errors"
	"testing"
)

func TestEnvironmentSetupRequiresPrefix(t *testing.T) {
	t.Parallel()

	_, _, err := Environment{}.Setup(context.Background(), nil)
	requireTruef(t, errors.Is(err, ErrNoPrefix), "expected ErrNoPrefix, got %v", err)
}

func TestEnvironmentSetupBuildsEnvAndTeardown(t *testing.T) {
	t.Parallel()

	fake := newExecutor()
	environment := Environment{Config: Config{Prefix: testPrefix}, Exec: fake.run}

	env, teardown, err := environment.Setup(context.Background(), nil)
	requireNoError(t, err)
	requireTruef(
		t,
		envMap(t, env)["WINEPREFIX"] != "",
		"environment is missing WINEPREFIX: %v",
		env,
	)
	requireTruef(t, teardown != nil, "expected a non-nil teardown")

	requireNoError(t, teardown(context.Background()))
	requireEqual(t, len(fake.calls), 2, "teardown should stop the server")
	requireEqual(t, fake.calls[0].step, "Wine server shutdown", "teardown shutdown step")
}

func TestEnvironmentTranslatePath(t *testing.T) {
	t.Parallel()

	fake := newExecutor()
	fake.stdout[winepathTool] = "C:\\in.exe\n"
	environment := Environment{Config: Config{Prefix: testPrefix}, Exec: fake.run}

	result, err := environment.TranslatePath(context.Background(), "/host/in.exe", nil)
	requireNoError(t, err)
	requireEqual(t, result, "C:\\in.exe", "translated path")
}

func TestEnvironmentCommandWrapsWithLoader(t *testing.T) {
	t.Parallel()

	environment := Environment{Config: Config{Binary: wineTool, Prefix: testPrefix}}

	executable, prefix, err := environment.Command("Themida.exe")
	requireNoError(t, err)
	requireEqual(t, executable, wineTool, "loader executable")
	requireSlice(t, prefix, []string{"Themida.exe"}, "command prefix")
}
