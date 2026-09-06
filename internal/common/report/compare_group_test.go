package report

import (
	"reflect"
	"strings"
	"testing"
)

// Authored for story 047, sub-task 2.3 — S047-R2.1, S047-R2.2, S047-R2.3,
// S047-R2.4, S047-R2.5.
//
// What GroupKeep DECIDES, pinned. compare_run_test.go pins what Sections says
// over a hand-built group and compare_run_json_test.go pins the document the
// payload marshals into; neither of them can fail when the rule that fills
// KeepGroups is wrong, because neither of them runs it.
//
// # One test per rule, so a failure names the rule
//
// The five requirements this sub-task implements are five different ways for a
// report to mislead, and a single test over a fixture would report all of them
// as one red:
//
//   - a pair that does not collapse says nothing about the repetition an
//     operator is being asked to read (S047-R2.1);
//   - a finding absorbed into a group hides the one row that needed action,
//     which is the failure grouping must never buy compression with
//     (S047-R2.2);
//   - a group of one is a second name for a single package (S047-R2.5);
//   - an order that is not total makes two runs over one overlay differ, and a
//     golden file impossible (S047-R2.4);
//   - a member the flat listing loses is data --all was supposed to reveal
//     (S047-R2.3).
//
// The determinism test is the one a single-run test cannot stand in for.
// Membership is accumulated in a map, Go randomises map iteration per run, and
// an implementation that emitted straight from the map would pass every
// assertion below on most runs.
//
// # The fixture's order is deliberately NOT the expected order
//
// compareGroupFixture lists the vulkan pair first and the gstreamer pair
// second, and interleaves the rust pair between gstreamer's members. An
// implementation that emitted groups in the order it first met them would
// therefore fail the ordering test rather than pass it by accident, which is
// what a fixture whose insertion order already equals its sorted order would
// have allowed.

// compareGroupFixture is a kept list shaped like the four groups the story's
// target rendering was built from, shrunk to the smallest input that still
// makes every rule decide something.
//
// It carries, on purpose: a pair with four members, two pairs with two members
// each that TIE on count, a pair with a single member, and a package whose
// version pair matches the largest group while it carries a finding of its own.
func compareGroupFixture() []ComparePkg {
	return []ComparePkg{
		{Package: "dev-util/vulkan-tools", Local: "1.4.361_p20260828", Remote: "1.4.357.0", Status: "newer"},
		{Package: "media-libs/vulkan-loader", Local: "1.4.361_p20260828", Remote: "1.4.357.0", Status: "newer"},
		{Package: "media-plugins/gst-plugins-good", Local: "1.29.2", Remote: "1.26.11", Status: "newer"},
		{Package: "dev-lang/rust", Local: "1.98.0", Remote: "1.97.1", Status: "newer"},
		{Package: "media-libs/gst-plugins-base", Local: "1.29.2", Remote: "1.26.11", Status: "newer"},
		{Package: "dev-lang/rust-bin", Local: "1.98.0", Remote: "1.97.1", Status: "newer"},
		{Package: "dev-python/gst-python", Local: "1.29.2", Remote: "1.26.11", Status: "newer"},
		{Package: "app-editors/vim", Local: "9.2.1021", Remote: "9.1.1652-r2", Status: "newer"},
		{Package: "media-plugins/gst-plugins-ugly", Local: "1.29.2", Remote: "1.26.11", Status: "newer"},
		{
			// The same version pair as the four above, and a finding. It must
			// stay its own row (S047-R2.2).
			Package: "media-plugins/gst-plugins-mpeg2dec",
			Local:   "1.29.2", Remote: "1.26.11", Status: "newer",
			Reason: "behind the rest of its own stack, which is at 1.29.2",
		},
	}
}

// compareGroupLabels is the label of every group returned, in order.
func compareGroupLabels(groups []KeepGroup) []string {
	labels := make([]string, 0, len(groups))
	for _, g := range groups {
		labels = append(labels, g.Label)
	}
	return labels
}

// compareGroupNamed finds one group by its label, so an assertion about a group
// does not depend on the position the ordering test already pins.
func compareGroupNamed(t *testing.T, groups []KeepGroup, label string) KeepGroup {
	t.Helper()
	for _, g := range groups {
		if g.Label == label {
			return g
		}
	}
	t.Fatalf("no group labelled %q: got %v", label, compareGroupLabels(groups))
	return KeepGroup{}
}

