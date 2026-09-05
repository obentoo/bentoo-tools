// Package main's tests for the three RUN-LEVEL summaries the report lost when it
// stopped rendering through `func FormatReport` in internal/overlay, and which
// `func compareRunNotes` now rebuilds.
//
// # Why they were lost, and why nothing went red when they were
//
// All three name no package. A summary with no package cannot be a Finding — a
// package-scoped finding with a blank atom is a bug — so none of them had an atom
// to travel on, and the move off the old renderer carried the numbers across
// while leaving the sentences behind. Every test that guarded them asserted the
// BUILDER in internal/overlay, which still exists and still passes; not one
// asserted the output an operator reads. These tests assert the output.
//
// # What each of them must not do
//
// A note that fires always is as wrong as one that never fires, so every case
// here is written in pairs: the run that must produce the sentence, and the run
// that must stay silent. That is most of the point for the prune advice, which
// recommends a DESTRUCTIVE command and must never be offered where the
// recommendation it belongs to was not made.
//
// _Requirements: S047-R7.2, S025-R3.3, S034-R8.1_
package main

import (
	"strings"
	"testing"

	"github.com/obentoo/bentoolkit/internal/overlay"
)

// classifiedResult is a result the run classified, which is the only kind the
// run-level share counts.
func classifiedResult(pkg string, versionMove, ours, unclassified int) overlay.CompareResult {
	return overlay.CompareResult{
		Category:   "dev-libs",
		Package:    pkg,
		Classified: overlay.Classified{VersionMove: versionMove, Ours: ours, Unclassified: unclassified},
	}
}

// TestCompareRunClassificationShareReachesTheReport pins the run-level
// classification block, which S034-R8.1 asks for as every count WITH its
// denominator (S034-R8.1, S047-R7.2).
func TestCompareRunClassificationShareReachesTheReport(t *testing.T) {
	t.Run("the share is stated over the packages it was taken across", func(t *testing.T) {
		rep := &overlay.CompareReport{Results: []overlay.CompareResult{
			classifiedResult("foo", 10, 4, 6),
			classifiedResult("bar", 2, 1, 7),
			// Never classified: no readable baseline, or no review requested.
			// Counting it as a package with zero differences would inflate the
			// reach of the classification with a row nobody looked at.
			{Category: "dev-libs", Package: "baz"},
		}}

		notes := compareRunNotes(rep, false, false, false)
		if len(notes) != 1 || notes[0].atom != "" {
			t.Fatalf("notes are %+v, want exactly one RUN-scoped classification note", notes)
		}

		got := notes[0].text
		for _, want := range []string{
			"30 differences",     // 12 + 3 + 13, the total the three classes sum to
			"across 2",           // the packages the classification actually reached
			"12 were attributed", // version move
			"5 to us",            // ours
			"13 to neither",      // unclassified
		} {
			if !strings.Contains(got, want) {
				t.Errorf("the classification note reads %q; it is missing %q, and a count without the number it is a share of is not a report of a share (S034-R8.1)", got, want)
			}
		}
	})

	t.Run("a run that classified nothing states no share", func(t *testing.T) {
		// Every run that asked for no review, which is the default one.
		rep := &overlay.CompareReport{Results: []overlay.CompareResult{
			{Category: "dev-libs", Package: "foo"},
			{Category: "dev-libs", Package: "bar"},
		}}
		if notes := compareRunNotes(rep, false, false, false); len(notes) != 0 {
			t.Errorf("a run with no classification said %+v; the share is emitted only for a review run", notes)
		}
	})
}

// TestComparePruneAdviceNamesTheCommandOnlyWhereRemovalIsRecommended is the
// regression S025-R3.3 asks for: the report recommends removing packages and,
// without this note, names no way to do it (S025-R3.3, S047-R7.2).
func TestComparePruneAdviceNamesTheCommandOnlyWhereRemovalIsRecommended(t *testing.T) {
	redundant := func(pkg string, reading overlay.Reading) overlay.CompareResult {
		return overlay.CompareResult{
			Category: "dev-libs",
			Package:  pkg,
			Verdict:  overlay.VerdictRedundant,
			Reading:  reading,
		}
	}

	t.Run("a redundant package somebody read gets the command", func(t *testing.T) {
		rep := &overlay.CompareReport{Results: []overlay.CompareResult{
			redundant("foo", overlay.ReadingDone),
			redundant("bar", overlay.ReadingFailed),
		}}

		notes := compareRunNotes(rep, false, false, false)
		if len(notes) != 1 || notes[0].atom != "" {
			t.Fatalf("notes are %+v, want exactly one RUN-scoped prune-advice note", notes)
		}
		got := notes[0].text
		if !strings.Contains(got, "bentoo overlay prune") {
			t.Errorf("the removal advice reads %q; a report that recommends a destructive action and names no command asks an operator to invent one (S025-R3.3)", got)
		}
		if !strings.Contains(got, "content") || !strings.Contains(got, "never on the verdict alone") {
			t.Errorf("the removal advice reads %q; it must say the command decides on CONTENT, or an operator reads any difference from the verdicts above as a bug", got)
		}
	})

	t.Run("a recommendation nobody's reading supports gets no command", func(t *testing.T) {
		// `func compareRemovalAdvice` in internal/common/report/compare_run.go
		// answers "No removal advice follows from a package nobody read." here.
		// Naming a destructive command under that sentence would offer an action
		// for an empty list.
		rep := &overlay.CompareReport{Results: []overlay.CompareResult{
			redundant("foo", overlay.ReadingFailed),
			redundant("bar", overlay.ReadingNotComparable),
			redundant("baz", overlay.ReadingNotRequested),
		}}
		if notes := compareRunNotes(rep, false, false, false); len(notes) != 0 {
			t.Errorf("an unsupported removal recommendation still said %+v; no removal advice follows from a package nobody read", notes)
		}
	})

	t.Run("a run with no redundant package at all gets no command", func(t *testing.T) {
		rep := &overlay.CompareReport{Results: []overlay.CompareResult{
			{Category: "dev-libs", Package: "foo", Verdict: overlay.VerdictKeep, Reading: overlay.ReadingDone},
		}}
		if notes := compareRunNotes(rep, false, false, false); len(notes) != 0 {
			t.Errorf("a run recommending no removal said %+v; there is nothing for the command to act on", notes)
		}
	})
}

