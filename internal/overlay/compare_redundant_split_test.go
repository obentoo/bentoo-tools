package overlay

import (
	"reflect"
	"slices"
	"testing"
)

// This file pins the GROUPING of the redundant section: which packages the
// removal recommendation is allowed to cover (R3.1), which ones it must not
// (R3.2), and when the section must not be divided at all (R3.4).
//
// The failure it exists to prevent is measured rather than imagined. On the live
// overlay `overlay compare` recommends removing 74 packages and then warns,
// beneath the same table, that 8 of them differ from ::gentoo in content.
// Neither sentence can be acted on: the recommendation never says which 8 to
// skip, and the warning never says which 66 are safe, so an operator following
// the advice literally deletes the eight and an operator reading the warning
// follows none of it. The split is what makes the recommendation cover only what
// the content check actually cleared.
//
// It is a PURE function, tested with no renderer in sight, because the two
// questions are separate: which group a package belongs to is a decision about
// evidence, and the table that prints the group is a decision about layout (3.2).

// splitCase is one package in a redundant section together with what its content
// check found — the only two things the grouping reads.
type splitCase struct {
	pkg      string
	verified Verification
}

// redundantSection builds a section in the order given.
//
// Every result is the shape comparedResult (authorship_test.go) already produces:
// up-to-date, both versions equal, VerdictRedundant. That is not a simplification
// — it is the only shape a content check can ever run on, since
// resolvePackagePaths refuses two different versions, and it is what the live
// section holds: all 74 redundant packages are up-to-date at equal versions.
func redundantSection(cases ...splitCase) []CompareResult {
	results := make([]CompareResult, 0, len(cases))
	for _, c := range cases {
		results = append(results, comparedResult("kde-plasma", c.pkg, "6.7.4", c.verified))
	}
	return results
}

// sectionPackages names the packages of a group, in order, so a failure reads as
// the report does rather than as a wall of structs.
func sectionPackages(results []CompareResult) []string {
	names := make([]string, len(results))
	for i, r := range results {
		names[i] = r.Package
	}
	return names
}

// assertGroup fails when a group does not hold exactly the named packages IN
// ORDER.
//
// Order is asserted and not merely membership. report.Results arrives sorted by
// category/package (sortCompareResults) and FormatReport preserves that sort into
// every section; a grouping that reordered would reshuffle the table under the
// operator for no reason the report could state.
func assertGroup(t *testing.T, name string, got []CompareResult, want []string) {
	t.Helper()
	if names := sectionPackages(got); !slices.Equal(names, want) {
		t.Errorf("the %s group holds %q, want %q", name, names, want)
	}
}

