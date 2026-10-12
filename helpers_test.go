//nolint:testpackage // White-box tests exercise the package's execution seam.
package wine

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

const (
	wineTool       = "wine"
	wineserverTool = "wineserver"
	winepathTool   = "winepath"
	testPrefix     = "prefix"
)

// errShutdown is a static sentinel used where a test injects a failure.
var errShutdown = errors.New("shutdown boom")

// recordedCall captures one invocation passed to the fake executor.
type recordedCall struct {
	step    string
	args    []string
	allowed []int
}

// fakeExecutor is an [Executor] that records each call, writes canned stdout for
// a step to the capture writer, and returns a canned error for a step.
type fakeExecutor struct {
	calls  []recordedCall
	stdout map[string]string
	fail   map[string]error
}

// newExecutor returns a fake executor with initialized tables.
func newExecutor() *fakeExecutor {
	return &fakeExecutor{calls: nil, stdout: map[string]string{}, fail: map[string]error{}}
}

func (f *fakeExecutor) run(
	_ context.Context, step string, cmd *exec.Cmd, capture io.Writer, allowed ...int,
) error {
	f.calls = append(f.calls, recordedCall{step: step, args: cmd.Args, allowed: allowed})

	if out, ok := f.stdout[step]; ok && capture != nil {
		_, _ = io.WriteString(capture, out)
	}

	return f.fail[step]
}

// requireNoError fails the test immediately when err is non-nil.
func requireNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// requireEqual fails the test when got and want differ.
func requireEqual[T comparable](t *testing.T, got, want T, label string) {
	t.Helper()

	if got != want {
		t.Fatalf("%s: got %v, want %v", label, got, want)
	}
}

// requireSlice fails the test when two slices differ.
func requireSlice[T comparable](t *testing.T, got, want []T, label string) {
	t.Helper()

	if !slices.Equal(got, want) {
		t.Fatalf("%s: got %v, want %v", label, got, want)
	}
}

// requireTruef fails the test with a formatted message when ok is false.
func requireTruef(t *testing.T, ok bool, format string, args ...any) {
	t.Helper()

	if !ok {
		t.Fatalf(format, args...)
	}
}

// envMap turns a KEY=VALUE environment slice into a map.
func envMap(t *testing.T, env []string) map[string]string {
	t.Helper()

	values := make(map[string]string, len(env))
	for _, entry := range env {
		if key, value, ok := strings.Cut(entry, "="); ok {
			values[key] = value
		}
	}

	return values
}
