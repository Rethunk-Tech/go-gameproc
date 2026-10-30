# Security

## Supported versions

The latest tagged release. There are no long-lived release branches, so a fix ships in the next tag
rather than being backported.

## Reporting a vulnerability

Use GitHub's **private vulnerability reporting** on this repository (Security → Report a
vulnerability). It stays private until a fix ships.

Please do not open a public issue for a suspected vulnerability first.

## What is in scope

- Process tables and command lines (`Processes`): another user's process controls its own `argv`. A crafted command
  line must not panic the parser, make a system program match as a game, or reach a path outside the tree handed in.
- The Flatpak host listing (`Processes("/proc", ...)`): its text comes from the host's `grep` and `stat` and is parsed
  with `Cut` and `Fields` only; malformed records are skipped, never indexed blindly.
- Pids (`Terminate`, `KillTree`, `Alive`, `WaitPID`, `TrackTree`, `ImagePath`): `0`, negative values and, for
  `KillTree`, `1` must never reach `kill(2)`, where they signal the caller's own group or every process the user owns.
- The process environment read for the polite Wine close (`Stop`): the variables are passed to `wine taskkill` as read,
  minus `WINESERVERSOCKET`, and only for a pid the caller named.

## What is out of scope

- The caller decides which process to stop. `Terminate` ends whatever process the pid names, if the OS permits; a
  caller that takes a pid from a file must check it is still the program it recorded (`ImagePath`).
- Program names, the install directory and the command to start are trusted: they are never run through a shell.
- Stopping a process the user owns, and a process that outlives its caller, are the purpose of the package.