// TestSplitRedundantSection covers the groupings the section can take, the case
// that must not be grouped at all, and the promise that grouping is all it does
// to the caller's slice.
//
// _Requirements: R3.1, R3.2, R3.4_
func TestSplitRedundantSection(t *testing.T) {
	t.Run("a mixed section splits, each group in source order (R3.1, R3.2)", func(t *testing.T) {
		// The live section's own opening rows, in the live order: breeze and
		// breeze-plymouth are byte-identical to ::gentoo's, breeze-gtk and drkonqi
		// are two of the eight that differ.
		section := redundantSection(
			splitCase{"breeze", VerifiedIdentical},
			splitCase{"breeze-gtk", VerifiedDiffers},
			splitCase{"breeze-plymouth", VerifiedIdentical},
			splitCase{"drkonqi", VerifiedDiffers},
		)

		identical, differing, split := splitRedundantSection(section)

		if !split {
			t.Fatalf("split = false for a section whose content WAS compared; the operator is back to one "+
				"recommendation covering %d packages and a warning about %d of them", len(section), len(differing))
		}
		assertGroup(t, "identical", identical, []string{"breeze", "breeze-plymouth"})
		assertGroup(t, "differing", differing, []string{"breeze-gtk", "drkonqi"})
	})

	t.Run("an all-identical section leaves nothing out of the recommendation", func(t *testing.T) {
		section := redundantSection(
			splitCase{"aurorae", VerifiedIdentical},
			splitCase{"bluedevil", VerifiedIdentical},
		)

		identical, differing, split := splitRedundantSection(section)

		if !split {
			t.Fatalf("split = false for a section every package of which was content-checked")
		}
		assertGroup(t, "identical", identical, []string{"aurorae", "bluedevil"})
		assertGroup(t, "differing", differing, nil)
	})

	t.Run("an all-differing section recommends no removal at all", func(t *testing.T) {
		section := redundantSection(
			splitCase{"kwin", VerifiedDiffers},
			splitCase{"plasma-desktop", VerifiedDiffers},
		)

		identical, differing, split := splitRedundantSection(section)

		if !split {
			t.Fatalf("split = false for a section every package of which was content-checked")
		}
		// The empty group is what makes the removal recommendation disappear: 3.2
		// renders no table for it, exactly as FormatReport already prints no section
		// for a verdict nothing carries.
		assertGroup(t, "identical", identical, nil)
		assertGroup(t, "differing", differing, []string{"kwin", "plasma-desktop"})
	})

	t.Run("a section no content check reached does not split (R3.4)", func(t *testing.T) {
		// The API-provider case, exactly: with no provider.PackageDirProvider there
		// is no ::gentoo copy on disk to compare against, so every result carries
		// NotVerified and the report has nothing to divide the section BY. Dividing
		// it anyway would print a "verified identical — safe to remove" heading over
		// packages nobody looked at.
		section := redundantSection(
			splitCase{"breeze", NotVerified},
			splitCase{"kwin", NotVerified},
			splitCase{"spectacle", NotVerified},
		)

		identical, differing, split := splitRedundantSection(section)

		if split {
			t.Errorf("split = true for a section no content check ran on; the two groups would be a claim about "+
				"evidence that does not exist (identical %q, differing %q)",
				sectionPackages(identical), sectionPackages(differing))
		}
		// THE CONTRACT, stated because the caller ignores both slices in this case
		// and an unstated one is a contract only until someone reads it differently:
		// when split is false BOTH groups are nil. The caller renders the original
		// section as one undivided table with the "no content was checked" note, and
		// a caller that ignored split and ranged over these would print two empty
		// tables rather than one table full of the wrong claim.
		if identical != nil || differing != nil {
			t.Errorf("split = false yet the groups are %q/%q, want both nil: with nothing checked there is no group "+
				"to render and the caller has the whole section already",
				sectionPackages(identical), sectionPackages(differing))
		}
	})

	t.Run("an unchecked package beside checked ones is never recommended for removal", func(t *testing.T) {
		// The case the requirements do not spell out, and it is decided on the
		// asymmetry the rest of this package already turns on.
		//
		// R3.1's group is "the packages whose compared ebuilds are IDENTICAL", under
		// a heading that recommends removal. A package whose ebuilds were never
		// compared is not in that set — nobody looked — and putting it there would
		// make the report vouch for a check it never ran, which is the exact failure
		// the split exists to end.
		//
		// R3.2's group recommends nothing and states what is UNRESOLVED, which is
		// precisely true of a package no check reached. So the unchecked ones join
		// it: the cost of that is one manual diff on a package that may well have
		// been fine, against the cost of the alternative, which is advising the
		// removal of a package nobody examined.
		//
		// The section is deliberately NOT in alphabetical order, so an implementation
		// that sorted instead of preserving the caller's order fails here.
		section := redundantSection(
			splitCase{"breeze", VerifiedIdentical},
			splitCase{"plasma-firewall", NotVerified},
			splitCase{"kwin", VerifiedDiffers},
		)

		identical, differing, split := splitRedundantSection(section)

		if !split {
			t.Fatalf("split = false for a section where two of three packages were content-checked; " +
				"R3.4 withholds the split only where NOTHING was checked")
		}
		assertGroup(t, "identical", identical, []string{"breeze"})
		assertGroup(t, "differing", differing, []string{"plasma-firewall", "kwin"})
	})

	t.Run("an empty section does not split", func(t *testing.T) {
		// No result carries a verification, vacuously, so the answer is the same one
		// R3.4 gives — and it costs no special case. FormatReport never renders an
		// empty section in any event.
		identical, differing, split := splitRedundantSection(nil)

		if split {
			t.Errorf("split = true for an empty section")
		}
		if identical != nil || differing != nil {
			t.Errorf("the groups are %q/%q for an empty section, want both nil",
				sectionPackages(identical), sectionPackages(differing))
		}
	})

	t.Run("the caller's backing array is not rewritten", func(t *testing.T) {
		// Interleaved on purpose. An in-place partition — the `kept := results[:0]`
		// idiom this package already uses elsewhere — would write the identical
		// results over indexes 0 and 1, and the caller's slice would come back
		// holding breeze twice: a report that lists a package that is not in the
		// overlay and drops one that is. A function whose whole job is to GROUP must
		// not be able to do that.
		section := redundantSection(
			splitCase{"breeze-gtk", VerifiedDiffers},
			splitCase{"breeze", VerifiedIdentical},
			splitCase{"drkonqi", VerifiedDiffers},
			splitCase{"breeze-plymouth", VerifiedIdentical},
		)
		before := slices.Clone(section)

		identical, differing, _ := splitRedundantSection(section)

		assertSectionUnchanged(t, section, before, "the split rewrote the section it was given")

		// The second way the caller's array can be reached: a group that RESLICES
		// it rather than copying into a new one. Nothing is asserted about what the
		// caller does with the groups — it is that whatever it does cannot reach
		// back through them.
		if len(identical) > 0 {
			identical[0].Package = "rewritten-through-the-identical-group"
		}
		if len(differing) > 0 {
			differing[0].Package = "rewritten-through-the-differing-group"
		}
		assertSectionUnchanged(t, section, before, "a returned group aliases the caller's backing array")
	})
}

