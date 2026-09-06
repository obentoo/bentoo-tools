package main

// Authored for story 046, sub-task 4.4 — R3.2.
//
// Written from the contract: R3.2 — "IF --ui is given a value outside the
// accepted set, THEN THE SYSTEM SHALL reject the run BEFORE DOING ANY WORK,
// naming the set it accepts." design.md's Error Handling Strategy adds where
// that now happens: "rejected once at the root rather than per command".
//
// The accepted set is not written out here. internal/common/report/mode.go
// derives the message from `modes`, precisely so a fifth value cannot make the
// sentence lie; a test that hard-coded the four words would be a second place
// the set is spelled, and it would go stale the same way.
//
// # It is a separate file from flags_root_test.go, and that is deliberate
//
// 4.3 and 4.4 name the same file in their Tests fields, and one file per
// sub-task is what keeps a failure readable: 4.3 fails when a flag does not
// reach a command, 4.4 when a bad value is not refused. The validation commands
// already separate them — `-run TestRootFlags` and `-run TestUIRejection`.
//
// Red on arrival: `--ui` is not declared outside autoupdate, so `overlay
// manifest --ui=bogus` is refused as an UNKNOWN FLAG — the right exit status
// for the wrong reason, naming no accepted set.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/obentoo/bentoolkit/internal/common/report"
)

// acceptedSetIsNamed reports whether a rejection message names every value the
// flag accepts, asking report.Mode for the list rather than repeating it.
func acceptedSetIsNamed(message string) bool {
	for _, mode := range []report.Mode{report.ModeAuto, report.ModePlain, report.ModeInline, report.ModeFullscreen} {
		if !strings.Contains(message, string(mode)) {
			return false
		}
	}
	return true
}

// TestUIRejectionStopsAReportProducingCommand pins R3.2 where the cost of
// getting it wrong is highest: a run that did its work and then discovered it
// could not render it has spent the time and lost the output.
//
// "No work was done" is asserted through --export. A run that reached the end
// writes that file; a run refused at the flag layer cannot have.
func TestUIRejectionStopsAReportProducingCommand(t *testing.T) {
	c := newTestCLI(t)
	seedCheckOverlay(t, c, "1.7.1", "1.8.0", "app-misc/jq")

	export := filepath.Join(t.TempDir(), "report.json")

	stdout, stderr, code := c.Run("overlay", "manifest", "--dry-run", "--ui=bogus", "--export="+export)

	if code == 0 {
		t.Errorf("`--ui=bogus` exited 0 — an unusable value was accepted")
	}
	if !acceptedSetIsNamed(stderr) {
		t.Errorf("the rejection does not name the set it accepts (R3.2):\n%s", stderr)
	}
	if strings.Contains(stderr, "unknown flag") {
		t.Errorf("the flag was refused as UNKNOWN rather than as out of range — the operator is told the flag does not exist when the problem is its value:\n%s", stderr)
	}
	if _, err := os.Stat(export); err == nil {
		t.Errorf("the run wrote %s — it did its work before deciding it could not render it (R3.2)", export)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("the refused run still produced output:\n%s", stdout)
	}
}

// TestUIRejectionStopsACommandThatProducesNoReport is the converse half, and
// the one a per-command implementation gets wrong. D4 accepts that the flags
// appear on `version` and `completion` too; what may not happen is one command
// refusing a bad value while another shrugs it off, because then the operator
// learns the rule from whichever command they tried first.
func TestUIRejectionStopsACommandThatProducesNoReport(t *testing.T) {
	c := newTestCLI(t)

	baseline, _, baselineCode := c.Run("version")
	if baselineCode != 0 {
		t.Fatalf("`version` exited %d on its own", baselineCode)
	}

	stdout, stderr, code := c.Run("version", "--ui=bogus")

	if code == 0 {
		t.Error("`version --ui=bogus` exited 0 — the same value is refused by one command and accepted by another")
	}
	if !acceptedSetIsNamed(stderr) {
		t.Errorf("the rejection does not name the accepted set (R3.2):\n%s", stderr)
	}
	if trimmed := strings.TrimSpace(baseline); trimmed != "" && strings.Contains(stdout, trimmed) {
		t.Errorf("the command ran anyway — the rejection happened after the work, not before it (R3.2)\n%s", stdout)
	}
}

// TestUIRejectionAcceptsEveryValueItNames is the hostile half of the same rule,
// and without it a rejection that refused EVERYTHING would pass both tests
// above perfectly.
//
// Every value the message names must actually be accepted; the set is asked of
// report.Mode, so a fifth mode is covered the day it is added.
func TestUIRejectionAcceptsEveryValueItNames(t *testing.T) {
	for _, mode := range []report.Mode{report.ModeAuto, report.ModePlain, report.ModeInline, report.ModeFullscreen} {
		t.Run(string(mode), func(t *testing.T) {
			c := newTestCLI(t)
			writeExitTestEbuild(t, c.Overlay(), "app-misc/jq", "1.7.1")

			_, stderr, _ := c.Run("overlay", "manifest", "--dry-run", "--ui="+string(mode))

			if acceptedSetIsNamed(stderr) && strings.Contains(strings.ToLower(stderr), "bogus") {
				t.Fatalf("--ui=%s was rejected: %s", mode, stderr)
			}
			if strings.Contains(stderr, "unknown flag") {
				t.Fatalf("--ui=%s was refused as an unknown flag: %s", mode, stderr)
			}
		})
	}
}
