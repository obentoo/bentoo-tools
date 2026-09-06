package main

// Authored for story 046, sub-task 9.1 — R6.2.
//
// Written from the contract: R6.2 — "WHERE this story modifies a file that
// declares a fixed column width, THE SYSTEM SHALL replace that declaration with
// a measured width" — and Constraint 8, which says what happens to the rest:
// "the typed column widths in files this story does not touch. The count at
// hand-off is recorded in story 047 rather than described, so that it can be
// checked rather than believed."
//
// story.md measures the debt at c8e347e: 18 typed widths in 6 files. Eight of
// them are in files this story touches — six in overlay_autoupdate.go, two in
// overlay_validate.go — so this test fails until they are measured, and then
// reports the ten that remain as a number story 047 can check.
//
// CORRECTED, sub-task 13.4: the number handed on is EIGHT, not ten. Two
// measurements moved it after this paragraph was written — a nineteenth width
// story.md's count had missed, and the three in overlay_prune.go that sub-task
// 11.3 measured away — and both are recorded at widthDebtBaseline below, which
// now equals what the sweep prints instead of sitting ten steps above it.
//
// # Why the AST and not grep
//
// internal/common/report/render/width.go is on the touched list and contains
// the text "%-45s" in a comment explaining why it no longer types one. A grep
// would report the explanation as the defect, for the lifetime of the file. The
// sweep below reads STRING LITERALS out of the parsed source, where a comment
// is not a literal and cannot be one.
//
// Red on arrival: not by mutation and not by a missing symbol — the eight
// widths are in the tree right now.
//
// # Evidence that it still fails, taken after those eight were measured (R8.3)
//
// The line above is true of the day this file was written and stopped being
// checkable the moment task 9 finished: the eight widths are gone, this guard
// passes, and a green test proves nothing about what it would catch. R8.3 asks
// for the evidence a green guard cannot supply, and the story artifacts holding
// the original Red (.draft/red-evidence.yaml) are not committed — `git ls-files
// .epic/` returns nothing — so a pointer to that file is a pointer to nothing
// for anyone who cloned this. internal/common/report/citation_test.go makes
// that argument in full; this is the same correction, applied here.
//
// Measured on 2026-08-29, against sub-task 10.6. One mutation, applied on its
// own, run, and reverted immediately with `git checkout --`, the restore
// verified by `git status --porcelain` reporting nothing and by a second run of
// this test going green.
//
// Mutation: ONE of the six widths this story took out of overlay_autoupdate.go
// was put back where it had stood. The unclaimed-ebuild line is handed a key
// already padded to a column render.ColumnWidth measured; it was returned to
// typing that column into the format string, exactly as it read at c8e347e —
//
//	return fmt.Sprintf("%s %s", key, d.Disk)        // measured, today
//	return fmt.Sprintf("%-45s %s", d.Key, d.Disk)   // typed, as it was
//
// Observed, wrapped to this comment's width and otherwise unedited:
//
//	--- FAIL: TestWidthDebt (0.00s)
//	    width_debt_test.go:220: cmd/bentoo/overlay_autoupdate.go:1146: "%-45s %s"
//	    declares a fixed column width (R6.2).
//	            Remedy: measure it. render.ColumnWidth(values) gives the width the
//	            values actually need, in display cells, and render.Shorten bounds
//	            it — a width typed into a format string is wrong in both
//	            directions at once, too narrow the moment one value outgrows it
//	            and too wide on every run that never comes close.
//
// The mutation went inside a file on the touched list deliberately, and the
// same run shows what that buys: TestWidthDebtRemainderIsCounted PASSED while
// this test failed. The two guards divide the tree between them — this one owns
// the files the story edits, that one counts what is left outside them — so a
// width typed into a touched file has exactly one guard that can catch it, and
// this run is the demonstration that the one which can, does.
//
// The literal is quoted above inside a COMMENT, which costs nothing: the sweep
// reads string literals out of the parsed source, for the reason the section
// above gives, so this note cannot make its own file the defect.

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// repoRoot is where this package sits relative to the module root: `go test`
// runs each package in its own directory.
const repoRoot = "../.."

// typedWidth matches a COLUMN whose width was typed rather than measured: a
// left-aligned string field, %-45s or %-10.10s.
//
// It is exactly the pattern story.md counted at c8e347e — 18 in 6 files — so
// the number this test reports and the number the story recorded are the same
// measurement. Two things are deliberately outside it:
//
//   - %3d and %02d. A zero-padded counter or a progress percentage is a
//     NUMBER's format, not a column's width: nothing in a run's data can make
//     "[ 42%]" need more room, so there is nothing to measure.
//   - %-*s. The width comes from an argument, which is what a measured column
//     looks like once it reaches Printf. Flagging it would flag the fix.
var typedWidth = regexp.MustCompile(`%-\d+(\.\d+)?s`)

