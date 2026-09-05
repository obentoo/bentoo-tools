package main

// Authored for story 047, sub-task 5.4 — S047-R1.2, S047-R8.2.
//
// The rule these inherit is story 046's R3.7: "IF the render mode is requested
// through the environment or the configuration with a value outside the
// accepted set, THEN THE SYSTEM SHALL render the report in plain and state the
// source, the value it refused and the mode it used instead." Story 047 does
// not restate it; S047-R1.2 asks that this payload reach the operator through
// the SAME renderer the other four kinds use, and the ambient-mode answer is
// part of what "the same" means.
//
// # These are green on arrival, and that is exactly why they exist
//
// `func presentCompareReport` in overlay_compare_report.go resolves the mode
// through `func reportModeOrPlain`, which warns and falls back, and nothing in
// runCompare resolves it earlier. So compare already behaves. Nothing about
// that fact is asserted anywhere: the fifth producer joined an envelope whose
// ambient-mode contract had a test file per producer, and it arrived without
// one. A guard that passes today is worth writing only when its ABSENCE would
// also be silent — that is this file's whole case, and the mutation table in
// 5.4's report is the evidence each case would bite if the behaviour went.
//
// # What this file asserts that sub-task 3.3 does not
//
// `func TestPresentCompareReport` in overlay_compare_report_test.go already
// proves an unusable --ui still renders. It calls presentCompareReport DIRECTLY,
// with a stub payload, and it cannot read the refusal sentence at all: `func
// Default` in internal/common/logger/logger.go fixes the logger's writer inside
// a sync.Once, so an in-process caller cannot redirect it and 3.3 says so.
//
// Everything below drives the REAL command through the root, over a real
// overlay, and reads the sentence back — because captureStream redirects file
// DESCRIPTOR 2 rather than the os.Stderr variable, so the singleton's writes
// land in the pipe. report_mode_fallback_test.go's package-level init() pins
// the logger to the real stderr before any capture has moved it, and
// requireLoggerOutputIsReadable (autoupdate_ambient_mode_test.go) is what makes
// that coverage OBSERVED here rather than assumed: without it, a missing
// sentence below would prove nothing about the code.
//
// # Which of the family's ten cases were ported, and which were not
//
// Ported, because each one bites on THIS command:
//   - FromTheEnvironmentKeepsTheRunAndStatesItself
//   - FromTheConfigurationKeepsTheRunAndStatesItself — the wrongly-split half by
//     SOURCE. A gate written against BENTOO_UI alone leaves ui.mode untouched,
//     and a config file survives the new shell an operator opens to test it.
//   - StatesTheRefusalExactlyOnce — R3.6. Compare has ONE speaker today; the
//     count is what notices the day a second one is added, which is precisely
//     what happened to `overlay autoupdate --check` and cost 12.3 a sub-task.
//   - LeavesTheRunsOwnFailureItsOwnStatusAndMessage — compare has a documented
//     non-zero condition of its own (S047-R7.2, D9: a --realign run that located
//     no ::gentoo tree), and it is reported AFTER the report is presented. That
//     ordering is the thing an early mode gate would destroy.
//
// NOT ported, with the reason:
//   - UnderJSONStatesItWithoutBreakingTheDocument (validate's). `overlay
//     compare` has no --json and no second renderer to fork into — its flags are
//     clone, cache-dir, no-cache, timeout, token, only-outdated, only-redundant,
//     only-patched, sync, concurrency, no-review, realign, depth and yes — so
//     the case has no compare equivalent. Its machine-readable path is --export,
//     which `func exportReport` writes from the payload and which no diagnostic
//     shares a stream with.
//   - DoesNotSoftenAnExplicitFlag (both files'). The explicit refusal is
//     root.go's PersistentPreRunE, one implementation for all 30 commands,
//     already pinned by autoupdate_ambient_mode_test.go, by
//     validate_ambient_mode_test.go and by root_rejection_once_test.go. In those
//     two files it is a COUNTERWEIGHT: each was written beside a fix whose
//     cheapest wrong form was "stop refusing the value anywhere", and that fix
//     would have turned their red cases green. No such fix is being made here —
//     compare already falls back — so a fourth copy would assert the root rather
//     than compare, which is the "asserts nothing about compare" shape 5.4 was
//     told to avoid. It stays covered where it is implemented.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedCompareFixture gives the harness a comparison it can actually run, with
// no network and no ::gentoo tree on the developer's machine.
//
// It is `func realignSetup` in overlay_compare_test.go rebuilt on top of
// `func newTestCLI` rather than beside it. realignSetup owns its own HOME and
// feeds `func realignRun`, which calls runCompare directly; everything here goes
// through the root command, because the two facts this file needs — an ambient
// value read from the environment, and a --ui refused before any run function —
// are both answered above runCompare and would be skipped by calling it.
//
// The ebuild bodies and the writer are realignSetup's, reused rather than
// respelled: they are one package ::gentoo also carries in a divergent form and
// one it does not, which is the smallest overlay that produces a report with
// both a compared row and a Bentoo-only one.
//
// carryBaseline=false leaves the ::gentoo tree without profiles/repo_name, which
// is how the review fails to locate it — compare's one non-zero condition, and
// the fixture the last case here needs.
func seedCompareFixture(t *testing.T, c *testCLI, carryBaseline bool) {
	t.Helper()

	gentoo := filepath.Join(c.Home(), "gentoo")
	realignWriteEbuild(t, c.Overlay(), "media-libs", "gst-plugins-qt6", "1.29.2", realignOursEbuild)
	realignWriteEbuild(t, c.Overlay(), "app-editors", "zed", "1.0.0", realignZedEbuild)
	realignWriteEbuild(t, gentoo, "media-libs", "gst-plugins-qt6", "1.29.2", realignBaselineEbuild)

	if carryBaseline {
		profiles := filepath.Join(gentoo, "profiles")
		if err := os.MkdirAll(profiles, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", profiles, err)
		}
		if err := os.WriteFile(filepath.Join(profiles, "repo_name"), []byte("gentoo\n"), 0o600); err != nil {
			t.Fatalf("write repo_name: %v", err)
		}
	}

	// `provider: local` is what keeps the run off the network. It is appended to
	// the config newTestCLI already wrote rather than replacing it, because the
	// overlay path in that file is what makes the run a run.
	appendHarnessConfig(t, c, "repositories:\n  gentoo:\n    provider: local\n    path: "+gentoo+"\n")

	// No `claude` on PATH. --no-review already makes every reviewer nil, so this
	// is belt and braces against a machine that has the CLI installed: the state
	// under test must be the same on every developer's box.
	t.Setenv("PATH", t.TempDir())
}

