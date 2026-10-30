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

func TestEnvBlockSortsAndKeepsTheLastDuplicate(t *testing.T) {
	out := envUTF16([]string{"b=1", "A=1", "B=2"})
	if got := string(utf16.Decode(out)); got != "A=1\x00B=2\x00\x00" {
		t.Fatalf("block %q", got)
	}
}
