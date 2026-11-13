package gameproc

import (
	"path/filepath"
	"strings"
	"testing"
)

func FuzzWineHostPath(f *testing.F) {
	f.Add(`Z:\home\user\Games\MyGame.exe`)
	f.Add(`z:/opt/game/game.exe`)
	f.Add(`C:\Program Files\game.exe`)
	f.Add(`/opt/game/game`)
	f.Add(`Z:`)
	f.Add("")
	f.Add(`Z:foo\bar`)
	f.Fuzz(func(t *testing.T, a string) {
		got := wineHostPath(a)
		want := a
		if len(a) > 2 && (a[0] == 'Z' || a[0] == 'z') && a[1] == ':' {
			want = strings.ReplaceAll(a[2:], `\`, "/")
		}
		if got != want {
			t.Fatalf("wineHostPath(%q) = %q, want %q", a, got, want)
		}
	})
}

func FuzzExeIs(f *testing.F) {
	f.Add("/opt/games/MyGame.exe", "MyGame")
	f.Add(`C:\Games\GAME.EXE`, "game")
	f.Add("/proc/123/exe (deleted)", "exe")
	f.Add("/usr/bin/dotnet", "dotnet")
	f.Add("", "game")
	f.Add("/games/foo.dll", "foo.dll")
	f.Fuzz(func(t *testing.T, exe, name string) {
		got := (Process{Exe: exe}).ExeIs(name)
		base := trimExe(strings.TrimSuffix(filepath.Base(exe), " (deleted)"))
		want := exe != "" && strings.EqualFold(base, trimExe(name))
		if got != want {
			t.Fatalf("ExeIs(%q, %q) = %v, want %v", exe, name, got, want)
		}
	})
}
