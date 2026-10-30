# AGENTS.md

Package `gameproc` finds, stops and waits for a game's processes on Linux (also from inside a Flatpak) and Windows,
and starts a child so its whole tree can be ended. Game launchers and mod managers import it. It imports no consumer
package, names no game, and carries no game data: every program name comes from the caller.

## Gate

`gate` is the command. Each override in `.gate.toml` is there because a runner
cannot resolve that check from PATH.

Only the `windows-latest` test job runs `exe_windows_test.go`, `console_windows_test.go`, `stop_windows_test.go`
and the Windows branch of `live_test.go` (Toolhelp listing, job objects, suspended start). `TestStopAsksAWineGameToCloseFirst` needs `/bin/sh`
and the Flatpak tests need `FLATPAK_ID`, so both are Unix-only. Real processes come from the test binary itself:
`TestMain` acts as a child when `GAMEPROC_HELPER` is set (`helper_test.go`).

No test reaches Wine and Proton `taskkill` behaviour, a real Flatpak host, macOS `WaitPID` and `ImagePath` (they
compile, nothing runs them), or `TrackTree`'s `ErrNotTracked` and resume-failure paths.

## Shape

- Matching rules and their three lists (`dotnet`/`mono` hosts, Wine loader names, system folders) are unexported and
  take no options: each entry is a general rule, not game data.
- A Linux name matches exactly; only `.exe` and `.dll` fold case. Folding everything makes `/bin/sh` a game called
  "SH". Only `argv[0]`, or `argv[1]` under dotnet/mono, names the program, or a launcher that lists the game in its
  arguments would be taken for the game.
- A program named without a path whose executable is in a system folder is skipped; one named by a path is judged by
  that path, because Wine names the game's `.exe` while its executable may be the system Wine.
- `Processes` switches to the host only for the literal `"/proc"` inside a Flatpak, so a fake tree in a test is never
  sent to the host. The host listing is two programs in all, never one per process.
- The Flatpak probe is the host's `sh` answering `alive` or `gone`, so a `flatpak-spawn` that cannot run is an error
  at every step and never reads as "gone". `hostOutput` is unexported so a stub in one importer's test cannot rewire
  another's.
- Every pid function refuses `pid <= 0` and Unix `KillTree` refuses `pid <= 1`: `kill(0, ...)` signals the caller's own
  group and `kill(-1, ...)` every process the user owns.
- The job registry is package state keyed by pid: `Terminate` has only a pid and must find the job. The price is that
  `UntrackTree` must follow `Wait`, or a reused pid finds a stale job.
- `PrepareTree` never sets HideWindow, and keeps one the caller set: Windows applies it to the first window the
  process shows, which for a loader that starts the game in a second process is the game's.
- `Status.Signal` strings come from `unix.SignalName` and callers may persist them, so they do not change.

## Files

| File | Holds |
| --- | --- |
| `doc.go` | Package doc |
| `types.go` | `Status`, `ErrNotTracked`, `ErrNotChild`, `ErrNoConsole` |
| `process.go` | `Process`, `RunsFrom`, `LinkedFrom`, `ExeIs`, path containment, `exitedWithin` |
| `system.go` | System-folder test that keeps system programs from matching |
| `processes_unix.go` | `Processes` and `Named` from `/proc` or the Flatpak host, start time |
| `processes_windows.go` | `Processes` and `Named` through Toolhelp |
| `imagepath_unix.go` | `ImagePath` from `/proc/<pid>/exe` |
| `stop_unix.go` | `Stop`, `Terminate`, `Alive`, the polite Wine close, host signalling in a Flatpak |
| `stop_windows.go` | `Stop`, `Terminate`, `Alive`, pid conversion |
| `waitpid_unix.go`, `wait_windows.go` | `WaitPID` |
| `wait_unix.go` | Unix `StatusOf` and signal names (Windows' is in `wait_windows.go`) |
| `tree_unix.go` | Process-group `PrepareTree`, `TrackTree`, `KillTree`, `UntrackTree` |
| `tree_windows.go` | Suspended start, job registry, resume |
| `console_windows.go` | `StartOwnConsole` and the environment block |
| `console_other.go` | `StartOwnConsole` off Windows |
