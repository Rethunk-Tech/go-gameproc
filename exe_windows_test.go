package gameproc

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestProcessesFindsAnUpperCaseExe(t *testing.T) {
	src, err := os.ReadFile(os.Args[0]) //nolint:gosec // G703: the test binary itself
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "HELPER.EXE")
	//nolint:gosec // G703: dst is in the test's own temp folder
	if err := os.WriteFile(dst, src, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := helperCmd(t, "sleep")
	cmd.Path = dst
	startHelper(t, cmd)
	procs, err := Processes("", "helper")
	if err != nil || !slices.ContainsFunc(procs, func(p Process) bool { return p.PID == cmd.Process.Pid }) {
		t.Fatalf("%+v, %v", procs, err)
	}
}