// assertSectionUnchanged compares a section element for element against the copy
// taken before the split.
//
// Whole results, not just their names: every field is part of what the report
// prints, and a grouping has no business editing any of them.
//
// DeepEqual rather than ==: CompareResult stopped being comparable when the
// baseline review's slice fields (Axes, Declarations, Others) landed on it. The
// assertion is unchanged and now reaches further — those slices are part of what
// the report prints too, and a grouping must not edit them either.
func assertSectionUnchanged(t *testing.T, got, want []CompareResult, why string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: the section holds %d results, want %d", why, len(got), len(want))
	}
	for i := range want {
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Errorf("%s: results[%d] is now %s/%s (verified %d), was %s/%s (verified %d)",
				why, i, got[i].Category, got[i].Package, got[i].Verified,
				want[i].Category, want[i].Package, want[i].Verified)
		}
	}
}

// The rest of this file is the constraint the whole story inherits: A FINDING
// NEVER CHANGES A VERDICT (R3.5, restating 025 R4.5). The declaration in
// `.autoupdate/packages.toml` and the version comparison decide; everything else
// this story adds — the diff counts, the authorship proof, this grouping —
// describes what was decided and rearranges where it is printed.
//
// Held as a test rather than as a sentence because the grouping is exactly where
// it would be easiest to lose. A split that "helpfully" downgraded a differing
// package's Verdict would make the two tables agree with their headings and
// would silently change what `prune` is later told about that package, since both
// commands read the same field.

// splitFixtureAtom is the fixture's atom for a package, spelled the way the
// report's own maps key it.
func splitFixtureAtom(r CompareResult) string { return r.Category + "/" + r.Package }

