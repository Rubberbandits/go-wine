package wine

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

// Environment bundles a [Config] with an execution policy into a ready-to-use
// implementer of a host tool's execution hook. Its Setup, TranslatePath, and
// Command methods have the shape such a hook expects, so a Wine run plugs into a
// host (such as go-themida's Runner) without either package importing the other.
//
// The zero value is not usable: Config.Prefix must be set. Exec and the output
// writers are optional.
type Environment struct {
	// Config names the Wine tools and prefix. Tool names are defaulted and tool
	// paths resolved automatically; Prefix must be set.
	Config Config
	// Exec runs the winepath and wineserver helper commands. Nil uses a built-in
	// runner that streams to Stdout and Stderr.
	Exec Executor
	// Stdout and Stderr receive the helper commands' output when Exec is nil.
	// Nil discards it; the output is always retained in error messages.
	Stdout io.Writer
	Stderr io.Writer
}

// Environment implements the execution hook that go-themida's Runner.Environment
// field expects. The inline interface mirrors that contract and fails to compile
// if the method set drifts; keep the two in sync.
var _ interface {
	Setup(ctx context.Context, base []string) (env []string, teardown func(context.Context) error, err error)
	TranslatePath(ctx context.Context, path string, env []string) (translated string, err error)
	Command(program string) (executable string, prefix []string, err error)
} = Environment{}

// Setup prepares the Wine environment for one run. It requires a prefix and
// returns the subprocess environment derived from base, plus a teardown that
// shuts the prefix's Wine server down. The teardown may run after the host's
// context is cancelled, so the host should give it an independent, bounded
// context.
func (e Environment) Setup(
	_ context.Context,
	base []string,
) ([]string, func(context.Context) error, error) {
	cfg, err := e.config()
	if err != nil {
		return nil, nil, err
	}

	if cfg.Prefix == "" {
		return nil, nil, fmt.Errorf("%w: a dedicated prefix is required", ErrNoPrefix)
	}

	env, err := cfg.Env(base)
	if err != nil {
		return nil, nil, err
	}

	run := e.executor()
	teardown := func(ctx context.Context) error { return cfg.Stop(ctx, run, env) }

	return env, teardown, nil
}

// TranslatePath converts a host path to the Windows path the program expects,
// using env from [Environment.Setup].
func (e Environment) TranslatePath(ctx context.Context, path string, env []string) (string, error) {
	cfg, err := e.config()
	if err != nil {
		return "", err
	}

	return cfg.WindowsPath(ctx, e.executor(), path, env)
}

// Command returns the executable to launch and the arguments that precede the
// program's own arguments: the wine loader, with the program as its first
// argument.
func (e Environment) Command(program string) (string, []string, error) {
	cfg, err := e.config()
	if err != nil {
		return "", nil, err
	}

	return cfg.Binary, []string{program}, nil
}

// config returns the defaulted, resolved configuration used for a run.
func (e Environment) config() (Config, error) {
	return e.Config.WithDefaults().Resolve()
}

// executor returns Exec, or a built-in runner that streams helper output to
// Stdout and Stderr.
func (e Environment) executor() Executor {
	if e.Exec != nil {
		return e.Exec
	}

	return func(ctx context.Context, step string, cmd *exec.Cmd, capture io.Writer, allowed ...int) error {
		return stream(ctx, step, cmd, capture, e.Stdout, e.Stderr, allowed...)
	}
}
