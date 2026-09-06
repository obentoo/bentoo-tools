package report

import (
	"strings"
	"testing"
)

// Sub-task 15.6 — S046-R8.3: SnapshotRun's Sections derivation, pinned in this
// package rather than through a renderer's golden, for the reason given in
// manifest_run_test.go — a golden's diff cannot say whether the sections moved
// or the writer did.
//
// Every name carries the TestSnapshotRunSections prefix.

// snapshotRunFixture is a completed run over one subvolume: two steps, one of
// each outcome, one of them a ship with a destination.
func snapshotRunFixture() SnapshotRun {
	return SnapshotRun{
		Subvolumes: []string{"/home"},
		Steps: []SnapshotStep{
			{Subvolume: "/home", Step: "create", Success: true},
			{Subvolume: "/home", Step: "ship", Target: "offsite", Success: false, Error: "ssh: connection refused"},
		},
		Ok:     1,
		Failed: 1,
	}
}

// TestSnapshotRunSectionsOrderAndLead pins the shape and the ORDER of the two
// lead sentences. Scope comes first and on every path: which subvolume a run
// operated on must not be a question whose answer depends on --all
// (S046-R1.6).
func TestSnapshotRunSectionsOrderAndLead(t *testing.T) {
	sections := snapshotRunFixture().Sections(SectionOptions{})

	if len(sections) != 1 {
		t.Fatalf("Sections returned %d sections, want 1", len(sections))
	}
	if sections[0].Title != "Snapshot Run" {
		t.Errorf("section title is %q, want %q", sections[0].Title, "Snapshot Run")
	}
	if len(sections[0].Lead) != 2 {
		t.Fatalf("lead has %d lines, want 2 (scope, then tally): %q", len(sections[0].Lead), sections[0].Lead)
	}
	if want := "The run operated on 1 subvolume(s): /home."; sections[0].Lead[0] != want {
		t.Errorf("the first lead line is %q, want %q — scope is stated before anything else", sections[0].Lead[0], want)
	}
	if want := "2 step(s) ran: 1 succeeded, 1 failed."; sections[0].Lead[1] != want {
		t.Errorf("the second lead line is %q, want %q", sections[0].Lead[1], want)
	}
}

// TestSnapshotRunSectionsTableColumns pins the column names, and that a ship's
// destination is folded into the STEP cell rather than given a column of its
// own — a fourth column would be blank on every create and every prune.
func TestSnapshotRunSectionsTableColumns(t *testing.T) {
	section := snapshotRunFixture().Sections(SectionOptions{ShowAll: true})[0]

	if got := strings.Join(section.Rows.Headers, ","); got != "SUBVOLUME,STEP,STATE" {
		t.Errorf("columns are %q, want \"SUBVOLUME,STEP,STATE\"", got)
	}
	if len(section.Rows.Rows) != 2 {
		t.Fatalf("--all listed %d rows over 2 steps", len(section.Rows.Rows))
	}
	if got := section.Rows.Rows[1].Cells[1]; got != "ship (offsite)" {
		t.Errorf("the ship step's cell is %q, want \"ship (offsite)\": the destination is folded into the step", got)
	}
	if got := section.Rows.Rows[1].Cells[2]; got != "failed" {
		t.Errorf("the failed step's state is %q, want \"failed\"", got)
	}
	if !strings.Contains(section.Rows.Rows[1].Detail, "connection refused") {
		t.Errorf("the failed step's detail is %q, want the producer's sentence whole", section.Rows.Rows[1].Detail)
	}
}

// TestSnapshotRunSectionsIncompleteRunStatesItsGap is the arm a golden cannot
// isolate: nothing counted, and counts without the rows they were taken over.
// Both are STATED (S046-R2.3), including the run given no subvolume at all,
// whose scope sentence must not trail off into an empty list.
func TestSnapshotRunSectionsIncompleteRunStatesItsGap(t *testing.T) {
	nothing := SnapshotRun{}.Sections(SectionOptions{})[0]
	if len(nothing.Lead) != 2 {
		t.Fatalf("a run with nothing counted has %d lead lines, want 2: %q", len(nothing.Lead), nothing.Lead)
	}
	if !strings.Contains(nothing.Lead[0], "given no subvolume") {
		t.Errorf("the scope sentence is %q, want the one that states there was no subvolume", nothing.Lead[0])
	}
	if !strings.Contains(nothing.Lead[1], "No step has an outcome to report") {
		t.Errorf("the second lead line is %q, want the sentence saying nothing was counted", nothing.Lead[1])
	}
	if len(nothing.Rows.Headers) != 0 || len(nothing.Rows.Rows) != 0 {
		t.Errorf("a run with nothing counted declared a table (%v / %d rows)", nothing.Rows.Headers, len(nothing.Rows.Rows))
	}

	counted := SnapshotRun{Subvolumes: []string{"/home"}, Ok: 2, Failed: 1}.Sections(SectionOptions{})[0]
	if want := "3 step(s) ran: 2 succeeded, 1 failed."; counted.Lead[1] != want {
		t.Errorf("the tally is %q, want %q — the counts are the run's, not len(Steps)", counted.Lead[1], want)
	}
	if len(counted.Notes) != 1 || !strings.Contains(counted.Notes[0], "No per-step row reached this report") {
		t.Errorf("notes are %q, want the sentence saying the counts are stated without their list", counted.Notes)
	}
}

// TestSnapshotRunSectionsShowAllChangesTheListNotTheCounts is the converse
// half, as in manifest_run_test.go: --all selects rows and moves no number.
func TestSnapshotRunSectionsShowAllChangesTheListNotTheCounts(t *testing.T) {
	fixture := snapshotRunFixture()

	def := fixture.Sections(SectionOptions{})[0]
	all := fixture.Sections(SectionOptions{ShowAll: true})[0]

	if len(def.Rows.Rows) != 1 {
		t.Errorf("by default %d rows were listed, want 1: a succeeded step is counted, not listed", len(def.Rows.Rows))
	}
	if len(all.Rows.Rows) != 2 {
		t.Errorf("with --all %d rows were listed, want 2", len(all.Rows.Rows))
	}
	if strings.Join(def.Lead, "|") != strings.Join(all.Lead, "|") {
		t.Errorf("--all changed the lead: %q vs %q; it selects rows and nothing else", def.Lead, all.Lead)
	}
	if len(def.Notes) == 0 || len(all.Notes) == 0 {
		t.Fatalf("one of the two dropped its notes (%d / %d)", len(def.Notes), len(all.Notes))
	}
	if !strings.Contains(def.Notes[0], "not listed") || !strings.Contains(all.Notes[0], "listed above") {
		t.Errorf("the note does not follow the listing: default %q, --all %q", def.Notes[0], all.Notes[0])
	}
}
