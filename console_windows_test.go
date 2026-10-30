package gameproc

import (
	"os"
	"testing"
	"unicode/utf16"
)

func TestStartOwnConsoleReportsTheExitCode(t *testing.T) {
	env := append(os.Environ(), helperEnv+"=exit=7")
	done, err := StartOwnConsole(env, "", os.Args[0], []string{"-test.run=^$"})
	if err != nil {
		t.Fatal(err)
	}
	werr := <-done
	if got := StatusOf(werr); got != (Status{Code: 7}) {
		t.Fatalf("exit %+v from %v", got, werr)
	}
}

func TestStartOwnConsoleWithNoEnvGivesTheCallersOwn(t *testing.T) {
	t.Setenv(helperEnv, "exit=7")
	done, err := StartOwnConsole(nil, "", os.Args[0], []string{"-test.run=^$"})
	if err != nil {
		t.Fatal(err)
	}
	if got := StatusOf(<-done); got != (Status{Code: 7}) {
		t.Fatalf("exit %+v: the child did not get the caller's environment", got)
	}
}

func TestEnvBlockSortsAndKeepsTheLastDuplicate(t *testing.T) {
	out := envBlock([]string{"b=1", "A=1", "B=2"})
	if got := string(utf16.Decode(out)); got != "A=1\x00B=2\x00\x00" {
		t.Fatalf("block %q", got)
	}
}
