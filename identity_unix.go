//go:build !windows

package gameproc

import (
	"strconv"
	"sync"
	"time"
)

// startSlack absorbs the whole-second rounding of the boot time startTime adds the start tick to; a recycled pid
// starts a process far later than this.
const startSlack = 2 * time.Second

// sameProcess reports whether the process now holding p.PID is the one p was listed as. A Process with no Start
// (macOS, a listing that could not read it) cannot be told apart, so it is taken as the same.
func sameProcess(p Process) bool {
	if p.Start.IsZero() {
		return true
	}
	now := startTime("/proc", strconv.Itoa(p.PID))
	return !now.IsZero() && now.Sub(p.Start).Abs() <= startSlack
}

// treeLeaders remembers when each tracked process group leader started, so KillTree can tell the leader from a later
// process that was given its pid. The entries outlive UntrackTree on purpose: KillTree usually runs after the leader
// was reaped, which is when its pid is free to be reused. The oldest entry is dropped past maxLeaders.
var treeLeaders = struct {
	sync.Mutex
	start map[int]time.Time
	order []int
}{start: map[int]time.Time{}}

const maxLeaders = 256

func rememberLeader(pid int) {
	start := startTime("/proc", strconv.Itoa(pid))
	if start.IsZero() {
		return
	}
	l := &treeLeaders
	l.Lock()
	defer l.Unlock()
	if _, seen := l.start[pid]; !seen {
		l.order = append(l.order, pid)
		if len(l.order) > maxLeaders {
			delete(l.start, l.order[0])
			l.order = l.order[1:]
		}
	}
	l.start[pid] = start
}

// leaderReplaced reports whether a live process holds pid and is not the leader TrackTree saw. With the leader gone,
// the kernel keeps its number out of reuse while any group member lives, so a group signal then reaches only
// the leader's own orphans.
func leaderReplaced(pid int) bool {
	l := &treeLeaders
	l.Lock()
	want, ok := l.start[pid]
	l.Unlock()
	if !ok {
		return false
	}
	now := startTime("/proc", strconv.Itoa(pid))
	return !now.IsZero() && now.Sub(want).Abs() > startSlack
}