// TestKeepGroupCollectsPackagesSharingAVersionPair is S047-R2.1: two or more
// kept packages with an identical version pair and no finding become one group,
// carrying the pair they share and every member of it.
func TestKeepGroupCollectsPackagesSharingAVersionPair(t *testing.T) {
	groups := GroupKeep(compareGroupFixture())

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d: %v", len(groups), compareGroupLabels(groups))
	}

	gst := compareGroupNamed(t, groups, "gst")

	if gst.Local != "1.29.2" || gst.Remote != "1.26.11" {
		t.Errorf("group carries the wrong version pair: local %q remote %q", gst.Local, gst.Remote)
	}

	// The order inside a group is the run's own, which is what the tie-break
	// above reads and what KeepGroup.Members documents.
	want := []string{
		"media-plugins/gst-plugins-good",
		"media-libs/gst-plugins-base",
		"dev-python/gst-python",
		"media-plugins/gst-plugins-ugly",
	}
	if !reflect.DeepEqual(gst.Members, want) {
		t.Errorf("members are %v, want %v", gst.Members, want)
	}
}

// TestKeepGroupNeverAbsorbsAPackageCarryingAFinding is S047-R2.2, and it is the
// assertion that keeps compression from costing an operator the one row that
// needed action.
//
// Two halves, and the second is the one an implementation could quietly break.
// The package must be in no group's Members — and it must still be in the kept
// list, because GroupKeep produces a VIEW over that list and removing an entry
// from it would make the group the only place the package is named.
func TestKeepGroupNeverAbsorbsAPackageCarryingAFinding(t *testing.T) {
	const carrier = "media-plugins/gst-plugins-mpeg2dec"

	kept := compareGroupFixture()
	groups := GroupKeep(kept)

	for _, g := range groups {
		for _, member := range g.Members {
			if member == carrier {
				t.Errorf("group %q absorbed %s, which carries a finding", g.Label, carrier)
			}
		}
	}

	// Its version pair matches the largest group's, so an implementation that
	// keyed on the pair alone would have absorbed it: the group must be one
	// member short of the pair's population.
	gst := compareGroupNamed(t, groups, "gst")
	if len(gst.Members) != 4 {
		t.Errorf("gst group has %d members, want 4 — the fifth carries a finding", len(gst.Members))
	}

	found := false
	for _, p := range kept {
		if p.Package == carrier {
			found = true
		}
	}
	if !found {
		t.Errorf("%s is no longer in the kept list: grouping is a view over it, never a move out of it", carrier)
	}
}

// TestKeepGroupIsNeverAGroupOfOne is S047-R2.5: a version pair with a single
// member is an ordinary row.
//
// Both arms are asserted. The fixture's lone package must appear in no group,
// and a list in which NOTHING repeats must produce no groups at all rather than
// one group per package.
func TestKeepGroupIsNeverAGroupOfOne(t *testing.T) {
	for _, g := range GroupKeep(compareGroupFixture()) {
		if len(g.Members) < 2 {
			t.Errorf("group %q has %d member(s): a pair with one member is an ordinary row", g.Label, len(g.Members))
		}
		for _, member := range g.Members {
			if member == "app-editors/vim" {
				t.Errorf("group %q claims app-editors/vim, whose version pair nothing shares", g.Label)
			}
		}
	}

	none := GroupKeep([]ComparePkg{
		{Package: "app-editors/vim", Local: "9.2", Remote: "9.1"},
		{Package: "app-editors/emacs", Local: "30.2", Remote: "30.1"},
	})
	if len(none) != 0 {
		t.Errorf("expected no group over a list where no pair repeats, got %v", compareGroupLabels(none))
	}

	// The rule RAN and collected nothing, which the exported document says with
	// [] rather than with null (S047-D8).
	if none == nil {
		t.Error("GroupKeep returned nil: an empty result must still be a result")
	}
}

