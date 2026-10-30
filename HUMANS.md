# HUMANS.md

Run and use `go-gameproc`. Internals: [AGENTS.md](AGENTS.md).

## Quick start

```bash
go get github.com/Rethunk-Tech/go-gameproc
```

```go
package main

import (
	"fmt"
	"os"

	"github.com/Rethunk-Tech/go-gameproc"
)

func main() {
	// List every running "MyGame" in one pass. On Windows the first argument is ignored.
	procs, err := gameproc.Processes("/proc", "MyGame")
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, p := range procs {
		if p.RunsFrom("/games/MyGame") {
			fmt.Println("running from the install:", p.PID, p.Exe)
		}
	}
	fmt.Println("this process is alive:", gameproc.Alive(os.Getpid()))
}
```

To start a program whose tree you may need to end, build the `exec.Cmd` (set `Dir` and `Env` yourself), call
`gameproc.PrepareTree(cmd, false)`, `cmd.Start()`, then `gameproc.TrackTree(cmd.Process.Pid)`. `KillTree` ends the tree
and `UntrackTree` follows `cmd.Wait()`.

## Contracts

| Call | Contract |
| --- | --- |
| `Processes` | One pass over the table. Linux names match exactly (`.exe`/`.dll` fold case); only `argv[0]`, or `argv[1]` under dotnet/mono, names the program; system programs never match. `procDir` is a `/proc`-shaped tree; only the literal `/proc` inside a Flatpak lists the host (needs `--talk-name=org.freedesktop.Flatpak`). Windows ignores `procDir`, leaves `Args` nil and keeps a process whose image path is unreadable with `Exe == ""`. |
| `Named`, `RunsFrom`, `LinkedFrom`, `ExeIs` | Pure judgements on a `Process`; `RunsFrom` reads a Wine `Z:\...` path as the Unix path it names. |
| `ImagePath` | `/proc/<pid>/exe` off Windows (an error on macOS), `QueryFullProcessImageName` on Windows. |
| `Alive`, `Terminate`, `Stop`, `WaitPID`, `ImagePath`, `TrackTree` | Refuse `pid <= 0`: `Alive` is false, the rest return an error. `KillTree` and `UntrackTree` do nothing for it, and `KillTree` also for pid 1 on Unix. |
| `Terminate` | Unix: SIGTERM, then SIGKILL after `grace`; a process already gone is success. Windows: ends the tracked job or the process at once and waits up to `grace`. In a Flatpak a failure to ask the host is an error, never "gone". |
| `Stop` | `Terminate`, after `wine taskkill` (no `/f`) in the process's own prefix environment for a Wine process (Linux, outside a Flatpak). |
| `WaitPID` | The exit status is readable only for the caller's own child. For any other process it polls until the exit and returns an empty `Status` and an error matching `ErrNotChild` (Flatpak: nil once the host says it is gone). |
| `StatusOf` | Maps `exec.Cmd.Wait`'s error to `Status`; a signal gives `Code` 128 plus the signal number and `Signal` its name (Unix). |
| `PrepareTree`, `TrackTree`, `KillTree`, `UntrackTree` | Unix: new process group, `KillTree` sends it SIGTERM. Windows: suspended start, a job per pid held in package state; call `UntrackTree` after `Wait`. `errors.Is(err, ErrNotTracked)` from `TrackTree` means resumed but outside a job. Never sets HideWindow; `noConsole` sets CREATE_NO_WINDOW. The module never touches `cmd.Env`. |
| `StartOwnConsole` | Windows only (`ErrNoConsole` elsewhere). `env` is the complete environment, empty meaning the caller's own. Takes no context: the program outlives the caller's request. |

Not testable without the real thing: Wine and Proton `taskkill` behaviour, a real Flatpak host, macOS `WaitPID` and
`ImagePath` (they compile, nothing runs them), and `TrackTree`'s `ErrNotTracked` and resume-failure paths.

## Requirements

Go 1.27.3 or newer, plus `lefthook` and `gitleaks` on PATH if you want the git hooks.

```bash
lefthook install
```

That is per clone; git does not commit hooks. Pre-commit scans staged changes for secrets, and
pre-push runs `gate`.

## Verify

```bash
gate
```
