//nolint:testpackage // White-box tests exercise the package's execution seam.
package wine

import (
	"context"
	"errors"
	"testing"
)

func TestWindowsPathTrimsAndCaptures(t *testing.T) {
	t.Parallel()

	fake := newExecutor()
	fake.stdout[winepathTool] = "Z:\\app\\out.exe\r\n"

	result, err := Config{PathBinary: winepathTool}.
		WindowsPath(context.Background(), fake.run, "/host/out.exe", nil)
	requireNoError(t, err)
	requireEqual(t, result, "Z:\\app\\out.exe", "converted path")
	requireEqual(t, len(fake.calls), 1, "winepath call count")
	requireEqual(t, fake.calls[0].step, winepathTool, "step name")
	requireSlice(
		t,
		fake.calls[0].args,
		[]string{winepathTool, "-w", "/host/out.exe"},
		"winepath args",
	)
}

func TestWindowsPathRejectsEmptyResult(t *testing.T) {
	t.Parallel()

	fake := newExecutor()
	fake.stdout[winepathTool] = "\r\n"

	_, err := Config{PathBinary: winepathTool}.
		WindowsPath(context.Background(), fake.run, "/host/out.exe", nil)
	requireTruef(t, errors.Is(err, ErrInvalidPath), "expected ErrInvalidPath, got %v", err)
}

func TestStopAllowsExitOneThenWaits(t *testing.T) {
	t.Parallel()

	fake := newExecutor()

	err := Config{ServerBinary: wineserverTool}.Stop(context.Background(), fake.run, nil)
	requireNoError(t, err)
	requireEqual(t, len(fake.calls), 2, "stop call count")
	requireEqual(t, fake.calls[0].step, "Wine server shutdown", "shutdown step")
	requireSlice(t, fake.calls[0].args, []string{wineserverTool, "-k"}, "shutdown args")
	requireSlice(t, fake.calls[0].allowed, []int{1}, "shutdown allows exit 1")
	requireEqual(t, fake.calls[1].step, "Wine server exit", "wait step")
	requireSlice(t, fake.calls[1].args, []string{wineserverTool, "-w"}, "wait args")
	requireEqual(t, len(fake.calls[1].allowed), 0, "wait allows no non-zero exit")
}

func TestStopStopsOnShutdownFailure(t *testing.T) {
	t.Parallel()

	fake := newExecutor()
	fake.fail["Wine server shutdown"] = errShutdown

	err := Config{ServerBinary: wineserverTool}.Stop(context.Background(), fake.run, nil)
	requireTruef(t, errors.Is(err, errShutdown), "expected shutdown error, got %v", err)
	requireEqual(t, len(fake.calls), 1, "wait must not run after shutdown failure")
}

func TestCommandSetsWaitDelay(t *testing.T) {
	t.Parallel()

	cmd := Command(context.Background(), wineTool, "program.exe")
	requireTruef(t, cmd.WaitDelay > 0, "expected a positive WaitDelay, got %v", cmd.WaitDelay)
	requireSlice(t, cmd.Args, []string{wineTool, "program.exe"}, "command args")
}
