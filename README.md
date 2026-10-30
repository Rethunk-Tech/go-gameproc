<h1 align="center">go-gameproc</h1>

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.27.3+-00ADD8?logo=go&logoColor=white)](go.mod)

</div>

---

Find, stop and wait for a game's processes, natively or under Wine, Proton, dotnet or mono, on Linux and Windows. Package `gameproc`, module `github.com/Rethunk-Tech/go-gameproc`.

A game can run natively, under Wine or Proton, or through dotnet or mono, and a launcher or mod manager must tell
which install a process belongs to, never mistake a system program or Steam for the game, and stop it cleanly. This
package does that from the process table alone: no game data, no options, names come from the caller.

## Quick start

```bash
go get github.com/Rethunk-Tech/go-gameproc
```

A worked example and the contracts: [HUMANS.md](HUMANS.md).

## Features

- List processes by program name in one pass, with executable, command line and start time, including from inside a Flatpak.
- Tell which install a process runs from, by executable, argument, Wine `Z:` path or symlinked deploy.
- Stop politely: a Wine process is asked to close first, so a launcher sees a clean exit.
- Wait for any pid and read its exit code and signal.
- Start a child so its whole tree ends together: a process group, or a job object on Windows.
- Start a Windows console program in a console of its own.

## Documentation

| Doc | Audience |
| --- | --- |
| [HUMANS.md](HUMANS.md) | Running and using the package |
| [AGENTS.md](AGENTS.md) | Internals, invariants, and why |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Process — commits, tests, docs |
| [SECURITY.md](SECURITY.md) | Vulnerability disclosure |
| [CHANGELOG.md](CHANGELOG.md) | Release notes |

## License

MIT — see [LICENSE](LICENSE). © 2026 Rethunk.Tech, LLC.