// widthDebtBaseline is the ceiling on the typed column widths left in files
// this story does NOT touch, and it is a MEASUREMENT rather than a target:
// SEVEN, over three files, which is what TestWidthDebtRemainderIsCounted below
// printed on 2026-09-01 against the tree exactly as it is committed. Nothing
// was picked to make the number fit — it is what the sweep prints, and
// TestWidthDebtBaselineEqualsTheMeasurement holds the two together so the
// published figure cannot drift above the thing it is a ceiling on.
//
// It may only FALL, and it is never raised. That trigger is what story 044
// lacked when it recorded three typed widths as accepted debt "to be repaid
// when one of them is next touched": none were touched, and the count had
// reached 18 by the time anyone looked. A debt figure edited upward to
// accommodate what was found records nothing, so the prohibition is not left as
// a sentence here — widthDebtHighWaterMark holds the highest value this pin has
// ever taken and TestWidthDebtBaselineNeverRises refuses a run that exceeds it.
// The shape is internal/common/report/citation_test.go's, where
// unattributedCitationDebt is pinned to its own measurement for the same reason
// and says so in the same words.
const widthDebtBaseline = 7

// # How this number reached 8, and why every fall was allowed
//
// 18, story.md's count at c8e347e: 18 typed widths in 6 files. That is the
// figure this story was handed and the highest the pin has ever stood, which is
// what widthDebtHighWaterMark now records.
//
// MEASURED, 2026-08-27: this sweep finds NINETEEN, not eighteen — eight in the
// two touched files and eleven outside them. The nineteenth is in
// misc/design/design-system/component/catalogue.go, which story.md's own count
// missed because it was taken over cmd/ and internal/ only. The baseline stayed
// at 18 rather than being quietly raised to 19: a debt figure edited to match
// what was found is not a check.
//
// MEASURED AGAIN, 2026-08-30, after sub-task 11.3 pulled overlay_prune.go onto
// the list below and measured the three widths it carried: eleven outside
// becomes EIGHT. The three did not move from one side of the hand-off to the
// other, they stopped existing — which is the only way this number is allowed
// to fall.
//
// LOWERED to 8, sub-task 13.4. The pin had stayed at 18 while the quantity it
// bounds was 8, because the two count different sets: 18 included the eight
// widths that were then INSIDE the touched files and have since been measured
// away. Ten steps of headroom is not a ceiling — ten further typed widths could
// have landed in untouched files with this suite green and still publishing 8
// as the debt, which is the same silence story 044 was left holding. Lowering
// the pin onto its measurement is the one move the never-raise rule above
// always permits; it is that rule applied, not an exception to it.
//
// LOWERED to 7, sub-task 14.2, 2026-09-01, and this fall is a DIFFERENT KIND
// from the two above it. Nothing was repaid: the eighth unit was never a live
// typed width. It was a sentence — misc/design/design-system/component/
// catalogue.go's `Why:` string, asserting in the present tense that
// overlay_prune.go "reaches for" a width sub-task 11.3 had already removed.
// Because the claim is a raw STRING LITERAL rather than a comment,
// typedWidthsIn's *ast.BasicLit walk counted it as debt, and neither
// design-system file was in the story's diff, so nothing failed and nobody
// noticed. Constraint 8 promises story 047 "a number that can be checked rather
// than believed", and that unit is the one that failed checking: seven of the
// eight are live (lintfix 1, sweep 5, compare_depth 1) and the eighth was a
// description of a width that no longer exists.
//
// Both design-system files are on the subject list above as of this sub-task,
// which is what the list's own rule asks of an executor that edits them. The
// discount that adds is ZERO — the rewrite left neither file with a typed width
// — so the fall to 7 is the literal's removal and nothing else.

