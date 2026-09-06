//go:build unix

package overlay

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// stopWithDescendants makes a cancelled run's child die together with
// everything it spawned, so cmd.Wait comes back when the work stops rather than
// when the last orphan happens to finish (S046-R1.3, R1.4).
//
// # The problem it solves, measured rather than assumed
//
// exec.CommandContext kills the DIRECT child on cancellation and nothing below
// it. pkgdev is a script: the process we spawn forks fetchers and helpers, and
// each of them inherits the pipe that cmd.Stdout is being copied from. Wait does
// not return while any descriptor for that pipe is open — the standard library
// says so in as many words ("If WaitDelay is zero, I/O pipes will be read until
// EOF, which might not occur until orphaned subprocesses of the command have
// also closed their descriptors") — so a cancelled target held the whole run
// open for the LIFETIME OF A GRANDCHILD. With a child that sleeps 30 seconds,
// Wait measured 30.002s after a cancel delivered at 300ms.
//
// That is not a slow interrupt, it is a broken one: the report is assembled
// before it is rendered (R1.3), so a run that cannot return cannot report, and
// R1.4's "render the report established up to that point" never happens.
//
// # Why the whole GROUP is killed, and not just bounded with WaitDelay
//
// cmd.WaitDelay is this repository's usual answer (internal/snapshot/runner.go,
// the three autoupdate fixers) and it does return promptly — but it returns by
// ABANDONING the descendant: the same measurement showed the grandchild still
// running afterwards. For a manifest run those descendants are network fetchers
// holding a distdir this process is about to delete, so "prompt" would be bought
// with orphaned downloads writing into a directory that no longer exists.
//
// WaitDelay also has a second edge that does not fit here: its timer starts when
// the child EXITS as well as when the context is done, so a lingering grandchild
// after a perfectly successful pkgdev would turn a regenerated Manifest into a
// reported failure. This story exists to stop reports claiming things that did
// not happen.
//
// Putting the child in its own process group and signalling the NEGATED pid
// signals every process in it. Nothing is left holding the pipe, so Wait returns
// because the work really ended. The same measurement: 301ms.
//
// # What it costs
//
//  1. The child is no longer in the terminal's foreground process group, so a
//     ctrl+c typed at a terminal no longer reaches pkgdev directly. It reaches
//     it through us — `overlay manifest` wires signal.NotifyContext for exactly
//     this and cancels the run context, which is what calls the function below.
//     The delivery path becomes one we control instead of two racing ones.
//  2. A descendant that calls setsid() leaves the group and is beyond this
//     signal. Such a process would already hang a run that was never cancelled,
//     so nothing here is made worse; it is simply not made better either.
//  3. SIGKILL, not SIGTERM, because that is the signal exec.CommandContext
//     already sends its direct child (Process.Kill). Sending something gentler
//     to the group would change what an interrupted pkgdev is asked to do, which
//     is a decision this sub-task has no reason to take.
func stopWithDescendants(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	cmd.Cancel = func() error {
		// Process is non-nil here: the standard library documents that Cancel is
		// not called if Start returned an error, and it is only armed after a
		// successful Start. Setpgid with a zero Pgid makes the child the leader
		// of a new group whose id IS its pid, which is what -pid then names.
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			if errors.Is(err, syscall.ESRCH) {
				// The group is already gone: the command finished in the instant
				// between the context being done and this call. os.ErrProcessDone
				// is the answer exec documents for that race — anything else
				// would make Wait report an error for a target that completed,
				// which is a failure invented by the reporting path.
				return os.ErrProcessDone
			}
			return err
		}
		return nil
	}
}
