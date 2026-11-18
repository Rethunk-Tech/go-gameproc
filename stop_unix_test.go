//go:build linux

package gameproc

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
)

func TestStopRefusesAPidNoProcessHas(t *testing.T) {
	if err := Stop(t.Context(), Process{PID: 0}, time.Millisecond); err == nil {
		t.Fatal("pid 0 was stopped")
	}
}

// taskkill /im closes every process of that image in the prefix, so Stop ends p by pid instead when another one runs.
func TestStopDoesNotCloseWineWindowsWhenAnotherProcessSharesTheImage(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("WINEPREFIX", filepath.Join(dir, "prefix"))
	closeLog := filepath.Join(dir, "close")
	wine := "#!/bin/sh\necho ran >\"" + closeLog + "\"\n"
	if err := safefs.WriteFile(filepath.Join(dir, "wine"), []byte(wine), 0o700); err != nil {
		t.Fatal(err)
	}
	pid1, done1 := startedAs(t, "Game.exe")
	pid2, _ := startedAs(t, "Game.exe")
	p := Process{PID: pid1, Exe: filepath.Join(dir, "wine64-preloader"), Args: []string{`C:\Games\Game.exe`}}
	if err := Stop(t.Context(), p, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	await(t, done1)
	if _, err := os.Stat(closeLog); err == nil {
		t.Fatal("taskkill ran; it would close the other process of the same image")
	}
	if !Alive(pid2) {
		t.Fatal("the other process of the same image was ended")
	}
}

func TestStopEndsAWineProcessByPidWhenAPoliteCloseCannotRun(t *testing.T) {
	dir := t.TempDir()
	if err := safefs.WriteFile(filepath.Join(dir, "wine"), []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "wine64-preloader")

	cmd := helperCmd(t, "sleep")
	startHelper(t, cmd)
	done := reap(cmd)
	if err := Stop(t.Context(), Process{PID: cmd.Process.Pid, Exe: exe, Args: []string{"Game"}}, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	await(t, done)

	if err := Stop(t.Context(), Process{PID: 1 << 30, Exe: exe, Args: []string{"Game.exe"}}, time.Millisecond); err != nil {
		t.Fatal(err)
	}
}

func TestWineLoaderIsTheWineBinaryBesideALoader(t *testing.T) {
	dir := t.TempDir()
	if wineLoader(filepath.Join(dir, "wine64-preloader")) != "" {
		t.Fatal("a loader folder with no wine binary was treated as a prefix")
	}
	wine64 := filepath.Join(dir, "wine64")
	if err := safefs.WriteFile(wine64, []byte{}, 0o700); err != nil {
		t.Fatal(err)
	}
	got := wineLoader(filepath.Join(dir, "wine-preloader") + " (deleted)")
	if got != wine64 {
		t.Fatalf("wineLoader = %q, want the wine64 beside the deleted loader", got)
	}
}

func TestHostTerminateKillsAProcessStillAliveAfterGrace(t *testing.T) {
	t.Setenv("FLATPAK_ID", "x")
	old := hostOutput
	t.Cleanup(func() { hostOutput = old })
	var sigs []string
	hostOutput = func(_ string, args ...string) ([]byte, error) {
		if strings.Contains(args[1], `kill -"$3"`) {
			sigs = append(sigs, args[len(args)-1])
			return nil, nil
		}
		if len(sigs) > 0 && sigs[len(sigs)-1] == "KILL" {
			return []byte("gone\n"), nil
		}
		return []byte("alive\n"), nil
	}
	if err := Terminate(42, 150*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sigs, []string{"TERM", "KILL"}) {
		t.Fatalf("signals %q, want TERM then KILL", sigs)
	}
}