// filesThisStoryTouches is R6.2's subject: every file this story MODIFIES.
//
// It used to say "every file named in the task list's Context blocks", and that
// is a strictly narrower set. R6.2 says "modifies"; D9 says "a file this story
// provably edits", which is the diff. The two readings came apart in the one
// place where a drift leaves no trace: commit 55e6d06 edited
// cmd/bentoo/overlay_prune.go for the constructor extraction, no Context block
// named it, and the three typed widths in it sat inside the requirement and
// outside the guard that enforces the requirement.
//
// It is still written out rather than derived from `git diff`, because the rule
// has to be checkable in a working tree with no branch point to diff against —
// and because a list somebody has to add to when they touch a further file is a
// list that states its own scope. An executor adding a file to this story adds
// it here.
//
// What is new is that the list is now CHECKED against the diff wherever a base
// commit resolves — widthDebtSubjectGapsAgainst below, driven by
// TestSubjectListAccountsForTheDiff. Written out AND checked: the offline
// property survives, and the list stops being a claim about the diff that
// nothing compares to the diff. That is how the gap above was able to open.
//
// BOTH DIRECTIONS are checked, and it took both. Until sub-task 13.4 only one
// was: every diff-touched file carrying a typed width had to appear here, which
// a list naming the whole repository satisfies. The converse — nothing here
// that the diff does not support — is subjectEntriesOutsideTheDiff, driven by
// TestSubjectListNamesNothingOutsideTheDiff, and it is not tidiness:
// TestWidthDebtRemainderIsCounted subtracts every listed file from the counted
// remainder, so a name added here removes that file's typed widths from the
// number story 047 is handed. An entry the diff does not support is an
// unaudited discount on the published debt.
//
// internal/common/report/render/style.go was such an entry and was removed
// here. It had been on this list since the list existed, while `git diff --stat
// c8e347e HEAD -- <it>` was empty — byte-identical to the branch point, so this
// story provably did not modify it. Its discount happened to be zero, because
// it carries no typed width, and that is exactly why nothing but a rule could
// have caught it: the number never looked wrong.
var filesThisStoryTouches = []string{
	"cmd/bentoo/main.go",
	"cmd/bentoo/overlay_autoupdate.go",
	"cmd/bentoo/overlay_autoupdate_report.go",
	"cmd/bentoo/overlay_autoupdate_ui.go",
	"cmd/bentoo/overlay_manifest.go",
	"cmd/bentoo/overlay_prune.go",
	"cmd/bentoo/overlay_validate.go",
	"cmd/bentoo/snapshot_run.go",
	"internal/common/report/classify.go",
	"internal/common/report/mode.go",
	"internal/common/report/model.go",
	"internal/common/report/render/fullscreen.go",
	"internal/common/report/render/inline.go",
	"internal/common/report/render/json.go",
	"internal/common/report/render/text.go",
	"internal/common/report/render/width.go",
	"internal/overlay/annotate_baseline.go",
	"internal/overlay/compare.go",
	"internal/overlay/manifest.go",
	"internal/snapshot/runner.go",
	"misc/design/design-system/component/catalogue.go",
	"misc/design/design-system/component/layout.go",
}

// typedWidthsIn returns every typed width in one Go file, as "line: literal".
func typedWidthsIn(t *testing.T, path string) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}

	var found []string
	ast.Inspect(file, func(node ast.Node) bool {
		lit, ok := node.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if typedWidth.MatchString(lit.Value) {
			found = append(found, fmt.Sprintf("%d: %s", fset.Position(lit.Pos()).Line, lit.Value))
		}
		return true
	})
	return found
}

// widthDebtSubjectGaps returns every file this story provably edited that
// filesThisStoryTouches does not account for, repo-relative.
//
// It is the check the list never had. R6.2's subject is "a file this story
// MODIFIES" and D9 spells that out as "a file this story provably edits" — the
// diff — while the list is a second, hand-kept declaration of the same set.
// Two declarations of one fact drift, and this pair did.
//
// The diff does not REPLACE the list, for the reason the list's own comment
// gives: a tree with no branch point to diff against — a shallow clone, a
// tarball of a tag — still has to be able to run the rule. So the list stays,
// and where a base commit resolves it is compared to what it claims to mirror.
// Where one does not, the caller skips and says which of the two reasons it
// was: a skip nobody can tell apart from a pass is the failure mode this whole
// cross-check exists to close.
func widthDebtSubjectGaps(t *testing.T) []string {
	t.Helper()
	return widthDebtSubjectGapsAgainst(t, filesThisStoryTouches)
}