// splitFixtureReport is one report carrying BOTH groups of the split and every
// other verdict beside them, with every counter non-zero.
//
// Every counter has to be non-zero or the assertion that it did not move is
// satisfied by 0 == 0, which a grouping that zeroed the report would also
// satisfy. The redundant section holds all THREE verifications on purpose: the
// two the split separates, plus the unchecked one, whose group membership is a
// judgement (see splitRedundantSection) and whose disappearance would otherwise
// be the easiest thing in the report to miss.
//
// Results are in the category/package order CompareWithProvider leaves behind, so
// the fixture is the shape FormatReport actually receives.
func splitFixtureReport() *CompareReport {
	return &CompareReport{
		TotalPackages:    9,
		ComparedPackages: 9,
		OutdatedCount:    1,
		NewerCount:       1,
		UpToDateCount:    5,
		NotInRemoteCount: 1,
		ErrorCount:       1,

		VerdictKeepCount:        3,
		VerdictRedundantCount:   4,
		VerdictNeedsRebaseCount: 1,
		VerdictUnknownCount:     1,

		Results: []CompareResult{
			{
				Category: "app-editors", Package: "zed",
				LocalVersion: "1.0", RemoteVersion: "2.0", Status: StatusOutdated,
				Verdict: VerdictNeedsRebase, Patched: true, PatchedBy: "app-editors/zed@stable",
			},
			{
				Category: "app-misc", Package: "ours",
				LocalVersion: "0.3", Status: StatusNotInRemote, Verdict: VerdictKeep,
			},
			{
				Category: "dev-lang", Package: "rust",
				LocalVersion: "1.90", RemoteVersion: "1.90", Status: StatusUpToDate,
				Verdict: VerdictKeep, Patched: true, PatchedBy: "dev-lang/rust",
			},
			{
				Category: "dev-lang", Package: "zig",
				LocalVersion: "0.16", RemoteVersion: "0.15", Status: StatusNewer, Verdict: VerdictKeep,
			},
			// The redundant section, as the live one reads: identical, differing,
			// differing, and one the check never reached.
			comparedResult("kde-plasma", "breeze", "6.7.4", VerifiedIdentical),
			comparedResult("kde-plasma", "breeze-gtk", "6.7.4", VerifiedDiffers),
			comparedResult("kde-plasma", "kwin", "6.7.4", VerifiedDiffers),
			comparedResult("kde-plasma", "plasma-firewall", "6.7.4", NotVerified),
			{
				Category: "sys-devel", Package: "binutils",
				LocalVersion: "2.47", Status: StatusError, Verdict: VerdictUnknown,
			},
		},
	}
}

// redundantResults extracts the section the split is given, the way FormatReport
// builds it: appended in Results order, so the sort survives.
func redundantResults(report *CompareReport) []CompareResult {
	var section []CompareResult
	for _, r := range report.Results {
		if r.Verdict == VerdictRedundant {
			section = append(section, r)
		}
	}
	return section
}

// verdictByAtom snapshots the verdict of every package, keyed by atom, so the
// comparison afterwards is package by package rather than in aggregate. Two
// verdicts that swapped between two packages would leave every total intact.
func verdictByAtom(results []CompareResult) map[string]Verdict {
	verdicts := make(map[string]Verdict, len(results))
	for _, r := range results {
		verdicts[splitFixtureAtom(r)] = r.Verdict
	}
	return verdicts
}

// reportCounter is one of the report's counters together with the name the
// failure message has to print — a number that moved is unactionable until the
// report says which one.
type reportCounter struct {
	name  string
	value int
}

// reportCounters returns EVERY counter the report carries, both axes.
//
// The per-Verdict four are what R3.5 is about, and they are listed first. The
// per-Status ones follow because the grouping has no business touching those
// either, and because the two axes are counted separately on purpose (see
// CompareReport) — a change that collapsed them would show up here first.
//
// A slice rather than a map: the order of a failure report must not come from map
// iteration.
func reportCounters(report *CompareReport) []reportCounter {
	return []reportCounter{
		{"VerdictKeepCount", report.VerdictKeepCount},
		{"VerdictRedundantCount", report.VerdictRedundantCount},
		{"VerdictNeedsRebaseCount", report.VerdictNeedsRebaseCount},
		{"VerdictUnknownCount", report.VerdictUnknownCount},
		{"TotalPackages", report.TotalPackages},
		{"ComparedPackages", report.ComparedPackages},
		{"OutdatedCount", report.OutdatedCount},
		{"NewerCount", report.NewerCount},
		{"UpToDateCount", report.UpToDateCount},
		{"NotInRemoteCount", report.NotInRemoteCount},
		{"ErrorCount", report.ErrorCount},
	}
}

