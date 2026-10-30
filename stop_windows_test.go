//go:build windows

package gameproc

import (
	"math"
	"testing"
	"time"
)

func TestPIDUint32RejectsWhatNoProcessHas(t *testing.T) {
	t.Parallel()
	for _, pid := range []int{-1, 0} {
		if _, err := pidUint32(pid); err == nil {
			t.Fatalf("pid %d accepted", pid)
		}
	}
}

func TestWaitMillisClampsLargeDuration(t *testing.T) {
	t.Parallel()
	d := time.Duration(int64(math.MaxUint32)+1) * time.Millisecond
	if got := waitMillis(d); got != math.MaxUint32 {
		t.Fatalf("waitMillis clamp: got %d", got)
	}
}