// widthDebtSubjectGapsAgainst is widthDebtSubjectGaps with the subject list
// passed in, so the same sweep can be run against a list that is missing
// something and be watched reporting it (R8.3). A guard nobody has seen fail
// has demonstrated nothing about what it would catch.
//
// # What it sweeps, and what counts as a width — READ THIS BEFORE NARROWING IT
//
// The files swept are every path in `git diff --name-only <base>..HEAD` that is
// a .go file, is not a _test.go, and is still on disk. Test files are out for
// the same reason TestWidthDebtRemainderIsCounted has them out: this
// repository's standing reading of R6.2's subject is production code, the code
// that actually prints a column to somebody.
//
// The detector is a RAW TEXT match and deliberately NOT typedWidthsIn. The two
// answer different questions and want different amounts of precision:
//
//   - typedWidthsIn asks "does this file still DECLARE a width?" Its answer
//     puts a file in violation, so it must be exact, and it reads string
//     literals out of the parsed source precisely so that a comment explaining
//     a width somebody REMOVED is not reported as the defect for the lifetime
//     of the file.
//   - this asks "is this file ACCOUNTED FOR?" Its answer sends a human to look
//     at a file, not a file to the naughty step. Over-reporting costs one line
//     added to a list the file already belongs on, since it is in the diff
//     either way.
//
// Using the exact detector here instead leaves the sweep with nothing to find,
// which is worth working through rather than rediscovering. Once
// overlay_prune.go's three widths are measured, the only diff-touched file
// still carrying a typed width in a STRING LITERAL is width_debt_test.go — this
// file, holding the positive-control fixtures of TestTypedWidthDetectorSeesOne.
// Include test files and the cross-check reports its own guard's fixtures as a
// gap. Exclude them, as the paragraph above does, and nothing is left to report
// against ANY subject list, an empty one included. A sweep that finds nothing
// against an empty list finds nothing ever: it is the green-on-arrival guard
// R8.3 exists to reject, and TestSubjectCrossCheckCanFail is what rejects it.
//
// The broad net has a real price: a COMMENT mentioning %-45s inside a
// diff-touched production file that is not on the list gets reported. Three
// files carry such a comment today — overlay_autoupdate.go, overlay_validate.go
// and render/width.go, each explaining a width this story took out — and all
// three are on the list, where they belong. The remedy for a fourth is to add
// it. That trade is taken on purpose: this function exists because the OPPOSITE
// error was made silently and cost three live widths.
//
// overlay_prune.go is deliberately NOT a fourth: sub-task 11.3's validation
// greps that file for a typed width and requires nothing back, so its comment
// spells the number in words. That is why the counterfactual below is worth
// keeping in writing — this sweep WOULD have caught the original gap, because
// at the time of it overlay_prune.go held three live widths in string literals
// and was absent from the list. Measured 2026-08-30 by putting that file back
// as commit 55e6d06 left it, dropping it from the list, and running the sweep:
// it reported "cmd/bentoo/overlay_prune.go" and nothing else.
func widthDebtSubjectGapsAgainst(t *testing.T, subject []string) []string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("cross-check skipped: git is not on PATH (%v), so the diff cannot be read", err)
	}
	if err := exec.Command("git", "-C", repoRoot, "cat-file", "-e", storyBaseCommit+"^{commit}").Run(); err != nil {
		t.Skipf("cross-check skipped: base commit %s is not resolvable in this clone (%v), "+
			"so there is no diff to check the subject list against", storyBaseCommit, err)
	}

	// --name-only because the only thing wanted here is which paths changed.
	// Git prints them one per line in path order, so the gaps come back
	// deterministically ordered without this function sorting anything.
	out, err := exec.Command("git", "-C", repoRoot, "diff", "--name-only", storyBaseCommit+"..HEAD").Output()
	if err != nil {
		// Named, like the two skips above. git answering neither a diff nor a
		// reason would be a guard that switched itself off in silence.
		detail := err.Error()
		var exit *exec.ExitError
		if errors.As(err, &exit) && len(exit.Stderr) > 0 {
			detail = fmt.Sprintf("%v: %s", err, strings.TrimSpace(string(exit.Stderr)))
		}
		t.Skipf("cross-check skipped: `git diff --name-only %s..HEAD` could not be read (%s), "+
			"so there is no diff to check the subject list against", storyBaseCommit, detail)
	}

	accounted := make(map[string]bool, len(subject))
	for _, rel := range subject {
		accounted[rel] = true
	}

	var gaps []string
	for _, rel := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if rel == "" || !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			continue
		}
		if accounted[rel] {
			continue
		}

		source, readErr := os.ReadFile(filepath.Join(repoRoot, rel))
		if errors.Is(readErr, fs.ErrNotExist) {
			// In the diff and not on disk: this story deleted or renamed it.
			// There is no text to sweep and no remedy to ask anyone for.
			continue
		}
		if readErr != nil {
			t.Fatalf("reading %s, which the diff since %s names: %v", rel, storyBaseCommit, readErr)
		}

		if typedWidth.Match(source) {
			gaps = append(gaps, rel)
		}
	}
	return gaps
}

