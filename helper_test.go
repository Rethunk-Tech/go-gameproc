package gameproc

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// helperEnv makes the test binary act as a child process instead of running tests, so tests start real processes
// without needing a program installed on the machine.
const helperEnv = "GAMEPROC_HELPER"

func TestMain(m *testing.M) {
	if mode := os.Getenv(helperEnv); mode != "" {
		helper(mode)
		return
	}
	os.Exit(m.Run())
}

// helper acts out one mode: sleep, exit=<n>, spawn (start a grandchild sleep, print its pid, stay) or term-ok
// (print ready, exit 0 on SIGTERM).
func helper(mode string) {
	switch {
	case mode == "sleep":
		time.Sleep(5 * time.Minute)
	case strings.HasPrefix(mode, "exit="):
		n, _ := strconv.Atoi(strings.TrimPrefix(mode, "exit="))
		os.Exit(n)
	case mode == "spawn":
		child := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^$") //nolint:gosec // G204: re-executes the test binary
		child.Env = append(os.Environ(), helperEnv+"=sleep")
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		// Reaped, or the exited grandchild stays a zombie that kill(pid, 0) still finds.
		go func() { _ = child.Wait() }()
		fmt.Println(child.Process.Pid)
		time.Sleep(5 * time.Minute)
	case mode == "term-ok":
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM)
		fmt.Println("ready")
		<-sig
	}
}

// helperCmd is the test binary set to run mode, not yet started.
func helperCmd(t *testing.T, mode string) *exec.Cmd {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^$") //nolint:gosec // G204: re-executes the test binary
	cmd.Env = append(os.Environ(), helperEnv+"="+mode)
	return cmd
}

// startHelper starts the helper with its first stdout line available and ends it with the test.
func startHelper(t *testing.T, cmd *exec.Cmd) *bufio.Reader {
	t.Helper()
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	return bufio.NewReader(out)
}

// reap waits for cmd in the background; Terminate and Alive see a process that is only a zombie as alive.
func reap(cmd *exec.Cmd) <-chan error {
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	return done
}

func readPID(t *testing.T, r *bufio.Reader) int {
	t.Helper()
	line, err := r.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatal(err)
	}
	return pid
}

// gone reports that pid no longer runs, counting a zombie nobody reaped as gone.
func gone(pid int) bool {
	if stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat"); err == nil {
		i := strings.LastIndex(string(stat), ")")
		return i >= 0 && strings.HasPrefix(strings.TrimSpace(string(stat)[i+1:]), "Z")
	}
	return !Alive(pid)
}

func waitGone(t *testing.T, pid int) {
	t.Helper()
	if !exitedWithin(func() bool { return gone(pid) }, 10*time.Second) {
		t.Fatalf("process %d is still running", pid)
	}
}

// await waits for a reaped helper; the error of one the test ended is not a failure.
func await(t *testing.T, done <-chan error) {
	t.Helper()
	if err := <-done; err != nil {
		t.Logf("helper ended: %v", err)
	}
}
