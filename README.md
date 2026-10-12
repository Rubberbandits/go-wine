# go-wine

[![Go Reference](https://pkg.go.dev/badge/github.com/Rubberbandits/go-wine.svg)](https://pkg.go.dev/github.com/Rubberbandits/go-wine)
[![CI](https://github.com/Rubberbandits/go-wine/actions/workflows/ci.yml/badge.svg)](https://github.com/Rubberbandits/go-wine/actions/workflows/ci.yml)

A small Go package for running Windows programs through [Wine](https://www.winehq.org)
on a non-Windows host.

Driving a Windows tool through Wine is more than spawning the `wine` loader: a
run needs a prefix (the simulated Windows environment) selected and isolated,
host paths translated to the Windows paths the tool expects, and the Wine server
shut down afterward so the prefix is left idle. This package packages those steps
behind a small, dependency-free API, and exposes them both as building blocks and
as a ready-to-plug execution environment for host tools.

## Features

- **Prefix-aware environment.** Build the subprocess environment with the prefix
  selected, Wine's interactive crash debugger disabled, and diagnostics quieted.
- **Path translation.** Convert host paths to Windows paths with `winepath`.
- **Clean shutdown.** Stop the prefix's Wine server and confirm it has exited.
- **Pluggable execution.** The helpers take an `Executor` so you control how
  processes are run, logged, and cancelled; a ready-made default is included.
- **Host integration.** An `Environment` value satisfies the execution hook that
  host tools expose (such as go-themida's `Runner`), with no import coupling in
  either direction.
- **Prefix and path helpers.** `Config.WithBase` layers a configuration over a
  shared installation, `SamePrefix` reports whether two prefixes resolve to the
  same location (symlinks and the Wine default included), and `HostPath`
  normalizes Windows-style separators for the host.
- **No external dependencies.** Standard library only.

## Requirements

- Go 1.24.2 or newer.
- Wine installed on the host to actually run anything (`wine`, `winepath`, and
  `wineserver` on `PATH`, or explicit paths). Building environments and
  computing the subprocess environment needs neither.

## Installation

```sh
go get github.com/Rubberbandits/go-wine
```

## Configure a prefix

[`Config`](https://pkg.go.dev/github.com/Rubberbandits/go-wine#Config) names the
Wine tools and the prefix. Tool names default to `wine`, `winepath`, and the
`wineserver` beside `wine`; the prefix has no default and must be set. The prefix
must be dedicated to this work, because shutting it down stops every process in
it.

```go
cfg := wine.Config{Prefix: "/srv/wine/app"}.WithDefaults()

env, err := cfg.Env(os.Environ())
if err != nil {
	return err
}
// env selects WINEPREFIX, disables winedbg, and defaults WINEDEBUG to errors only.
```

## Run the helpers

The execution helpers take an [`Executor`](https://pkg.go.dev/github.com/Rubberbandits/go-wine#Executor).
[`Run`](https://pkg.go.dev/github.com/Rubberbandits/go-wine#Run) is the built-in
default; pass your own to integrate custom logging, streaming, or a process
registry.

```go
// Translate a host path to the Windows path the program expects.
windows, err := cfg.WindowsPath(ctx, wine.Run, "/srv/app/input.dat", env)
if err != nil {
	return err
}

// Later, shut the prefix's server down and confirm it exited.
if err := cfg.Stop(ctx, wine.Run, env); err != nil {
	return err
}
```

[`Command`](https://pkg.go.dev/github.com/Rubberbandits/go-wine#Command) builds a
context-bound `*exec.Cmd` with a bounded wait delay and process-tree cancellation
applied, if you need to assemble an invocation yourself (for example the `wine`
loader followed by the program and its arguments).

## As a host execution environment

[`Environment`](https://pkg.go.dev/github.com/Rubberbandits/go-wine#Environment)
bundles a `Config` with an execution policy into a value whose `Setup`,
`TranslatePath`, and `Command` methods match the execution hook that host tools
expose. It plugs a Wine run into such a tool without either package importing the
other.

For example, [go-themida](https://github.com/Rubberbandits/go-themida)'s `Runner`
has an `Environment` field of a matching interface type, so running the Themida
protector through Wine is just:

```go
runner := themida.Runner{
	Path: "/opt/themida",
	Environment: wine.Environment{
		Config: wine.Config{Prefix: "/srv/wine/themida"},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	},
}
```

`Setup` requires a prefix, returns the subprocess environment, and provides a
teardown that stops the server. `TranslatePath` runs `winepath`. `Command` wraps
the program with the `wine` loader. Set `Exec` to run the helper commands through
your own executor; nil streams their output to `Stdout` and `Stderr`.

## Development

Use golangci-lint 2.14.0 or newer. The repository configuration enables all
current linters and all six formatters.

```sh
go test -race ./...
go vet ./...
golangci-lint fmt
golangci-lint run ./...
```

The tests run without Wine installed: the execution helpers are exercised through
an injected executor, so they verify environment construction, tool resolution,
path translation, server shutdown, and the host-environment wiring without
launching real processes.

## License

Released into the public domain under the [Unlicense](LICENSE).