// TestTypedWidthDetectorSeesOne is the positive control, and it is not
// ceremony: a sweep whose pattern stopped matching would report zero typed
// widths across the whole repository and be read as the story having succeeded.
//
// The star case is the one that matters most. %-*s takes its width from an
// argument — it IS the measured form — so a detector that flagged it would make
// the fix impossible to apply.
func TestTypedWidthDetectorSeesOne(t *testing.T) {
	cases := map[string]bool{
		`"%-45s"`:                       true,
		`"    %-45s %s\n"`:              true,
		`"  %-14s"`:                     true,
		`"%-10.10s"`:                    true,
		`"%-*s"`:                        false, // measured: the width is an argument
		`"\r  Checking: [%3d%%] %d/%d"`: false, // a counter's format, not a column
		`"%s: %v"`:                      false,
		`"%d ok, %d failed"`:            false,
		`"no format at all"`:            false,
	}

	for literal, want := range cases {
		if got := typedWidth.MatchString(literal); got != want {
			t.Errorf("typedWidth.MatchString(%s) = %v, want %v", literal, got, want)
		}
	}
}

// TestWidthDebt is R6.2 itself: no file this story touches declares a width.
//
// The message names the file, the line and the literal, because "a typed width
// exists somewhere in the files you edited" costs its reader a search through
// twenty files — which is how a mechanical rule turns back into a conventional
// one.
func TestWidthDebt(t *testing.T) {
	scanned := 0

	for _, rel := range filesThisStoryTouches {
		path := filepath.Join(repoRoot, rel)
		if _, err := os.Stat(path); err != nil {
			// A file this story renamed or has not created yet. Reported
			// rather than skipped in silence: a list that quietly stopped
			// covering half the story would pass by covering nothing.
			t.Logf("not present, not scanned: %s (%v)", rel, err)
			continue
		}
		scanned++

		for _, hit := range typedWidthsIn(t, path) {
			t.Errorf("%s:%s declares a fixed column width (R6.2).\n"+
				"    Remedy: measure it. render.ColumnWidth(values) gives the width the values actually need, "+
				"in display cells, and render.Shorten bounds it — a width typed into a format string is wrong in "+
				"both directions at once, too narrow the moment one value outgrows it and too wide on every run "+
				"that never comes close.", rel, hit)
		}
	}

	if scanned == 0 {
		t.Fatal("scanned no file — the sweep passed without inspecting anything")
	}
}

// TestWidthDebtRemainderIsCounted is Constraint 8's hand-off: the widths in
// files this story does NOT touch are carried as a number story 047 can check,
// rather than as a description somebody has to believe.
//
// It fails if the debt GROWS. That is the trigger story 044 lacked — it
// recorded three widths as debt "to be repaid when one of them is next
// touched", none were touched, and the count reached 18.
//
// The count comes from measuredWidthDebtRemainder rather than from a walk
// written out here, and that sharing is load-bearing rather than tidiness. This
// test PUBLISHES the number; TestWidthDebtBaselineEqualsTheMeasurement asserts
// that widthDebtBaseline IS it. Those two statements only mean something about
// each other while one sweep produces both — two copies that drifted would pin
// the constant exactly against a count nobody publishes, which is the same
// two-declarations-of-one-fact failure that let overlay_prune.go sit outside
// filesThisStoryTouches for a whole story.
func TestWidthDebtRemainderIsCounted(t *testing.T) {
	remaining, perFile := measuredWidthDebtRemainder(t)

	// Sorted, so that two runs of the same tree print the same lines: this log
	// is the hand-off itself, and a breakdown that reorders on every run cannot
	// be diffed against the last one.
	files := make([]string, 0, len(perFile))
	for file := range perFile {
		files = append(files, file)
	}
	sort.Strings(files)

	t.Logf("typed column widths remaining in files this story does not touch: %d", remaining)
	for _, file := range files {
		t.Logf("    %2d  %s", perFile[file], file)
	}

	if remaining > widthDebtBaseline {
		t.Errorf("the typed-width debt GREW to %d, from the %d widthDebtBaseline publishes — the count is carried so that it can be checked, not so that it can rise (Constraint 8).\n"+
			"    Remedy: measure the width that pushed it over, named in the breakdown above. The ceiling is a\n"+
			"    measurement and may only fall; raising it to meet a new width is what left story 044 with 18.",
			remaining, widthDebtBaseline)
	}
}
