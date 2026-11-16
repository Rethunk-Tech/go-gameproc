//go:build !windows

package gameproc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProcessesSkipsUsrLib(t *testing.T) {
	proc := t.TempDir()
	write := func(pid, cmdline string) {
		t.Helper()
		d := filepath.Join(proc, pid)
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "cmdline"), []byte(cmdline+"\x00"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("100", "/usr/lib/systemd/systemd")
	write("101", "/usr/lib64/firefox/firefox")
	write("102", "/usr/lib64/ld-linux-x86-64.so.2")
	for _, name := range []string{"systemd", "firefox", "ld-linux-x86-64.so.2"} {
		got, err := Processes(proc, name)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("Processes(%q) = %v", name, got)
		}
	}
}
