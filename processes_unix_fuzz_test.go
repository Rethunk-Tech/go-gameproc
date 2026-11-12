//go:build !windows

package gameproc

import (
	"path/filepath"
	"strings"
	"testing"
)

func FuzzParseHostProcesses(f *testing.F) {
	f.Add("42 1700000000\n\x0042/cmdline:Z:\\games\\MyGame.exe\x00", "MyGame")
	f.Add("7 100\n\x007/cmdline:dotnet\x007/cmdline:Game.dll\x00", "Game")
	f.Add("1 0\n\x001/cmdline:/opt/game/game\x00", "game")
	f.Add("100 1\n200 2\n\x00100/cmdline:/opt/game/game\x00200/cmdline:/opt/game/tool\x00", "game")
	f.Add("8 0\n\x008/cmdline:mono\x008/cmdline:/mods/Helper.exe\x00", "Helper")
	f.Add("", "game")
	f.Add("not-a-pid 1\n\x00junk/cmdline:x\x00", "game")
	f.Fuzz(func(t *testing.T, out, name string) {
		procs := parseHostProcesses(out, name)
		for i, p := range procs {
			if p.PID < 0 {
				t.Fatalf("procs[%d] pid %d", i, p.PID)
			}
			if !hostArgsMatch(p.Args, name) {
				t.Fatalf("procs[%d] pid %d args %q do not match %q", i, p.PID, p.Args, name)
			}
		}
		again := parseHostProcesses(out, name)
		if !sameHostProcs(procs, again) {
			t.Fatalf("parse not deterministic")
		}
		wider := parseHostProcesses(out, name, name+"\xff")
		if len(wider) < len(procs) {
			t.Fatalf("adding a name dropped %d -> %d matches", len(procs), len(wider))
		}
	})
}

// hostArgsMatch is the name rule in runs, minus the system-folder skip, which can
// only drop a match. Every process parseHostProcesses returns must still satisfy it.
func hostArgsMatch(args []string, name string) bool {
	match := func(a string) bool {
		base := filepath.Base(strings.ReplaceAll(a, `\`, "/"))
		if ext := strings.ToLower(filepath.Ext(base)); ext == ".exe" || ext == ".dll" {
			return strings.EqualFold(strings.TrimSuffix(base, filepath.Ext(base)), name)
		}
		return base == name
	}
	if len(args) == 0 {
		return false
	}
	if match(args[0]) {
		return true
	}
	host := filepath.Base(args[0])
	return (host == "dotnet" || host == "mono") && len(args) > 1 && match(args[1])
}

func sameHostProcs(a, b []Process) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].PID != b[i].PID || !a[i].Start.Equal(b[i].Start) {
			return false
		}
		if len(a[i].Args) != len(b[i].Args) {
			return false
		}
		for j := range a[i].Args {
			if a[i].Args[j] != b[i].Args[j] {
				return false
			}
		}
	}
	return true
}
