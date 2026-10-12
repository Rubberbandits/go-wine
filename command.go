package wine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"time"
)

// Executor runs a single prepared command to completion, letting the caller
// control how Wine's helper processes are run, logged, and cancelled.
//
// The command is unstarted: the implementation must run it and wait for it to
// finish. ctx is the job context and must be honored for cancellation. step is a
// short human-readable name for the step, suitable for error wrapping. capture,
// when non-nil, must receive the command's stdout only (winepath output is read
// this way). allowedExitCodes lists non-zero exit codes to treat as success, and
// apply only to normal process exits, never to cancellation or startup failures.
// [Run] is the package's default implementation.
type Executor func(
	ctx context.Context, step string, cmd *exec.Cmd, capture io.Writer, allowedExitCodes ...int,
) error

// processWaitDelay bounds how long Run waits for inherited output handles to
// close after the process exits, so a tool that leaves handles open cannot block
// indefinitely.
const processWaitDelay = 2 * time.Second

// Command builds a context-bound command with the shared wait delay and the
// platform-specific process configuration (window hiding and process-tree
// cancellation) applied, so a cancelled run tears down the Wine process tree
// rather than leaking wineserver and its children. No shell is involved.
func Command(ctx context.Context, executable string, args ...string) *exec.Cmd {
	// #nosec G204 -- The caller explicitly chooses the executable and arguments; no shell is used.
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.WaitDelay = processWaitDelay
	configureProcess(ctx, cmd)

	return cmd
}

// Run is the default [Executor]. It runs cmd with no extra output destinations
// beyond the optional stdout capture, retaining a bounded tail of the combined
// output to include in any error. A cancelled context takes priority over the
// process error, an exit code in allowed is treated as success, and a tool that
// exits cleanly while leaving output handles open is not an error.
func Run(
	ctx context.Context, step string, cmd *exec.Cmd, capture io.Writer, allowed ...int,
) error {
	return stream(ctx, step, cmd, capture, nil, nil, allowed...)
}

// stream runs cmd, teeing stdout and stderr to the diagnostic tail and the
// optional stdout/stderr writers, and stdout additionally to capture. It is the
// shared implementation behind [Run] and [Environment]'s executor.
func stream(
	ctx context.Context,
	step string,
	cmd *exec.Cmd,
	capture, stdout, stderr io.Writer,
	allowed ...int,
) error {
	{
		err := ctx.Err()
		if err != nil {
			return fmt.Errorf("%s: %w", step, err)
		}
	}

	tail := &diagnosticTail{}
	attachStreams(cmd, capture, stdout, stderr, tail)

	err := commandResult(ctx, cmd.Run())
	if err == nil || allowedExit(ctx, err, allowed) {
		return nil
	}

	return fmt.Errorf("%s: %w: %s", step, err, tail.String())
}

// attachStreams wires the command's stdout and stderr to the diagnostic tail,
// the optional stdout/stderr writers, and the optional stdout capture, all
// serialized through one mutex so interleaved writes from both streams stay
// intact.
func attachStreams(cmd *exec.Cmd, capture, stdout, stderr io.Writer, tail *diagnosticTail) {
	outs := []io.Writer{tail}
	errs := []io.Writer{tail}

	if stdout != nil {
		outs = append(outs, stdout)
	}

	if stderr != nil {
		errs = append(errs, stderr)
	}

	if capture != nil {
		outs = append(outs, capture)
	}

	var streams sync.Mutex

	cmd.Stdout = &streamWriter{mu: &streams, writer: io.MultiWriter(outs...)}
	cmd.Stderr = &streamWriter{mu: &streams, writer: io.MultiWriter(errs...)}
}

// commandResult normalizes a completed Run: a cancelled context takes priority
// over the raw error, and a WaitDelay expiry (a tool exiting cleanly while
// leaving output handles open) is treated as success.
func commandResult(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("command cancelled: %w", ctx.Err())
	}
	// A successful tool may leave inherited output handles open; bound pipe draining.
	if errors.Is(err, exec.ErrWaitDelay) {
		return nil
	}

	return err
}

// allowedExit reports whether err is a normal process exit (not a cancellation)
// whose exit code is in allowed.
func allowedExit(ctx context.Context, err error, allowed []int) bool {
	var exitErr *exec.ExitError

	return ctx.Err() == nil && errors.As(err, &exitErr) &&
		slices.Contains(allowed, exitErr.ExitCode())
}

// WindowsPath converts a host path to the Windows path a Wine program expects,
// by running winepath -w in this configuration's prefix via run and reading its
// stdout. env is the subprocess environment, normally from [Config.Env]. The
// result is rejected with [ErrInvalidPath] if it is empty or contains a line
// break or NUL.
func (c Config) WindowsPath(
	ctx context.Context,
	run Executor,
	path string,
	env []string,
) (string, error) {
	cmd := Command(ctx, c.PathBinary, "-w", path)
	cmd.Env = env

	var stdout strings.Builder

	err := run(ctx, "winepath", cmd, &stdout)
	if err != nil {
		return "", fmt.Errorf("convert Wine path %q: %w", path, err)
	}

	result := strings.TrimRight(stdout.String(), "\r\n")
	if result == "" || strings.ContainsAny(result, "\r\n\x00") {
		return "", fmt.Errorf("%w: winepath returned an invalid path for %q", ErrInvalidPath, path)
	}

	return result, nil
}

// Stop shuts this configuration's Wine server down with wineserver -k and
// confirms it has fully exited with wineserver -w, both run via run with the
// given env. Exit code 1 from -k also means no server was running, so it is
// allowed; the subsequent -w must still succeed to confirm the prefix is idle
// before it is reused.
func (c Config) Stop(ctx context.Context, run Executor, env []string) error {
	kill := Command(ctx, c.ServerBinary, "-k")
	kill.Env = env
	// Exit 1 also means no server was running; a successful -w must confirm it.
	err := run(ctx, "Wine server shutdown", kill, nil, 1)
	if err != nil {
		return err
	}

	wait := Command(ctx, c.ServerBinary, "-w")
	wait.Env = env

	return run(ctx, "Wine server exit", wait, nil)
}

// diagnosticTail is a thread-safe io.Writer that retains only the last 64 KiB
// written, providing a bounded excerpt of a tool's output to include in errors.
type diagnosticTail struct {
	mu   sync.Mutex
	data []byte
}

// Write appends to the tail, discarding the oldest bytes so the retained data
// never exceeds the limit. It never returns an error and reports the full input
// length as written.
func (t *diagnosticTail) Write(data []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	const limit = 64 * 1024

	size := len(data)
	if size >= limit {
		t.data = append(t.data[:0], data[size-limit:]...)
	} else {
		if len(t.data)+size > limit {
			t.data = t.data[len(t.data)+size-limit:]
		}

		t.data = append(t.data, data...)
	}

	return size, nil
}

// String returns the retained output with surrounding whitespace trimmed.
func (t *diagnosticTail) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	return strings.TrimSpace(string(t.data))
}

// streamWriter serializes writes to an underlying writer through a shared mutex,
// so stdout and stderr delivered to the same destination do not interleave
// mid-write.
type streamWriter struct {
	mu     *sync.Mutex
	writer io.Writer
}

// Write forwards data to the underlying writer under the shared lock.
func (w *streamWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	size, err := w.writer.Write(data)
	if err != nil {
		return size, fmt.Errorf("stream command output: %w", err)
	}

	return size, nil
}
