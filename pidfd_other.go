//go:build !linux && !windows

package gameproc

import "time"

// Off Linux there is no pidfd: signals go to the bare pid, after the start-time check where the platform lists one.
func pidfdTerminate(Process, time.Duration) (bool, error) { return false, nil }

func pidfdWait(int, time.Time) bool { return false }