// TestKeepGroupOrderIsMemberCountThenFirstAtom is S047-R2.4, and the fixture is
// built so that both keys have to work.
//
// The largest group is not the first pair the list mentions, so an
// implementation emitting in insertion order fails on the count key. The two
// groups behind it hold two members each — a real tie — and the one whose first
// atom sorts earlier appears LATER in the input, so an implementation that
// broke the tie by insertion order fails too. Without both, the test would
// prove only the half that happened to be exercised.
func TestKeepGroupOrderIsMemberCountThenFirstAtom(t *testing.T) {
	groups := GroupKeep(compareGroupFixture())

	got := compareGroupLabels(groups)
	want := []string{"gst", "rust", "vulkan"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("group order is %v, want %v", got, want)
	}

	if len(groups[1].Members) != len(groups[2].Members) {
		t.Fatalf("the tie case does not tie: %d and %d members — the test proves only the count key",
			len(groups[1].Members), len(groups[2].Members))
	}
	if groups[1].Members[0] >= groups[2].Members[0] {
		t.Errorf("tied groups are ordered %q before %q, which is not the first member's atom ascending",
			groups[1].Members[0], groups[2].Members[0])
	}
}

// TestKeepGroupByCategoryIsCountDescendingThenCategory pins the breakdown's own
// order, which S047-R2.4's byte-identical requirement covers as much as it
// covers the groups.
//
// The gstreamer group is chosen because it ties: two members in media-plugins
// and one each in dev-python and media-libs. Count descending decides the
// first entry and cannot decide the other two, so the category name has to.
func TestKeepGroupByCategoryIsCountDescendingThenCategory(t *testing.T) {
	gst := compareGroupNamed(t, GroupKeep(compareGroupFixture()), "gst")

	want := []CategoryCount{
		{Category: "media-plugins", Count: 2},
		{Category: "dev-python", Count: 1},
		{Category: "media-libs", Count: 1},
	}
	if !reflect.DeepEqual(gst.ByCategory, want) {
		t.Errorf("breakdown is %v, want %v", gst.ByCategory, want)
	}

	total := 0
	for _, c := range gst.ByCategory {
		total += c.Count
	}
	if total != len(gst.Members) {
		t.Errorf("breakdown counts %d members, the group has %d", total, len(gst.Members))
	}
}

// TestKeepGroupIsIdenticalAcrossRepeatedCalls is the map-iteration hazard, and
// it is the one property a single call cannot observe.
//
// Membership is accumulated in a map keyed by version pair. Go randomises map
// iteration per run and per range, so an implementation that emitted groups —
// or a group's categories — straight out of the map would produce a different
// document on most runs while passing every other test here on the run that
// happened to come out sorted.
func TestKeepGroupIsIdenticalAcrossRepeatedCalls(t *testing.T) {
	first := GroupKeep(compareGroupFixture())

	for i := 0; i < 64; i++ {
		again := GroupKeep(compareGroupFixture())
		if !reflect.DeepEqual(first, again) {
			t.Fatalf("call %d produced a different result:\n first: %+v\n again: %+v", i+2, first, again)
		}
	}
}

// TestKeepGroupLabelIsTheSharedStem pins the label derivation: the longest
// prefix the members' package names share, cut back to where a token ends.
//
// # The target rendering's four labels are NOT these, and that is recorded
//
// The agreed target calls its groups "gstreamer stack", "rust toolchain",
// "vulkan sdk" and "mesa". Those are domain prose; nothing in this repository
// can derive them, and a hand-maintained map would put Gentoo domain knowledge
// in the package whose guards exist to keep domain out. The first four cases
// below are those same four groups, and what they produce is the stem —
// measured, not assumed.
//
// The last three cases are the ones that must not produce a bad label: a stem
// under the floor, members sharing no token at all, and a shared prefix that
// stops mid-token because the next character continues a run of digits.
func TestKeepGroupLabelIsTheSharedStem(t *testing.T) {
	cases := []struct {
		name  string
		atoms []string
		want  string
	}{
		{
			name:  "a prefix cut back to a separator",
			atoms: []string{"media-plugins/gst-plugins-good", "media-libs/gst-plugins-base", "dev-python/gst-python"},
			want:  "gst",
		},
		{
			name:  "a whole name that another extends",
			atoms: []string{"dev-lang/rust", "dev-lang/rust-bin"},
			want:  "rust",
		},
		{
			name:  "a prefix ending in the separator itself",
			atoms: []string{"dev-util/vulkan-tools", "media-libs/vulkan-loader"},
			want:  "vulkan",
		},
		{
			name:  "one name extended past an underscore",
			atoms: []string{"media-libs/mesa", "dev-util/mesa_clc"},
			want:  "mesa",
		},
		{
			name:  "a digit opens a token",
			atoms: []string{"x11-libs/gtk3", "gui-libs/gtk4"},
			want:  "gtk",
		},
		{
			name:  "a digit that continues a run opens nothing",
			atoms: []string{"dev-lang/python3", "dev-lang/python311"},
			want:  "python",
		},
		{
			name:  "a stem under the floor falls back to the first member's atom",
			atoms: []string{"app-misc/a-one", "app-misc/a-two"},
			want:  "app-misc/a-one",
		},
		{
			name:  "no shared token at all falls back to the first member's atom",
			atoms: []string{"app-editors/vim", "app-editors/emacs"},
			want:  "app-editors/vim",
		},
		{
			name:  "the same package name under two categories",
			atoms: []string{"dev-util/mesa", "media-libs/mesa"},
			want:  "mesa",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := compareGroupLabel(c.atoms); got != c.want {
				t.Errorf("label for %v is %q, want %q", c.atoms, got, c.want)
			}
		})
	}
}