// appendHarnessConfig appends text to the config newTestCLI wrote for c.
//
// `func writeConfiguredUIMode` in validate_ambient_mode_test.go does the same
// thing for one key and is reused below for exactly that key; this is the
// general form, needed because the repositories block is a nested map rather
// than a single value.
func appendHarnessConfig(t *testing.T, c *testCLI, text string) {
	t.Helper()

	configPath := filepath.Join(c.Home(), ".config", "bentoo", "config.yaml")
	existing, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading the harness config at %s: %v", configPath, err)
	}
	if err := os.WriteFile(configPath, append(existing, []byte(text)...), 0o644); err != nil {
		t.Fatalf("appending to the harness config: %v", err)
	}
}

// runAmbientCompare runs `overlay compare --no-review` over c, plus any extra
// arguments.
//
// --no-review contacts no model, which is what makes the run deterministic and
// offline. --ui is deliberately absent: the flag is the OTHER source, answered
// at the root before any run function, and passing it would settle the question
// this file asks before the run started.
func runAmbientCompare(t *testing.T, c *testCLI, extra ...string) (stdout, stderr string, code int) {
	t.Helper()

	args := append([]string{"overlay", "compare", "--no-review"}, extra...)
	stdout, stderr, code = c.Run(args...)
	if strings.Contains(stderr, "unknown flag") {
		t.Fatalf("a flag was rejected: %s", stderr)
	}
	return stdout, stderr, code
}

// compareBodyMarker is a string only `func (r CompareRun) Sections` in
// internal/common/report/compare_run.go produces. It is
// `var comparePresentMarkers`' first entry, and it is here for the same reason:
// finding it proves the run reached the REPORT rather than some earlier exit.
const compareBodyMarker = "scanned in the Bentoo overlay"

