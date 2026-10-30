# Changelog

Notable changes to `go-gameproc`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-10-29

### Added

- `Processes` lists the running processes whose program is one of the given names in one pass: executable, command
  line and start time, from a `/proc`-shaped tree, from the host through `flatpak-spawn` inside a Flatpak, or from
  Toolhelp on Windows. It never mistakes a system program, or a launcher that names the program in its arguments,
  for the program.
- `Process.Named`, `RunsFrom`, `LinkedFrom` and `ExeIs` tell which install a process runs from: by executable, by an
  argument, by a Wine `Z:` path, or through a symlinked deploy.
- `Alive`, `Terminate`, `Stop` and `ImagePath` check, end and identify a process by pid. `Stop` asks a Wine process
  to close first, so a launcher that tracks it sees a clean exit.
- `WaitPID` and `StatusOf` wait for any pid and report its exit code and signal; a Unix process that is not a child
  gives `ErrNotChild` once it is gone.
- `PrepareTree`, `TrackTree`, `KillTree` and `UntrackTree` start a child so that it and everything it starts can be
  ended together: a process group on Unix, a job object on Windows.
- `StartOwnConsole` starts a Windows console program in a console that it reads and writes itself.

[Unreleased]: https://github.com/Rethunk-Tech/go-gameproc/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Rethunk-Tech/go-gameproc/releases/tag/v0.1.0
