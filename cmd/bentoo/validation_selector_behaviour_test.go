package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

// Sub-task 17.4 — S046-R4.4, S046-R8.3: a `Tests:` field that names a behaviour
// has a `-run` pattern that actually reaches it.
//
// 15.3's guard (validation_selector_test.go, the sibling this file extends)
// decides that AT LEAST ONE test of the named file is selected. Its own header
// says that is all a name-level check can decide, and that the other half —
// "the RIGHT ones" — is left to a person reading. This file closes one
// mechanically decidable slice of that half.
//
// # THE RULE
//
// For a test file THIS STORY ADDED, every `func Test` the file declares must be
// selected by some sub-task's `-run` pattern. The sub-task that NAMES the file
// is where the failure points, because its `Tests:` field is the claim that
// went unproven.
//
// The defect the rule catches, twice over at the commit this was written
// against: sub-task 6.2's Validation is `go test ./cmd/bentoo/ -run
// TestSnapshotReport`, which selects two of the three tests in the file its own
// `Tests:` field names. The unselected one is the ONLY place the snapshot
// envelope's kind is asserted anywhere in the tree — so 6.2's stated behaviour
// "the envelope names the snapshot kind" ships behind a gate that never runs
// the assertion. Sub-task 5.2 has the identical shape for the manifest kind.
// Both pass 15.3's guard, by construction, because both DO select something.
//
// # THE TWO EXEMPTIONS, AND WHY THE RULE WOULD BE WRONG WITHOUT THEM
//
//  1. A PRE-EXISTING file is not swept. Sub-tasks 1.2, 2.1, 2.3, 3.2, 5.6, 7.1
//     and 7.2 all extend a file that predates story 046 and holds unrelated
//     neighbours — 2.1's named file declares 23 tests, 15 of them about a
//     renderer this story never touched. Requiring a sub-task's pattern to
//     select a stranger's tests would be a demand no correct plan could meet.
//     The story's own files have no strangers in them, which is exactly what
//     makes the rule decidable there. Membership is read from
//     `git diff --name-only --diff-filter=A <base>..HEAD`, the storyBaseCommit
//     the anchor guard already uses.
//
//  2. A test some OTHER sub-task's pattern selects, over the same package, is
//     covered. A story file grows: sub-task 10.4 created a file that 13.1 and
//     16.5 later added tests to, and 9.1's file gained a test at 13.4. Holding
//     the first sub-task responsible for a test written three tasks later would
//     make the guard red on a plan that is correct. What the rule insists on is
//     that SOME targeted gate reaches every test, not which one.
//
//     A pattern-free command — `go test ./...` on a Commit sub-task — is NOT a
//     cover. It runs everything, so honouring it would exempt everything and
//     leave the guard vacuous. The claim under audit is what the sub-task's own
//     targeted gate proves.
//
// # WHAT THIS STILL DOES NOT DECIDE — read before trusting a green run
//
// Declaration-completeness, not correctness. A selected test can still assert
// the wrong thing, and a test the story added to a PRE-EXISTING file is out of
// scope by exemption 1 — 11.3's defect, four tests selected and none of them
// its own, lives in that gap and only a reader caught it. A green run here
// means: no test in a file story 046 created is unreachable from every
// targeted gate in the plan.
//
// # GOING GREEN
//
// Widening the offending patterns is the whole fix — no test is renamed and no
// sub-task outside the flagged list is touched. If a future edit makes this
// guard demand changes across many sub-tasks at once, the rule has drifted
// broader than the two exemptions above allow and should be narrowed again
// rather than satisfied.
//
// Every name carries the TestValidationSelectorBehaviour prefix.

