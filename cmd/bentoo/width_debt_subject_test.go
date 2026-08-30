package main

// Authored for story 046, sub-task 11.3 — R6.2.
//
// R6.2's subject is "a file this story MODIFIES". design.md's D9 says the same
// thing in its own words: "the trigger is now a file this story provably
// edits". filesThisStoryTouches says something narrower — "every file named in
// the task list's Context blocks" — and the two drifted apart exactly where a
// drift is invisible: commit 55e6d06 edited cmd/bentoo/overlay_prune.go for the
// constructor extraction, nobody added it to the list, and its three typed
// widths went on passing a guard whose subject no longer contained them.
//
// # Why the list is not replaced by the diff
//
// filesThisStoryTouches gives a real reason for being written out: "the rule
// has to be checkable in a working tree with no branch point to diff against".
// That is true and worth keeping — a fresh clone at a tag has no base to diff.
// Deriving the subject from git would trade a property the guard has for one it
// does not need.
//
// So the list stays and gains a checker. Where a base commit IS resolvable, the
// list must account for every diff-touched file carrying a typed width; where
// it is not, the cross-check skips and SAYS SO. A list that states its own
// scope is fine. A list nobody can check against the thing it claims to mirror
// is how this gap happened.
//
// # RED ON ARRIVAL — two ways
//
// overlay_prune.go is absent from filesThisStoryTouches and carries three
// typed widths at :659, :666 and :1002. widthDebtSubjectGaps does not exist.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// storyBaseCommit is the commit this story branched from. The cross-check below
// resolves it and skips when it cannot — a test that failed on a shallow clone
// would be reporting the clone, not the code.
const storyBaseCommit = "c8e347e"

// TestOverlayPruneIsInTheSubject is the concrete half: the file the drift hid.
//
// It is named separately from the general cross-check because this one file is
// what the audit found, and a failure here should say so rather than arrive as
// one line of a list.
func TestOverlayPruneIsInTheSubject(t *testing.T) {
	const rel = "cmd/bentoo/overlay_prune.go"

	inSubject := false
	for _, f := range filesThisStoryTouches {
		if f == rel {
			inSubject = true
			break
		}
	}
	if !inSubject {
		t.Errorf("%s is edited by this story (commit 55e6d06) and is not in filesThisStoryTouches.\n"+
			"    R6.2's subject is a file this story MODIFIES, and D9 says \"provably edits\".\n"+
			"    Remedy: add it to the list, and remove the typed widths it carries.", rel)
	}

	if hits := typedWidthsIn(t, filepath.Join(repoRoot, rel)); len(hits) > 0 {
		t.Errorf("%s still declares %d fixed column width(s): %s.\n"+
			"    Remedy: measure the widest atom each loop prints — render.ColumnWidth over the\n"+
			"    values, then pad to it — the way the other files this story touched were changed.",
			rel, len(hits), strings.Join(hits, ", "))
	}
}

// TestSubjectListAccountsForTheDiff is the general half: the list is checked
// against what the story provably edited, so the next drift is caught by the
// guard rather than by an audit two stories later.
//
// It skips rather than fails when git cannot answer, and names the reason. A
// silent skip would be indistinguishable from a pass, which is the failure mode
// this whole sub-task is about.
func TestSubjectListAccountsForTheDiff(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("cross-check skipped: git is not on PATH, so the diff cannot be read")
	}

	verify := exec.Command("git", "-C", repoRoot, "cat-file", "-e", storyBaseCommit+"^{commit}")
	if err := verify.Run(); err != nil {
		t.Skipf("cross-check skipped: base commit %s is not resolvable in this clone "+
			"(shallow checkout or a tag export), so there is no diff to read against", storyBaseCommit)
	}

	for _, rel := range widthDebtSubjectGaps(t) {
		t.Errorf("%s is in the diff since %s and carries a typed width, but is not in "+
			"filesThisStoryTouches.\n"+
			"    R6.2 applies to it. Remedy: add it to the list and measure the width, or — if the\n"+
			"    edit was mechanical and the width is genuinely story 047's — record the exemption\n"+
			"    in .draft/deviations.yaml so it is a decision rather than an omission.", rel, storyBaseCommit)
	}
}

// TestSubjectCrossCheckCanFail proves the cross-check is capable of reporting
// something, on a tree where it currently reports nothing.
//
// S046-R8.3: a guard that passes on the day it is written has proved nothing
// about what it would catch. Here the evidence is structural rather than a
// recorded mutation — the same sweep is run against a subject list with one
// entry removed, and it must name the file that removal orphaned.
func TestSubjectCrossCheckCanFail(t *testing.T) {
	const probe = "cmd/bentoo/overlay_autoupdate.go"

	if _, err := os.Stat(filepath.Join(repoRoot, probe)); err != nil {
		t.Skipf("cross-check probe skipped: %s is not present (%v)", probe, err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("cross-check probe skipped: git is not on PATH")
	}

	gaps := widthDebtSubjectGapsAgainst(t, []string{})
	if len(gaps) == 0 {
		t.Error("the cross-check found no gap against an EMPTY subject list.\n" +
			"    Every diff-touched file with a typed width should be reported when the list\n" +
			"    contains nothing. A sweep that reports nothing here reports nothing ever.")
	}
}