// TestSplitLeavesVerdicts is R3.5 made executable: the grouping arranges the
// report and decides nothing.
//
// _Requirements: R3.4, R3.5_
func TestSplitLeavesVerdicts(t *testing.T) {
	report := splitFixtureReport()

	// The fixture is checked before it is trusted. "Unchanged" is satisfied by
	// 0 == 0 and by an empty group, so a fixture that had quietly stopped holding
	// both groups, or a counter that was zero all along, would turn each assertion
	// below into a tautology.
	for _, c := range reportCounters(report) {
		if c.value == 0 {
			t.Fatalf("fixture check: %s is 0, so asserting it unchanged asserts nothing", c.name)
		}
	}

	section := redundantResults(report)
	if len(section) != report.VerdictRedundantCount {
		t.Fatalf("fixture check: the redundant section holds %d results but the counter says %d",
			len(section), report.VerdictRedundantCount)
	}
	sectionBefore := slices.Clone(section)
	verdictsBefore := verdictByAtom(report.Results)
	countersBefore := reportCounters(report)

	identical, differing, split := splitRedundantSection(section)
	if !split {
		t.Fatalf("fixture check: the section did not split, so neither group is exercised")
	}
	if len(identical) == 0 || len(differing) == 0 {
		t.Fatalf("fixture check: the split produced %d identical and %d differing; a group that is empty here "+
			"cannot show a verdict moving inside it", len(identical), len(differing))
	}

	t.Run("every package keeps the verdict it arrived with", func(t *testing.T) {
		// The GROUPS are checked, not only the report: these are the rows 3.2
		// renders, and a split that rewrote a verdict on its way into one of them
		// would print a table disagreeing with the report it came from.
		for _, group := range [][]CompareResult{identical, differing} {
			for _, r := range group {
				atom := splitFixtureAtom(r)
				want, known := verdictsBefore[atom]
				if !known {
					t.Errorf("the split returned %s, which was not in the section it was given", atom)
					continue
				}
				if r.Verdict != want {
					t.Errorf("%s arrives in a group as %v, went in as %v; the grouping arranges the report "+
						"and decides nothing (R3.5) — and `prune` later reads the same field", atom, r.Verdict, want)
				}
			}
		}

		// And the report the caller is still holding, package by package: a swap
		// between two packages leaves every total intact.
		for _, r := range report.Results {
			atom := splitFixtureAtom(r)
			if want := verdictsBefore[atom]; r.Verdict != want {
				t.Errorf("%s is now %v in the report, was %v", atom, r.Verdict, want)
			}
		}
		// The section handed in is the caller's slice, and the split may not write
		// through it either.
		assertSectionUnchanged(t, section, sectionBefore, "the split rewrote the section it was given")
	})

	t.Run("every per-verdict counter is where it was", func(t *testing.T) {
		countersAfter := reportCounters(report)
		for i, before := range countersBefore {
			if after := countersAfter[i]; after.value != before.value {
				t.Errorf("%s = %d after the split, was %d; the counts are taken over every scanned package "+
					"and a grouping never rescopes them", after.name, after.value, before.value)
			}
		}
	})

	t.Run("the two groups together are exactly the section", func(t *testing.T) {
		// The failure this subtest exists for is invisible to everything above. A
		// grouping that silently DROPPED a package keeps every verdict identical and
		// every counter identical — and the package simply stops being printed,
		// while the summary goes on counting it. The report would then disagree with
		// itself, which is the one thing FormatReport's "unlisted" fallback already
		// refuses to allow for a verdict with no section.
		want := make(map[string]int, len(sectionBefore))
		for _, r := range sectionBefore {
			want[splitFixtureAtom(r)]++
		}
		got := make(map[string]int, len(sectionBefore))
		for _, group := range [][]CompareResult{identical, differing} {
			for _, r := range group {
				got[splitFixtureAtom(r)]++
			}
		}

		if total := len(identical) + len(differing); total != len(sectionBefore) {
			t.Errorf("the two groups hold %d results for a section of %d", total, len(sectionBefore))
		}
		for atom, n := range want {
			switch c := got[atom]; {
			case c == 0:
				t.Errorf("%s is in neither group: it vanishes from the report while the summary goes on counting it", atom)
			case c != n:
				t.Errorf("%s appears %d times across the two groups, want %d; a package listed twice is a removal "+
					"recommendation and a refusal for the same ebuild", atom, c, n)
			}
		}
		for atom := range got {
			if want[atom] == 0 {
				t.Errorf("%s appears in a group but was not in the section", atom)
			}
		}
	})
}

