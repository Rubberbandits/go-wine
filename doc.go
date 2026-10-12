// Package wine runs Windows programs through [Wine] on a non-Windows host.
//
// Wine (https://www.winehq.org) is a compatibility layer that runs Windows
// executables on Unix-like systems. Driving a Windows tool through Wine from Go
// means more than spawning the wine loader: a run needs a prefix (the simulated
// Windows environment) selected and isolated, host paths translated to the
// Windows paths the tool expects, and the Wine server shut down afterward so the
// prefix is left idle.
//
// # Primitives
//
// [Config] names the Wine tools and the prefix to use. Its methods are the
// building blocks: [Config.WithDefaults] fills in tool names, [Config.Resolve]
// makes relative tool paths absolute, [Config.Env] derives the subprocess
// environment, [Config.WindowsPath] converts a host path with winepath, and
// [Config.Stop] shuts the prefix's server down. The execution primitives take an
// [Executor] so the caller controls how processes are run, logged, and
// cancelled; [Run] is a ready-made default.
//
// # Host integration
//
// [Environment] bundles a [Config] with an [Executor] and output writers into a
// single value whose Setup, TranslatePath, and Command methods match the
// execution hook that host tools (such as go-themida) expose, so a Wine run can
// be plugged into one without either package importing the other.
package wine
