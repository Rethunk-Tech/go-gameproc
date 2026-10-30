//go:build !windows

package gameproc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-safefs"
)

// Stop is Terminate, after asking a Wine process to close its windows first, as quitting it would: a program that
// registered with a running launcher (Steam) signs off only on a clean exit, and killed instead it stays listed under
// its Wine pid, which the launcher then waits on for good. Whatever is still running after grace is terminated. The
// polite close is skipped inside a Flatpak, where the process's environment is not readable.
func Stop(ctx context.Context, p Process, grace time.Duration) error {
	if p.PID > 0 && closeWine(ctx, p, grace) && exitedWithin(func() bool { return !Alive(p.PID) }, grace) {
		return nil
	}
	return Terminate(p.PID, grace)
}

// closeWine runs taskkill without /f in the process's own Wine environment, which posts WM_CLOSE to the windows of
// every process with that image name in its prefix. It reports whether taskkill ran.
func closeWine(ctx context.Context, p Process, timeout time.Duration) bool {
	wine := wineLoader(p.Exe)
	if wine == "" || len(p.Args) == 0 || flatpak.Inside() {
		return false
	}
	image := filepath.Base(strings.ReplaceAll(p.Args[0], `\`, "/"))
	if !strings.HasSuffix(strings.ToLower(image), ".exe") {
		return false
	}
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(p.PID), "environ"))
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, wine, "taskkill", "/im", image) //nolint:gosec // G204: wine is the prefix's wine; image is the game exe name
	// WINESERVERSOCKET names a descriptor the game inherited, which taskkill does not have.
	for kv := range strings.SplitSeq(strings.TrimRight(string(raw), "\x00"), "\x00") {
		if !strings.HasPrefix(kv, "WINESERVERSOCKET=") {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	return cmd.Run() == nil
}

// wineLoader is the wine launcher beside exe when exe is a Wine loader (Proton keeps both in one folder), or "".
func wineLoader(exe string) string {
	exe = strings.TrimSuffix(exe, " (deleted)")
	switch filepath.Base(exe) {
	case "wine", "wine64", "wine-preloader", "wine64-preloader":
	default:
		return ""
	}
	for _, name := range []string{"wine", "wine64"} {
		path := filepath.Join(filepath.Dir(exe), name)
		if safefs.IsFile(path) {
			return path
		}
	}
	return ""
}

// Terminate asks the process to exit (SIGTERM) and kills it if it is still there after grace. A process that is
// already gone is success; one the caller may not signal is an error.
func Terminate(pid int, grace time.Duration) error {
	if pid <= 0 {
		return errPID(pid)
	}
	if flatpak.Inside() {
		return hostTerminate(pid, grace)
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return err
	}
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}

// Alive reports whether a process with this pid exists; a pid of 0 or less never does. In a Flatpak a failure to ask
// the host reads as false; callers that must tell the two apart (Stop, WaitPID) use hostProbe.
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if flatpak.Inside() {
		alive, err := hostProbe(pid)
		return err == nil && alive
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// hostOutput runs a command on the host and returns its stdout. It is a variable so the package's tests can stand in
// for flatpak-spawn.
var hostOutput = func(name string, args ...string) ([]byte, error) {
	return flatpak.Command(context.Background(), name, args...).Output()
}

// Host PIDs mean nothing in the sandbox's PID namespace, so a Flatpak signals through the host's kill.
func hostKill(sig string, pid int) error {
	_, err := hostOutput("kill", "-"+sig, strconv.Itoa(pid))
	return err
}

// hostProbe reports whether the host process pid exists. kill's "no such process" is told from a failure to run
// flatpak-spawn at all by asking the host's sh to answer in its own stdout: any error here is the latter.
func hostProbe(pid int) (bool, error) {
	out, err := hostOutput("sh", "-c", `if kill -0 "$1" 2>/dev/null; then echo alive; else echo gone; fi`, "sh", strconv.Itoa(pid))
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "alive", nil
}

func hostTerminate(pid int, grace time.Duration) error {
	alive, err := hostProbe(pid)
	if err != nil || !alive {
		return err
	}
	if killErr := hostKill("TERM", pid); killErr != nil {
		if alive, err := hostProbe(pid); err != nil || !alive {
			return err
		}
	}
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if alive, err := hostProbe(pid); err != nil || !alive {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	killErr := hostKill("KILL", pid)
	if alive, err := hostProbe(pid); err != nil {
		return err
	} else if alive {
		return fmt.Errorf("process %d is still running after KILL: %w", pid, killErr)
	}
	return nil
}