// compareReportBody is stdout from the end of the live progress region onward,
// or "" when no report was printed.
//
// The region above it redraws in place with carriage returns, and which fraction
// lands between two writes is a property of how fast the workers finished rather
// than of the render — the same trap `func checkReportBody` documents for the
// check producer, and `func realignRun` strips it the same way. Below the last
// carriage return the two runs are byte-identical.
func compareReportBody(stdout string) string {
	if i := strings.LastIndex(stdout, "\r"); i >= 0 {
		stdout = stdout[i+1:]
	}
	if !strings.Contains(stdout, compareBodyMarker) {
		return ""
	}
	return stdout
}

// requireCompareStatesTheRefusal is R3.7's three facts plus R3.6's count, over
// one stream.
//
// They are checked separately so a failure names WHICH fact is missing. They
// fail differently in practice: parseMode's message already carries the source
// and the value, so a change that merely lowered it to Debug would take all
// three at once, while a second speaker added above the presenter would take
// only the count.
func requireCompareStatesTheRefusal(t *testing.T, stderr, source string) {
	t.Helper()

	if !strings.Contains(stderr, source) {
		t.Errorf("stderr does not name %s as the SOURCE of the refused value (R3.7).\n"+
			"    An operator told only that \"a UI mode\" was refused has three places to search:\n"+
			"    the flag they did not pass, their shell profile, and their config file.\n"+
			"    Observed stderr: %q", source, stderr)
	}
	if !strings.Contains(stderr, "bogus") {
		t.Errorf("stderr does not name the VALUE it refused (R3.7).\n"+
			"    Remedy: check that `func presentCompareReport` in overlay_compare_report.go\n"+
			"    still calls reportModeOrPlain before it renders. That call is what states the\n"+
			"    refusal, and this producer keeps no other use for the mode it gets back, so it\n"+
			"    is easy to delete as dead and lose the sentence with it.\n"+
			"    Observed stderr: %q", stderr)
	} else if got := strings.Count(stderr, "is not a UI mode"); got != 1 {
		t.Errorf("the refusal is stated %d time(s); R3.6 requires exactly once.\n"+
			"    One typo answered in two voices is the duplication 11.1 already paid for at the\n"+
			"    root, and this run gains a second speaker the moment a mode is resolved anywhere\n"+
			"    above `func presentCompareReport`.\n"+
			"    Observed stderr:\n%s", got, stderr)
	}
	if !statesTheFallback(stderr, "bogus") {
		t.Errorf("stderr does not say which mode the run used INSTEAD (R3.7).\n"+
			"    This is the fact no existing string carries: parseMode's message lists plain\n"+
			"    among the values it WOULD have accepted, which is not the same as telling the\n"+
			"    reader that the report above was rendered in it.\n"+
			"    Observed stderr: %q", stderr)
	}
}

// TestCompareAmbientModeFromTheEnvironmentKeepsTheRunAndStatesItself is R3.7
// whole, on the fifth producer.
//
// # One harness, two runs, one variable changed
//
// The control is not decoration. "Renders the report in plain" and "the run
// survives" are both claims about SAMENESS: what has to hold is that the run
// with the unusable value produced the same status and the same report as the
// run without it, and that the only difference is a sentence gained. Both runs
// go over the same home, the same config and the same overlay, so the
// environment is literally the only thing that differs between them — which also
// keeps the two temporary paths out of a byte comparison of the report.
//
// Compare is the producer where losing the run costs most. It is the most
// expensive report of the five to produce — it reads a remote tree and diffs
// ebuilds to get there — so a typo in a shell profile turning into "your
// comparison did not run" is the failure `func presentCompareReport`'s own doc
// says it resolves the mode late to avoid.
func TestCompareAmbientModeFromTheEnvironmentKeepsTheRunAndStatesItself(t *testing.T) {
	requireLoggerOutputIsReadable(t)

	c := newTestCLI(t)
	seedCompareFixture(t, c, true)

	controlOut, controlErr, controlCode := runAmbientCompare(t, c)
	if compareReportBody(controlOut) == "" {
		t.Fatalf("the control run produced no report — the fixture is not driving the producer this test is about.\n    exit %d, stderr %q\n%s", controlCode, controlErr, controlOut)
	}

	t.Setenv("BENTOO_UI", "bogus")
	stdout, stderr, code := runAmbientCompare(t, c)

	if code != controlCode {
		t.Errorf("`overlay compare` exited %d with BENTOO_UI=bogus and %d without it — R3.7 degrades the RENDER, it does not fail the run (the status is what a CI job branches on).\n"+
			"    Remedy: resolve the mode where the report is PRESENTED, not at the top of the\n"+
			"    run. overlay_autoupdate.go:617 is the gate this warns against; it cost the third\n"+
			"    producer both its report and its exit status.\n"+
			"    Observed stderr: %q", code, controlCode, stderr)
	}
	if compareReportBody(stdout) == "" {
		t.Fatalf("BENTOO_UI=bogus cost the operator the comparison entirely (R3.7): a value inherited from a shell profile decided whether the most expensive report of the five was printed at all.\n    Observed stdout: %q", stdout)
	}
	if compareReportBody(stdout) != compareReportBody(controlOut) {
		t.Errorf("the report differs from the one the same run produces with no BENTOO_UI set — R3.7 asks for the report in plain, not for a different report.\n--- with BENTOO_UI=bogus ---\n%s\n--- control ---\n%s", compareReportBody(stdout), compareReportBody(controlOut))
	}

	requireCompareStatesTheRefusal(t, stderr, "BENTOO_UI")
}