// TestComparePartialRealignmentIsCountedNotInferred pins the refinement: the
// no-verdict notice must reflect judgement DELIVERED, not a reviewer that merely
// existed (S047-R7.2).
func TestComparePartialRealignmentIsCountedNotInferred(t *testing.T) {
	t.Run("a reviewer that failed some of its calls is reported with the count", func(t *testing.T) {
		// judged is `reviewer != nil` at the caller — a model was REACHABLE.
		// Three of the eight divergences still came back unjudged, and the old
		// flag-only condition produced no note at all for this run.
		rep := &overlay.CompareReport{RealignAsked: 8, RealignNoVerdict: 3}

		notes := compareRunNotes(rep, true, true, false)
		if len(notes) != 1 || notes[0].atom != "" {
			t.Fatalf("notes are %+v, want exactly one RUN-scoped partial-realignment note", notes)
		}
		got := notes[0].text
		if !strings.Contains(got, "3 of the 8") {
			t.Errorf("the partial-realignment note reads %q; it must state how many came back unjudged out of how many were asked", got)
		}
		if !strings.Contains(got, "not judged") {
			t.Errorf("the partial-realignment note reads %q; an unjudged divergence is not a justified one, and the note exists to say so", got)
		}
	})

	t.Run("a run where every divergence came back judged says nothing", func(t *testing.T) {
		rep := &overlay.CompareReport{RealignAsked: 8, RealignNoVerdict: 0}
		if notes := compareRunNotes(rep, true, true, false); len(notes) != 0 {
			t.Errorf("a fully judged realignment said %+v; there is nothing to qualify", notes)
		}
	})

	t.Run("no realignment at all says nothing about verdicts", func(t *testing.T) {
		// The counters are zero on a run that asked nothing, but the guard is
		// realignRan: a stale counter must not speak for a run that never ran.
		rep := &overlay.CompareReport{RealignAsked: 8, RealignNoVerdict: 3}
		if notes := compareRunNotes(rep, false, false, false); len(notes) != 0 {
			t.Errorf("a run that never realigned said %+v", notes)
		}
	})

	t.Run("the two branches are exclusive", func(t *testing.T) {
		// Nothing is asked when no reviewer exists, so this shape is not
		// reachable in production — but if it ever were, an operator must not be
		// told both that nothing was judged and that some of it was.
		notes := compareRunNotes(&overlay.CompareReport{RealignAsked: 8, RealignNoVerdict: 3}, true, false, true)
		if len(notes) != 1 {
			t.Fatalf("notes are %+v, want exactly one no-verdict notice", notes)
		}
		if !strings.Contains(notes[0].text, "--no-review") {
			t.Errorf("the notice reads %q, want the flag-derived branch: no model was contacted, so no count over asked divergences applies", notes[0].text)
		}
	})
}

// TestCompareRunSummariesAreOrderedForReading pins the order the three arrive in
// beside the notes already there, because a run-scoped note carries no heading
// and its position is the only thing that groups it (S047-R7.2).
func TestCompareRunSummariesAreOrderedForReading(t *testing.T) {
	rep := &overlay.CompareReport{
		ComparedPackages: 4,
		NoBaselineCount:  1,
		RealignAsked:     2,
		RealignNoVerdict: 1,
		Results: []overlay.CompareResult{
			classifiedResult("foo", 1, 2, 3),
			{Category: "dev-libs", Package: "bar", Verdict: overlay.VerdictRedundant, Reading: overlay.ReadingDone},
		},
	}

	notes := compareRunNotes(rep, true, true, false)
	if len(notes) != 4 {
		t.Fatalf("notes are %+v, want four: the no-baseline count, the classification share, the partial realignment, and the prune advice", notes)
	}

	want := []string{"no ::gentoo counterpart", "Classification:", "Realignment verdicts:", "bentoo overlay prune"}
	for i, marker := range want {
		if !strings.Contains(notes[i].text, marker) {
			t.Errorf("note %d reads %q, want the one carrying %q — the run's own state is read before the action it recommends", i, notes[i].text, marker)
		}
		if notes[i].atom != "" {
			t.Errorf("note %d carries atom %q; a run-level summary names no package, which is why it is a note at all", i, notes[i].atom)
		}
	}
}
