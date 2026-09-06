//go:build !unix

package overlay

import (
	"os/exec"
	"time"
)

// manifestWaitDelay bounds how long cmd.Wait may block draining I/O after a
// cancelled run's child has been killed, on the platforms that cannot kill a
// process group.
//
// It is generous on purpose. WaitDelay's timer also starts when the child exits
// NORMALLY, so a short delay would turn a successful pkgdev whose helper lingers
// a moment into a target reported as failed — a failure invented by the
// reporting path, which is the class of defect story 046 exists to remove. Ten
// seconds is long enough that only a genuinely stuck descendant reaches it, and
// short enough that an interrupt still returns while the operator is watching.
const manifestWaitDelay = 10 * time.Second

// stopWithDescendants is the non-Unix substitute, and it is deliberately weaker
// than the Unix one — see manifest_cancel_unix.go for what is being worked
// around and why the group kill is the answer there.
//
// syscall.SysProcAttr has no Setpgid field outside Unix, so there is no group to
// signal and no portable way to reach a grandchild at all. WaitDelay is what
// remains: it does not kill the descendant, it stops WAITING for it, closing the
// pipes so Wait returns. The orphan keeps running.
//
// That trade is accepted here rather than hidden, for one reason: pkgdev is a
// Gentoo tool (dev-util/pkgdev) and this command's own discovery step refuses the
// run without it, so this file is compiled for completeness rather than for a
// host that regenerates Manifests. A prompt, honest report with a leaked helper
// beats a run that cannot return at all.
func stopWithDescendants(cmd *exec.Cmd) {
	cmd.WaitDelay = manifestWaitDelay
}
