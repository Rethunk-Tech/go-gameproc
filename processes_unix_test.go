//go:build !windows

package gameproc

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
)

func TestProcesses(t *testing.T) {
	proc := t.TempDir()
	add := func(pid, cmdline string) {
		dir := filepath.Join(proc, pid)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte(cmdline), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	add("101", "/g/Loader\x00--mods-path\x00/p/a/mods\x00")
	add("102", "/g/Loader\x00--mods-path=/p/b/mods\x00")
	add("103", "/usr/bin/vim\x00Loader\x00")
	add("104", "dotnet\x00/g/Loader.dll\x00--mods-path\x00/p/c/mods\x00")
	add("105", "/g/Loader\x00")
	if err := os.MkdirAll(filepath.Join(proc, "self"), 0o700); err != nil {
		t.Fatal(err)
	}
	procs, err := Processes(proc, "Loader")
	if err != nil || len(procs) != 4 {
		t.Fatalf("procs = %+v, %v", procs, err)
	}
	add("107", "/bin/sh\x00-c\x00steam-runtime-launch\x00")
	add("108", `Z:\\g\\SH.EXE`+"\x00")
	if sh, err := Processes(proc, "SH"); err != nil || len(sh) != 1 || sh[0].PID != 108 {
		t.Fatalf("a shell is not Game's SH, its Windows exe is: %+v, %v", sh, err)
	}
	add("106", "/g/Game\x00")
	if both, err := Processes(proc, "Game", "Loader"); err != nil || len(both) != 5 {
		t.Fatalf("one pass for two names = %+v, %v", both, err)
	}
}

func TestProcessStartTime(t *testing.T) {
	proc := t.TempDir()
	dir := filepath.Join(proc, "7")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// comm holds a space and a parenthesis; starttime is the 22nd field.
	stat := "7 (Game (x) y) S 1 7 7 0 -1 0 0 0 0 0 0 0 0 0 20 0 1 0 12345 0 0\n"
	files := map[string]string{
		"7/cmdline": "/g/Loader\x00", "7/stat": stat, "stat": "cpu 1 2\nbtime 1000000\n",
	}
	for rel, body := range files {
		if err := os.WriteFile(filepath.Join(proc, rel), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	procs, err := Processes(proc, "Loader")
	if err != nil || len(procs) != 1 {
		t.Fatalf("procs = %+v, %v", procs, err)
	}
	if want := time.Unix(1000000+123, 0); !procs[0].Start.Equal(want) {
		t.Fatalf("start = %v, want %v", procs[0].Start, want)
	}
}

func TestHostScriptListsRealProcesses(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("reads /proc")
	}
	// Matched by its arguments: the test may run in a PID namespace whose pids /proc does not show.
	// read is a builtin, so the shell waits on its stdin with no child to leave behind.
	cmd := exec.CommandContext(t.Context(), "sh", "-c", "read -r _", "--mods-path", "/m\nx:y", "")
	if _, err := cmd.StdinPipe(); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	out, err := exec.CommandContext(t.Context(), "sh", "-c", hostScript).Output()
	if err != nil {
		t.Fatal(err)
	}
	got := parseHostProcesses(string(out), "sh")
	i := slices.IndexFunc(got, func(p Process) bool { return slices.Equal(p.Args, cmd.Args) })
	if i < 0 || got[i].PID <= 0 || time.Since(got[i].Start) > time.Minute {
		t.Fatalf("not listed with args %q: %+v", cmd.Args, got)
	}
}

func TestHostTerminateSignalsThroughHostKill(t *testing.T) {
	var calls []string
	t.Setenv("FLATPAK_ID", "x")
	old := hostOutput
	t.Cleanup(func() { hostOutput = old })
	alive := true
	hostOutput = func(name string, args ...string) ([]byte, error) {
		call := name
		if name != "sh" {
			call = args[0]
		}
		calls = append(calls, call)
		if name == "sh" {
			if alive {
				return []byte("alive\n"), nil
			}
			return []byte("gone\n"), nil
		}
		alive = call != "-TERM"
		return nil, nil
	}
	if err := Terminate(42, time.Second); err != nil {
		t.Fatal(err)
	}
	want := []string{"sh", "-TERM", "sh"}
	if !slices.Equal(calls, want) {
		t.Fatalf("calls %q", calls)
	}
}

func TestHostTerminateReportsAFailureToRunFlatpakSpawn(t *testing.T) {
	t.Setenv("FLATPAK_ID", "x")
	old := hostOutput
	t.Cleanup(func() { hostOutput = old })
	hostOutput = func(string, ...string) ([]byte, error) { return nil, errors.New("flatpak-spawn: not found") }
	if err := Terminate(42, time.Second); err == nil {
		t.Fatal("a game that may still run was reported stopped")
	}
	if _, err := WaitPID(42); err == nil {
		t.Fatal("WaitPID reported a process gone when the host could not be asked")
	}
}

func TestStopAsksAWineGameToCloseFirst(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pidFile, closeLog := filepath.Join(dir, "pid"), filepath.Join(dir, "close")
	wine := "#!/bin/sh\necho \"$@\" $WINESERVERSOCKET >\"$CLOSE_LOG\"\nkill \"$(cat \"$PID_FILE\")\"\n"
	if err := safefs.WriteFile(filepath.Join(dir, "wine"), []byte(wine), 0o700); err != nil {
		t.Fatal(err)
	}
	game := exec.CommandContext(t.Context(), "sh", "-c", `echo $$ >"$PID_FILE"; trap "exit 0" TERM; while :; do sleep 0.1; done`)
	game.Env = append(os.Environ(), "PID_FILE="+pidFile, "CLOSE_LOG="+closeLog, "WINESERVERSOCKET=34")
	if err := game.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { _ = game.Wait() }()
	for range 50 {
		if _, err := os.Stat(pidFile); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	p := Process{PID: game.Process.Pid, Exe: filepath.Join(dir, "wine64-preloader"), Args: []string{`C:\Games\My Game.exe`}}
	if err := Stop(t.Context(), p, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	got, _ := safefs.ReadFile(closeLog)
	if strings.TrimSpace(string(got)) != "taskkill /im My Game.exe" {
		t.Fatalf("wine ran with %q", got)
	}
	if wineLoader("/g/My Game.exe") != "" {
		t.Fatal("a game's own executable is not a Wine loader")
	}
}

// A mod manager that deploys a loader into the game folder as a link into its staging folder is running the link's
// target, which is what the process names.
func TestLinkedFromMatchesALoaderDeployedAsALink(t *testing.T) {
	game, staging := t.TempDir(), t.TempDir()
	target := filepath.Join(staging, "Loader 4.3.2", "Loader")
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(game, "Loader")); err != nil {
		t.Fatal(err)
	}
	p := Process{PID: 1, Exe: target}
	if p.RunsFrom(game) {
		t.Fatal("the target is outside the game folder; RunsFrom should not claim it")
	}
	if !p.LinkedFrom(game) {
		t.Fatal("the game folder's Loader links to the running executable")
	}
	if (Process{PID: 2, Exe: filepath.Join(t.TempDir(), "Loader")}).LinkedFrom(game) {
		t.Fatal("an unrelated executable of the same name matched")
	}
}

// Stopping or cancelling a game ends the processes that run it, never Steam or the wrappers Steam starts it through,
// though those name the game in their arguments.
func TestSteamAndItsWrappersAreNeverTheGame(t *testing.T) {
	game := "My Game"
	for _, args := range [][]string{
		{"/home/u/.local/share/Steam/ubuntu12_32/steam", "-srt-logger-opened"},
		{"/home/u/.local/share/Steam/ubuntu12_32/steam", "-applaunch", "4000"},
		{"/home/u/.local/share/Steam/ubuntu12_32/steamwebhelper", "-lang=en_US"},
		{"/home/u/.local/share/Steam/ubuntu12_32/reaper", "SteamLaunch", "AppId=4000", "--", "/games/My Game/My Game.exe"},
		{"/usr/bin/python3", "/home/u/.local/share/Steam/steamapps/common/Proton 9.0/proton", "waitforexitandrun", "/games/My Game/My Game.exe"},
	} {
		if runs(args, game) {
			t.Errorf("%q counts as the game", args)
		}
	}
	if !runs([]string{`Z:\games\My Game\My Game.exe`}, game) {
		t.Error("the game's own process does not count as the game")
	}
}

// A game's process may be named "SH": the system's shell, named or resolved from a system folder, is never the game.
func TestASystemProgramIsNeverTheGame(t *testing.T) {
	for _, args := range [][]string{{"/bin/sh", "-c", "x"}, {"/usr/bin/SH"}, {"/usr/libexec/SH"}, {"/usr/sbin/SH"}} {
		if runs(args, "SH") || runs(args, "sh") {
			t.Errorf("%q counts as the game", args)
		}
	}
	if !runs([]string{"/home/u/games/Game/SH"}, "SH") {
		t.Error("the game's own SH does not count")
	}
	if !runs([]string{"/usr/bin/dotnet", "/home/u/Game/Loader.dll"}, "Loader") {
		t.Error("a game run by the system's dotnet does not count")
	}
	for path, want := range map[string]bool{"/usr/bin/sh": true, "/lib64/ld-linux.so": true, "/usr/local/bin/game": false, "/home/u/.local/share/Steam/x": false, `C:\Windows\System32\cmd.exe`: true, `D:\Games\SH.exe`: false, "": false} {
		if got := systemProgram(path); got != want {
			t.Errorf("systemProgram(%q) = %v", path, got)
		}
	}
}

func TestProcessesSkipsAProgramWhoseExecutableIsTheSystems(t *testing.T) {
	dir := t.TempDir()
	proc := filepath.Join(dir, "4242")
	if err := os.MkdirAll(proc, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proc, "cmdline"), []byte("sh\x00-c\x00true\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/usr/bin/sh", filepath.Join(proc, "exe")); err != nil {
		t.Fatal(err)
	}
	if got, err := Processes(dir, "sh"); err != nil || len(got) != 0 {
		t.Fatalf("the system shell was found as the game: %+v, %v", got, err)
	}

	// A game a distribution's Wine runs names its .exe though its executable is the system Wine's.
	wine := filepath.Join(dir, "4343")
	if err := os.MkdirAll(wine, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wine, "cmdline"), []byte(`Z:\games\Game\SH.exe`+"\x00"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/usr/bin/wine64-preloader", filepath.Join(wine, "exe")); err != nil {
		t.Fatal(err)
	}
	if got, err := Processes(dir, "SH"); err != nil || len(got) != 1 || got[0].PID != 4343 {
		t.Fatalf("a game run by the system's Wine was not found: %+v, %v", got, err)
	}
}

func TestHostTerminateReportsAProbeThatFailsAfterTERM(t *testing.T) {
	t.Setenv("FLATPAK_ID", "x")
	old := hostOutput
	t.Cleanup(func() { hostOutput = old })
	probes := 0
	hostOutput = func(name string, _ ...string) ([]byte, error) {
		if name != "sh" {
			return nil, nil
		}
		if probes++; probes == 1 {
			return []byte("alive\n"), nil
		}
		return nil, errors.New("flatpak-spawn: gone")
	}
	if err := Terminate(42, time.Second); err == nil {
		t.Fatal("a game that may still run was reported stopped")
	}
}

func TestParseHostProcessesSkipsMalformedRecords(t *testing.T) {
	t.Parallel()
	out := "7 100\nx\n8\n 9 y\n\x00" +
		"7/cmdline:/g/Game\x00junk\x00x/cmdline:Game\x00-1/cmdline:Game\x008/stat:Game\x009/cmdline\x00" +
		"10/cmdline:Game\x00"
	got := parseHostProcesses(out, "Game")
	if len(got) != 2 || got[0].PID != 7 || !got[0].Start.Equal(time.Unix(100, 0)) || got[1].PID != 10 ||
		!got[1].Start.IsZero() {
		t.Fatalf("%+v", got)
	}
}