// storyAddedTestFiles is the set of _test.go files story 046 added, relative to
// the repository root. Files the story merely EDITED are deliberately absent:
// they may hold neighbours nobody in this plan owns.
func storyAddedTestFiles(t *testing.T) map[string]bool {
	t.Helper()

	if err := exec.Command("git", "-C", repoRoot, "cat-file", "-e", storyBaseCommit+"^{commit}").Run(); err != nil {
		t.Skipf("selector-behaviour guard skipped: the story base commit %s does not resolve here (%v), so "+
			"the set of files this story added cannot be derived. A declared no-op rather than a silent pass.",
			storyBaseCommit, err)
	}
	out, err := exec.Command("git", "-C", repoRoot, "diff", "--name-only", "--diff-filter=A", storyBaseCommit+"..HEAD").Output()
	if err != nil {
		t.Skipf("selector-behaviour guard skipped: `git diff --name-only --diff-filter=A %s..HEAD` could not be read (%v)",
			storyBaseCommit, err)
	}

	added := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		rel := strings.TrimSpace(line)
		if strings.HasSuffix(rel, "_test.go") {
			added[rel] = true
		}
	}
	return added
}

// patternsByPackage indexes every targeted gate in the plan by the package
// directory its command runs, which is how exemption 2 is decided.
func patternsByPackage(subTasks []subTaskValidation) map[string][]string {
	index := map[string][]string{}
	for _, sub := range subTasks {
		for _, dir := range sub.dirs {
			clean := filepath.Clean(strings.TrimPrefix(strings.TrimSuffix(dir, "/"), "./"))
			clean = strings.TrimSuffix(clean, "/...")
			index[clean] = append(index[clean], sub.patterns...)
		}
	}
	return index
}

// unreachedTests returns the tests `own` does not select and no other gate over
// the same package selects either — the ones no targeted command in the plan
// ever runs.
func unreachedTests(t *testing.T, names, own, elsewhere []string) (selected, missed []string) {
	t.Helper()

	// Each pattern is matched on its own rather than joined into one regexp:
	// an empty pattern joined with "|" matches every name, which would turn a
	// parse slip into a silent blanket pass.
	reaches := func(patterns []string, name string) bool {
		for _, pattern := range patterns {
			if pattern == "" {
				continue
			}
			if selectsAny(t, pattern, []string{name}) {
				return true
			}
		}
		return false
	}

	for _, name := range names {
		switch {
		case reaches(own, name):
			selected = append(selected, name)
		case reaches(elsewhere, name):
			// Covered by another sub-task's targeted gate over the same package.
		default:
			missed = append(missed, name)
		}
	}
	return selected, missed
}

// behaviourFault names the five things needed to act without re-deriving any of
// them: the sub-task, the file it claims, the pattern it gates with, what that
// pattern does reach, and what it leaves unreached.
func behaviourFault(number, file, pattern string, selected, missed []string) string {
	return fmt.Sprintf(
		"sub-task %s: `-run %s` reaches %d of the %d tests in %s.\n"+
			"    selects: %s\n"+
			"    MISSES:  %s\n"+
			"    Story 046 created this file, so it holds no test belonging to another sub-task, and no\n"+
			"    other targeted `-run` in the plan reaches the missed names either — the behaviours they\n"+
			"    assert are gated by nothing. The sub-task's Tests: field claims them; its Validation does\n"+
			"    not run them. Remedy: widen the pattern until it selects every test the file declares.\n"+
			"    Do NOT rename a test to fit a pattern, and do not delete the unreached one.",
		number, pattern, len(selected), len(selected)+len(missed), file,
		strings.Join(selected, ", "), strings.Join(missed, ", "))
}