// The RENDERER half (S025-R3.2) stood below and was RETIRED by story 047,
// sub-task 5.5 (S047-R8.2). Four tests, all of them taken over FormatReport's
// text, which sub-task 4.2 deletes — together with the scaffolding that cut that
// text into blocks (reportBlocks, blockFor, tableRows, assertTableHolds,
// splitRenderSection, redundantSection's render fixtures) and the two constants
// pruneCommand and otherPackagesHeading.
//
// The GROUPING half above is untouched and still passes unmodified:
// TestSplitRedundantSection and TestSplitLeavesVerdicts assert
// splitRedundantSection, a pure function over results, and they never render.
//
// # Where each retired claim went
//
//   - TestFormatReportSplitsTheRedundantSection — "two headings, each over its
//     own packages" and "the recommendation covers only the cleared group". The
//     TWO-TABLE split is gone as a shape: story 047's redundant section is one
//     table whose Lead is DERIVED from the reading states, so "which group was
//     cleared" is now stated in the lead and in each row's Diff cell instead of
//     by which heading a row sits under. Covered by
//     TestCompareRunRedundantLeadNamesTheRefusedVersionPair (all three arms) in
//     internal/common/report and by
//     testdata/TestCompareGoldenPlain.golden, where the lead reads "The
//     recommendation to remove covers the 1 somebody read, and none of the rest"
//     over a table in which the unread rows are visibly the rest.
//   - TestDifferingTableSpeaksForWhatWasNeverChecked — "the note names the
//     packages nothing compared". Covered by the same lead's refusal clause ("1
//     were never compared at all, because the check refuses to diff two
//     different versions"), asserted in
//     TestCompareRunRedundantLeadNamesTheRefusedVersionPair/"nothing read" and
//     pinned in TestCompareGoldenPlain and TestCompareGoldenPlainNoneRead. The
//     row-level half — a package nobody compared shows "not compared" and not a
//     diff — is pinned in both those goldens and in TestCompareJSONGolden.
//   - TestRedundantSectionUndividedWithoutContent — the API-only run. The
//     undivided/​divided distinction no longer exists, so the branch it guarded
//     is gone rather than moved. What it also asserted — that the run states its
//     own emptiness of evidence — is covered by
//     TestCompareGoldenPlainNoneRead.golden, which is exactly an all-unread run.
//     Its third subtest pinned the literal heading "Redundant Packages (::gentoo
//     ships the same or more)"; the heading is now "Redundant — ::gentoo ships
//     the version, and nothing cleared the content check", pinned in every plain
//     and markdown golden.
//   - TestSplitLeavesTheReportWhole — "every package is printed exactly once"
//     and "the unlisted fallback stays unreached". Structural now rather than
//     textual: `func buildCompareReport` switches on the verdict, so a package
//     reaches exactly one of the four payload lists and there is no fallback
//     heading to reach. Covered by TestBuildCompareReport/"each verdict lands in
//     its own list" in cmd/bentoo. Its summary subtest asserted the per-status
//     lines ("Outdated: 1", "Up-to-date: 5", "Other: 3", "Total: 9"); those
//     counters are still asserted directly, on the report rather than on its
//     text, by TestSplitLeavesVerdicts' own survival table above.
//
// # ONE claim was NOT carried over, and it is a real loss
//
// R3.3 required a removal recommendation to NAME the command that acts on it
// safely: `redundantPruneAdvice` in compare.go — "Nothing is deleted here: act
// on it with 'bentoo overlay prune', which decides on content ... and never on
// the verdict alone." Nothing in internal/common/report or cmd/bentoo emits it,
// and no golden contains it, so once FormatReport is deleted the compare report
// recommends removals and points at no command. The authorship caveat survived
// the move verbatim; this sentence did not. It is recorded rather than quietly
// dropped — see /tmp/047-triage.md, Group A, LOST #2.