// TestCompareAmbientModeFromTheConfigurationKeepsTheRunAndStatesItself is the
// wrongly-split half by SOURCE.
//
// R3.7 names two ambient sources — "the environment or the configuration" — and
// every measurement that has ever motivated a sub-task in this family used only
// the first. A gate written against that measurement satisfies the case above
// while `ui.mode` stays exactly as it was, and the configured source is the one
// that survives a new shell, a cron entry and a reboot: an operator can unset a
// variable to find out whether it was the problem, and cannot unset a file they
// have not thought to suspect.
//
// The two sources must come out of the resolution together. What must NOT come
// out together is WHICH one is named, because "which of my three places is
// wrong" is the only question the operator has left — which is why the source is
// asserted by name rather than by "some source was mentioned".
func TestCompareAmbientModeFromTheConfigurationKeepsTheRunAndStatesItself(t *testing.T) {
	requireLoggerOutputIsReadable(t)

	c := newTestCLI(t)
	seedCompareFixture(t, c, true)

	controlOut, controlErr, controlCode := runAmbientCompare(t, c)
	if compareReportBody(controlOut) == "" {
		t.Fatalf("the control run produced no report — the fixture is not driving the producer this test is about.\n    exit %d, stderr %q\n%s", controlCode, controlErr, controlOut)
	}

	writeConfiguredUIMode(t, c, "bogus")
	stdout, stderr, code := runAmbientCompare(t, c)

	if code != controlCode {
		t.Errorf("`overlay compare` exited %d with an unusable ui.mode and %d without it — a typo in a config file degrades the render, it does not change the run's answer (R3.7).\n    Observed stderr: %q", code, controlCode, stderr)
	}
	if compareReportBody(stdout) == "" {
		t.Fatalf("an unusable ui.mode cost the operator the comparison entirely (R3.7).\n    Observed stdout: %q", stdout)
	}
	if compareReportBody(stdout) != compareReportBody(controlOut) {
		t.Errorf("the report differs from the one the same run produces with no ui.mode set (R3.7).\n--- with ui.mode: bogus ---\n%s\n--- control ---\n%s", compareReportBody(stdout), compareReportBody(controlOut))
	}

	requireCompareStatesTheRefusal(t, stderr, "ui.mode")
}