// TestValidationSelectorBehaviourGatesEveryTestItsFileDeclares is the guard.
func TestValidationSelectorBehaviourGatesEveryTestItsFileDeclares(t *testing.T) {
	subTasks := readSubTaskValidations(t)
	added := storyAddedTestFiles(t)
	index := patternsByPackage(subTasks)

	swept, faults := 0, 0
	var pending, preexisting []string

	for _, sub := range subTasks {
		for _, file := range sub.files {
			path := filepath.Join(repoRoot, file)
			if _, err := os.Stat(path); err != nil {
				// Task 16 is open and four of its sub-tasks name files not yet
				// written. Skipped, and named below: a skip nobody can tell
				// apart from a pass is the failure mode this guard exists for.
				pending = append(pending, sub.number+" "+file)
				continue
			}
			if !added[file] {
				preexisting = append(preexisting, sub.number+" "+file)
				continue
			}

			swept++
			names := testNamesIn(t, path)
			if len(names) == 0 {
				faults++
				t.Errorf("sub-task %s: %s declares no `func Test` at all", sub.number, file)
				continue
			}

			pkg := filepath.Dir(file)
			var elsewhere []string
			for _, pattern := range index[pkg] {
				if !slices.Contains(sub.patterns, pattern) {
					elsewhere = append(elsewhere, pattern)
				}
			}

			selected, missed := unreachedTests(t, names, sub.patterns, elsewhere)
			if len(missed) > 0 {
				faults++
				t.Errorf("%s", behaviourFault(sub.number, file, strings.Join(sub.patterns, " / "), selected, missed))
			}
		}
	}

	sort.Strings(pending)
	sort.Strings(preexisting)
	t.Logf("swept %d story-added test files across %d sub-tasks; %d faults", swept, len(subTasks), faults)
	t.Logf("SKIPPED — named in a Tests: field, not in the tree yet (%d): %s", len(pending), strings.Join(pending, "; "))
	t.Logf("SKIPPED — pre-existing file, may hold neighbours this plan does not own (%d): %s",
		len(preexisting), strings.Join(preexisting, "; "))

	if swept == 0 {
		t.Fatal("the sweep read 0 story-added files: a guard that reads nothing cannot fail, so a zero " +
			"count is a broken guard rather than a clean plan")
	}
}

// TestValidationSelectorBehaviourCanFail is the hostile half for the classifier
// itself. A decision procedure that approved everything would report a clean
// plan forever; one that approved nothing would be deleted within a week. Both
// directions are asserted here, and the exemption is asserted to be an
// exemption rather than a blanket pass.
func TestValidationSelectorBehaviourCanFail(t *testing.T) {
	names := []string{"TestSnapshotReportNamesIt", "TestSnapshotReportStatesIt", "TestSnapshotExportIsToldApartByItsKindAlone"}

	// Wrongly collapses: a partial gate must NOT read as full coverage.
	if _, missed := unreachedTests(t, names, []string{"TestSnapshotReport"}, nil); len(missed) != 1 {
		t.Errorf("a pattern reaching 2 of 3 tests was read as reaching all of them; the classifier approves everything (missed %v)", missed)
	}
	// Wrongly splits: a gate that does reach everything must not be faulted.
	if _, missed := unreachedTests(t, names, []string{"TestSnapshot"}, nil); len(missed) != 0 {
		t.Errorf("a pattern that reaches every test was faulted anyway; the classifier refuses everything (missed %v)", missed)
	}
	// The exemption covers the named test, and only it.
	if _, missed := unreachedTests(t, names, []string{"TestSnapshotReport"}, []string{"TestSnapshotExport"}); len(missed) != 0 {
		t.Errorf("a test another sub-task's gate reaches was still faulted; exemption 2 does not apply (missed %v)", missed)
	}
	if _, missed := unreachedTests(t, names, []string{"TestSnapshotReport"}, []string{"TestSomethingUnrelated"}); len(missed) != 1 {
		t.Errorf("an unrelated gate was accepted as cover; the exemption is a blanket pass (missed %v)", missed)
	}

	message := behaviourFault("6.2", "cmd/bentoo/snapshot_report_test.go", "TestSnapshotReport",
		names[:2], names[2:])
	for _, want := range []string{"6.2", "cmd/bentoo/snapshot_report_test.go", "TestSnapshotReport", "MISSES", names[2]} {
		if !strings.Contains(message, want) {
			t.Errorf("the failure message does not name %q; a message that stops at \"mismatch\" gets deleted rather than answered.\n%s", want, message)
		}
	}
}