// TestCompareGroupingFeedsTheKeepSection is the seam: what Sections renders
// once the groups are real rather than hand-built.
//
// # Three properties, and each is a different way for the seam to fail
//
// Collapsed, every group is one row and every package no group claimed is still
// a row of its own — including the one carrying a finding, which is what
// S047-R2.2 buys. Expanded, every kept package has a row, which is what --all
// means and the whole of what it changes (S047-R2.3). And the sentence under
// the table states the same numbers in both directions, because it reads the
// payload's own fields rather than the rows that were built: a count that moved
// when the listing did would disagree with the count above it exactly when an
// operator passed the flag to check (S047-D7).
func TestCompareGroupingFeedsTheKeepSection(t *testing.T) {
	kept := compareGroupFixture()
	run := CompareRun{
		Repository: "gentoo",
		Scanned:    279,
		InBoth:     180,
		OnlyLocal:  99,
		Keep:       kept,
		KeepGroups: GroupKeep(kept),
		Verdicts:   VerdictTally{Keep: 258},
	}

	collapsed := compareKeepBlock(t, run, SectionOptions{})
	expanded := compareKeepBlock(t, run, SectionOptions{ShowAll: true})

	// Three group rows, then the two packages no group claimed.
	wantCollapsed := []string{
		"gst (4 packages)",
		"rust (2 packages)",
		"vulkan (2 packages)",
		"app-editors/vim",
		"media-plugins/gst-plugins-mpeg2dec",
	}
	if got := compareFirstCells(collapsed); !reflect.DeepEqual(got, wantCollapsed) {
		t.Errorf("collapsed rows are %v, want %v", got, wantCollapsed)
	}

	wantExpanded := make([]string, 0, len(kept))
	for _, p := range kept {
		wantExpanded = append(wantExpanded, p.Package)
	}
	if got := compareFirstCells(expanded); !reflect.DeepEqual(got, wantExpanded) {
		t.Errorf("--all rows are %v, want the kept list %v", got, wantExpanded)
	}

	// 8 of the 10 are grouped, into 3 groups. Both sentences say so.
	collapsedNote := strings.Join(collapsed.Notes, " ")
	expandedNote := strings.Join(expanded.Notes, " ")
	if !reflect.DeepEqual(compareDigits(collapsedNote), compareDigits(expandedNote)) {
		t.Errorf("the note's numbers moved with the listing:\n without --all: %s\n with --all:    %s",
			collapsedNote, expandedNote)
	}
	for _, want := range []string{"8", "10", "3"} {
		if !strings.Contains(collapsedNote, want) {
			t.Errorf("the note does not state %s, which is a count the payload carries: %s", want, collapsedNote)
		}
	}
}

// compareKeepBlock is the keep section of one rendering, found by its title so
// that the assertions do not also pin the position the block list test owns.
func compareKeepBlock(t *testing.T, r CompareRun, opts SectionOptions) Section {
	t.Helper()
	for _, s := range r.Sections(opts) {
		if strings.HasPrefix(s.Title, "Keep") {
			return s
		}
	}
	t.Fatal("no keep section in the rendering")
	return Section{}
}

// compareFirstCells is the first cell of every row of a table: the label or the
// atom, which is what says whether a row stands for a group or for a package.
func compareFirstCells(s Section) []string {
	cells := make([]string, 0, len(s.Rows.Rows))
	for _, row := range s.Rows.Rows {
		if len(row.Cells) > 0 {
			cells = append(cells, row.Cells[0])
		}
	}
	return cells
}