// TestCompareAmbientModeStatesTheRefusalExactlyOnce is R3.6 over the speakers
// this command has, and it is a third element rather than either half of R3.7.
//
// R3.7's own text is satisfied by a sentence and says nothing about how many.
// Compare has exactly ONE speaker today — `func presentCompareReport` is the
// only place in this command's path that resolves a mode — and that is the
// property worth pinning, not celebrating: `overlay autoupdate --check` had one
// too, until a gate resolving the whole precedence chain was added at the top of
// its run for a requirement that covered the FLAG alone. The result answered one
// typo in two voices, and 12.3 was the sub-task that paid for it.
//
// The count only means something about a run that SURVIVED — today the sentence
// appears once from a gate that then kills the run, on the third producer — so
// survival is asserted first and fatally.
func TestCompareAmbientModeStatesTheRefusalExactlyOnce(t *testing.T) {
	requireLoggerOutputIsReadable(t)

	c := newTestCLI(t)
	seedCompareFixture(t, c, true)

	t.Setenv("BENTOO_UI", "bogus")
	stdout, stderr, code := runAmbientCompare(t, c)

	if code != 0 || compareReportBody(stdout) == "" {
		t.Fatalf("the run did not survive its refused BENTOO_UI, so how many times the refusal was stated says nothing yet (exit %d).\n"+
			"    See TestCompareAmbientModeFromTheEnvironmentKeepsTheRunAndStatesItself for the\n"+
			"    survival half; this test measures the count on a run that finished.\n"+
			"    Observed stderr: %q", code, stderr)
	}

	const reason = "is not a UI mode"
	if got := strings.Count(stderr, reason); got != 1 {
		t.Errorf("the refusal is stated %d time(s); R3.6 requires exactly once.\n"+
			"    Remedy: exactly one place in this command may resolve the mode, and today that\n"+
			"    is `func presentCompareReport` in overlay_compare_report.go. A second resolution\n"+
			"    added above it — a preflight beside the --concurrency and --depth checks, say —\n"+
			"    would answer one typo in two voices.\n"+
			"    Observed stderr:\n%s", got, stderr)
	}
}

// TestCompareAmbientModeLeavesTheRunsOwnFailureItsOwnStatusAndMessage is the
// converse of the survival half: R3.7 makes a refused ambient mode stop COSTING
// the run, not stop the run from failing on its own account.
//
// The fixture is compare's ONE non-zero condition (S047-D9): a `--realign` run
// that located no ::gentoo tree examined nothing, which is a different sentence
// from having looked and found nothing. `func exitOnSkippedBaseline` in
// overlay_compare_realign.go answers it with exit 1, and S047-R7.2 fixes WHERE:
// after the render and after the export, so the operator keeps the comparison
// they waited on.
//
// That ordering is what an early mode gate destroys, and it destroys it
// invisibly — the exit status is 1 either way. So the status alone would not do:
// the report is required beside it, and required to be the SAME report, because
// "exit 1 with the comparison" and "exit 1 with a sentence about a shell
// variable" are the two outcomes this case exists to keep apart.
func TestCompareAmbientModeLeavesTheRunsOwnFailureItsOwnStatusAndMessage(t *testing.T) {
	requireLoggerOutputIsReadable(t)

	c := newTestCLI(t)
	seedCompareFixture(t, c, false) // no profiles/repo_name: the review refuses the tree

	controlOut, controlErr, controlCode := runAmbientCompare(t, c, "--realign")
	if controlCode != 1 {
		t.Fatalf("the control run exited %d, want 1 — this fixture exists to produce a failure that has nothing to do with rendering, and it is not producing one.\n    stderr %q\n%s", controlCode, controlErr, controlOut)
	}
	if compareReportBody(controlOut) == "" {
		t.Fatalf("the control run exited 1 without presenting its report, so there is nothing here to preserve (S047-R7.2).\n    stderr %q\n%s", controlErr, controlOut)
	}
	if !strings.Contains(controlOut, "::gentoo tree") {
		t.Fatalf("the control run does not say why it refused the review, so this case cannot tell a preserved diagnostic from a replaced one.\n%s", controlOut)
	}

	t.Setenv("BENTOO_UI", "bogus")
	stdout, stderr, code := runAmbientCompare(t, c, "--realign")

	if code != controlCode {
		t.Errorf("the run exited %d with BENTOO_UI=bogus and %d without it, on a failure that has nothing to do with rendering — a refused display value must not change what a broken run reports (R3.7).\n    Observed stderr: %q", code, controlCode, stderr)
	}
	if compareReportBody(stdout) == "" {
		t.Fatalf("BENTOO_UI=bogus cost the operator the report on a run that had already done all of its work (S047-R7.2): the exit pre-empted the render.\n    Observed stdout: %q", stdout)
	}
	if compareReportBody(stdout) != compareReportBody(controlOut) {
		t.Errorf("BENTOO_UI=bogus replaced the run's own diagnostic with one about the display — the operator is told about a shell variable that could not have caused their failure, and learns the real one only on the run after they fix the typo.\n--- with BENTOO_UI=bogus ---\n%s\n--- control ---\n%s", compareReportBody(stdout), compareReportBody(controlOut))
	}
}
