package report

import (
	"strings"
	"testing"
)

// Sub-task 15.6 — S046-R8.3: ManifestRun's Sections derivation, pinned where
// design.md's Unit level puts it — in this package, with no renderer in the
// picture.
//
// ManifestRun was exercised only through render's goldens until now, which
// couples the derivation to the rendering: when a golden moves, its diff does
// not say whether the sections changed or the writer did, and telling those two
// apart is the whole point of D1 and D2.
//
// What is pinned here is what the payload DECIDES: how many sections and in
// what order, each section's lead, the column names its tables declare, and
// what a run with nothing to report states. Nothing about width, colour or
// layout — those belong to render.
//
// Every name carries the TestManifestRunSections prefix.

// manifestRunFixture is a completed run: two targets, one of each outcome.
func manifestRunFixture() ManifestRun {
	return ManifestRun{
		Targets: []ManifestTarget{
			{Package: "app-misc/ok", Success: true},
			{Package: "app-misc/bad", Success: false, Error: "exit status 1", Output: "boom"},
		},
		Ok:     1,
		Failed: 1,
	}
}

// TestManifestRunSectionsOrderAndLead pins the shape: one section, named, with
// the counts the run established stated before any row.
func TestManifestRunSectionsOrderAndLead(t *testing.T) {
	sections := manifestRunFixture().Sections(SectionOptions{})

	if len(sections) != 1 {
		t.Fatalf("Sections returned %d sections, want 1: the payload describes one run", len(sections))
	}
	if sections[0].Title != "Manifest Regeneration" {
		t.Errorf("section title is %q, want %q", sections[0].Title, "Manifest Regeneration")
	}
	if len(sections[0].Lead) != 1 {
		t.Fatalf("lead has %d lines, want 1: %q", len(sections[0].Lead), sections[0].Lead)
	}
	if want := "2 target(s) processed: 1 regenerated, 1 failed."; sections[0].Lead[0] != want {
		t.Errorf("lead is %q, want %q — the counts are what the run established (S046-R1.5)", sections[0].Lead[0], want)
	}
}

// TestManifestRunSectionsTableColumns pins the column names each arm declares.
// The two arms differ on purpose: a preview has no outcome to put in a STATE
// column, and declaring one would print a blank cell for every row.
func TestManifestRunSectionsTableColumns(t *testing.T) {
	run := manifestRunFixture().Sections(SectionOptions{})[0]
	if got := strings.Join(run.Rows.Headers, ","); got != "PACKAGE,STATE" {
		t.Errorf("a completed run declares columns %q, want \"PACKAGE,STATE\"", got)
	}

	preview := ManifestRun{
		Targets: []ManifestTarget{{Package: "app-misc/one"}, {Package: "app-misc/two"}},
		DryRun:  true,
	}.Sections(SectionOptions{})[0]
	if got := strings.Join(preview.Rows.Headers, ","); got != "PACKAGE" {
		t.Errorf("a dry run declares columns %q, want \"PACKAGE\": nothing ran, so no row has a state", got)
	}
	if len(preview.Rows.Rows) != 2 {
		t.Errorf("a dry run listed %d rows over 2 targets: on a preview the list IS the finding", len(preview.Rows.Rows))
	}
	if want := "Dry run: 2 target(s) would have their Manifest regenerated."; preview.Lead[0] != want {
		t.Errorf("dry-run lead is %q, want %q — the count is the length of the list, not Ok+Failed", preview.Lead[0], want)
	}
}

// TestManifestRunSectionsIncompleteRunStatesItsGap is the arm that matters most
// and the one a renderer golden cannot isolate: a run that reached no outcome,
// and a run whose counts arrived without the rows they were taken over. Each
// must SAY so (S046-R2.3) rather than render an empty frame.
func TestManifestRunSectionsIncompleteRunStatesItsGap(t *testing.T) {
	nothing := ManifestRun{}.Sections(SectionOptions{})[0]
	if len(nothing.Lead) != 1 || !strings.Contains(nothing.Lead[0], "No target has an outcome to report") {
		t.Errorf("a run with nothing counted leads with %q, want the sentence saying no target has an outcome", nothing.Lead)
	}
	if len(nothing.Rows.Rows) != 0 || len(nothing.Rows.Headers) != 0 {
		t.Errorf("a run with nothing counted declared a table (%v / %d rows); the sentence stands alone",
			nothing.Rows.Headers, len(nothing.Rows.Rows))
	}

	// Counts established, rows lost: the counts are still stated, and the
	// missing list is stated too. Asserting the gap is stated is what stops a
	// report claiming a run did nothing when it did.
	counted := ManifestRun{Ok: 3, Failed: 1}.Sections(SectionOptions{})[0]
	if want := "4 target(s) processed: 3 regenerated, 1 failed."; counted.Lead[0] != want {
		t.Errorf("lead is %q, want %q", counted.Lead[0], want)
	}
	if len(counted.Notes) != 1 || !strings.Contains(counted.Notes[0], "No per-target row reached this report") {
		t.Errorf("notes are %q, want the sentence saying the counts are stated without their list (S046-R2.3)", counted.Notes)
	}
}

// TestManifestRunSectionsShowAllChangesTheListNotTheCounts is the converse
// half: --all is about which rows are LISTED. A derivation that let it move a
// count would make two documents of one run disagree.
func TestManifestRunSectionsShowAllChangesTheListNotTheCounts(t *testing.T) {
	fixture := manifestRunFixture()

	def := fixture.Sections(SectionOptions{})[0]
	all := fixture.Sections(SectionOptions{ShowAll: true})[0]

	if len(def.Rows.Rows) != 1 {
		t.Errorf("by default %d rows were listed, want 1: a succeeded target is counted, not listed", len(def.Rows.Rows))
	}
	if len(all.Rows.Rows) != 2 {
		t.Errorf("with --all %d rows were listed, want 2", len(all.Rows.Rows))
	}
	if def.Lead[0] != all.Lead[0] {
		t.Errorf("--all changed the counts: %q vs %q; it selects rows and nothing else", def.Lead[0], all.Lead[0])
	}
	if len(def.Notes) == 0 || len(all.Notes) == 0 {
		t.Fatalf("one of the two dropped its notes (%d / %d): what was left out is stated either way (S046-R2.3)",
			len(def.Notes), len(all.Notes))
	}
	if !strings.Contains(def.Notes[0], "not listed") || !strings.Contains(all.Notes[0], "listed above") {
		t.Errorf("the note does not follow the listing: default %q, --all %q", def.Notes[0], all.Notes[0])
	}
}
