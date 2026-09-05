package main

import (
	"strings"
	"testing"

	"github.com/obentoo/bentoolkit/internal/overlay"
)

// This file carries two jobs, and the first one is inherited rather than new.
//
// UB3 guaranteed the user-visible summary — "Total packages scanned / Found in
// both / Only in Bentoo" — by saying that no task modifies printComparisonSummary.
// That was the only guarantee available in v1: the function lives in package
// main, and compare_nilmap_test.go lives in package overlay, which cannot reach
// it. Task 7.2 modifies it, so the guarantee is spent and this test replaces it
// with a direct pin of the three lines.
//
// The second job is R3.9: the four per-Verdict counters existed on CompareReport
// from task 2.2 and were read by nothing at all.
//
// Both assert on the pure line builders rather than on captured log output.
// logger binds its io.Writer once at first use and exposes no setter
// (logger.go:44-52), so capturing it would mean either adding production API for
// a test or swapping os.Stderr at the fd level. Splitting the decision from the
// emission is cheaper and leaves the emission trivial enough to read.

func summaryReport() *overlay.CompareReport {
	return &overlay.CompareReport{
		TotalPackages:    10,
		ComparedPackages: 8,
		NotInRemoteCount: 2,
		ErrorCount:       1,

		VerdictKeepCount:        4,
		VerdictRedundantCount:   2,
		VerdictNeedsRebaseCount: 1,
		VerdictUnknownCount:     3,
	}
}

// _Requirements: R3.9, UB3_
func TestPrintComparisonSummary(t *testing.T) {
	// Story 047, sub-task 5.5 (S047-R8.2): the subtest "the three pre-existing
	// lines are unchanged" stood here and was RETIRED with
	// comparisonSummaryLines, which sub-task 4.2 deletes.
	//
	// It pinned four literal bytes-for-bytes lines — "\nSummary:", "  Total
	// packages scanned: 10", "  Found in both repos: 5", "  Only in Bentoo: 2" —
	// as UB3's promise that an operator's habits could rely on them. Story 047
	// retires that promise deliberately: the same three facts open the report as
	// its scope lead ("N package(s) scanned in the Bentoo overlay. M also exist
	// in ::gentoo and are compared below; K exist only here..."), pinned in every
	// plain and markdown golden, and close it as the summary block.
	//
	// The FACTS are asserted on the payload, and one of them is now derived
	// DIFFERENTLY on purpose: "Found in both repos" was ComparedPackages minus
	// the errored and the not-in-remote, and CompareRun.InBoth is the three
	// statuses that required a remote version to be read, summed. That is not a
	// rename — a package whose lookup errored used to be subtracted and is now
	// simply not counted as in-both, on the argument that a subtraction files it
	// under "in both" on no evidence. TestBuildCompareReport/"the counts come
	// from the producer's own counters" in overlay_compare_report_test.go asserts
	// all three and states that reasoning.

	t.Run("each non-zero verdict count is printed", func(t *testing.T) {
		lines := verdictSummaryLines(summaryReport())
		if len(lines) != 1 {
			t.Fatalf("verdict summary has %d lines, want exactly 1: %q", len(lines), lines)
		}
		line := lines[0]

		for _, want := range []string{"keep 4", "redundant 2", "needs-rebase 1", "unknown 3"} {
			if !strings.Contains(line, want) {
				t.Errorf("verdict summary %q is missing %q", line, want)
			}
		}
	})

	t.Run("the verdict line cannot be read as a status count", func(t *testing.T) {
		line := verdictSummaryLines(summaryReport())[0]

		// The counter FIELDS carry a Verdict prefix so that a bare UnknownCount
		// beside the existing ErrorCount does not read as a Status count. The
		// printed line sits directly under "Errors (API issues)" and needs the
		// same protection, or UB3's separation of the two axes survives in the
		// struct and dies on screen.
		if !strings.Contains(strings.ToLower(line), "verdict") {
			t.Errorf("verdict summary %q never says which axis it counts; under the Errors line it reads as a Status count", line)
		}
	})

	t.Run("a zero count prints nothing", func(t *testing.T) {
		report := summaryReport()
		report.VerdictNeedsRebaseCount = 0
		report.VerdictUnknownCount = 0

		line := verdictSummaryLines(report)[0]

		if strings.Contains(line, "needs-rebase") {
			t.Errorf("verdict summary %q lists needs-rebase with a count of zero", line)
		}
		if strings.Contains(line, "unknown") {
			t.Errorf("verdict summary %q lists unknown with a count of zero", line)
		}
	})

	t.Run("no verdict line at all when every count is zero", func(t *testing.T) {
		report := &overlay.CompareReport{TotalPackages: 3}

		if lines := verdictSummaryLines(report); len(lines) != 0 {
			t.Errorf("verdict summary printed %q for a report with no verdicts at all", lines)
		}
	})

	t.Run("the counts come from the report, not from a filtered Results", func(t *testing.T) {
		// runCompare assigns the filtered slice back to report.Results before
		// rendering (task 5.2), deliberately leaving the counts as computed over
		// the whole scan: the summary answers "what is in the overlay", not
		// "what did you ask to see". Counting Results here would silently make
		// --only-redundant rewrite the overlay's own totals.
		report := summaryReport()
		report.Results = nil

		line := verdictSummaryLines(report)[0]

		if !strings.Contains(line, "keep 4") {
			t.Errorf("verdict summary %q lost its counts when Results was emptied — it is counting the filtered view instead of the scan", line)
		}
		// The same claim for the SCANNED total is now made about the payload, by
		// TestBuildCompareReportCountsTheWholeRunNotTheView in
		// overlay_compare_present_test.go: `func buildCompareReport` takes the
		// unfiltered results as a required third parameter precisely so a
		// narrowed view cannot narrow a count (story 047, sub-task 5.5,
		// S047-R8.2). Asserting it here as well would be a second answer to one
		// question, taken over a builder that is being deleted.
	})
}

// liveSummaryReport is a MEASURED report, not an invented one: `bentoo overlay
// compare gentoo` run against /var/db/repos/gentoo on 2026-08-07 scanned 318
// packages, found 84 of them only in Bentoo, and printed
// "Verdicts: keep 231 | redundant 74 | unknown 13" — 318 packages counted on the
// Verdict axis, since every compared package is counted exactly once there.
//
// rows is how many rows that same run actually PRINTED, which is the whole
// variable in this story and changes with the flags:
//
//	(no flags)         234 rows — 155 keep, 74 redundant, 5 unknown
//	--only-redundant    74 rows
//	--only-outdated      0 rows (the report is empty and prints no table at all)
//
// Only the LENGTH of Results is meaningful here — the builder under test counts
// rows and never reads one — so zero-valued results stand in for the packages.
func liveSummaryReport(rows int) *overlay.CompareReport {
	return &overlay.CompareReport{
		TotalPackages:    318,
		ComparedPackages: 318,
		NotInRemoteCount: 84,

		VerdictKeepCount:      231,
		VerdictRedundantCount: 74,
		VerdictUnknownCount:   13,

		Results: make([]overlay.CompareResult, rows),
	}
}

// R4 adds a FOURTH summary line, in its own builder, and the three UB3 froze
// stay exactly where they are.
//
// The line exists because the verdict counts and the tables above them count
// different things and never said so: on the live overlay the summary reports
// keep 231 above a keep table holding 155 rows, and a --only-outdated run reports
// 318 verdicts above no table at all. Both totals are correct; with nothing
// explaining the gap, the larger number reads as a defect (R4.1, R4.2).
//
// The name of this test is deliberate. The task's gate is
// `go test ./cmd/bentoo/ -run TestComparisonSummary`, and -run matches an
// unanchored regexp against the test NAME: it selects this function and does NOT
// select TestPrintComparisonSummary above. R4.3's byte pin is therefore repeated
// here rather than left to the neighbour the gate cannot reach.
//
// _Requirements: R4.1, R4.2, R4.3_
func TestComparisonSummaryScope(t *testing.T) {
	t.Run("the line states the scanned universe and how many are unlisted", func(t *testing.T) {
		lines := verdictScopeLines(liveSummaryReport(234))
		if len(lines) != 1 {
			t.Fatalf("scope summary has %d lines, want exactly 1: %q", len(lines), lines)
		}
		line := lines[0]

		// "84 of 318" pins the RELATION, not the sentence: how many are unlisted,
		// out of how many the counts cover. A line naming one number without the
		// other cannot settle whether a count larger than the rows on screen is a
		// fact or a bug, which is the whole reason for the line.
		if !strings.Contains(line, "84 of 318") {
			t.Errorf("scope summary %q does not say how many of how many are unlisted (want %q)", line, "84 of 318")
		}
		lower := strings.ToLower(line)
		for _, want := range []string{
			"verdict", // names the axis, as the line above it must (UB3's two axes)
			"scanned", // R4.1: counted over the scan…
			"row",     // …rather than over the rows printed
			"table",   // R4.2: they appear in no table
		} {
			if !strings.Contains(lower, want) {
				t.Errorf("scope summary %q never says %q; R4.1/R4.2 need it to say what the counts cover and what the number means", line, want)
			}
		}
	})

	t.Run("no line when every counted package has a row", func(t *testing.T) {
		// Nothing to explain: the counts and the tables agree, and a line saying
		// "0 of 10 have no row" would be noise. Same rule as the zero terms the
		// verdict line drops and the ErrorCount line above it.
		report := summaryReport() // ten packages on the Verdict axis
		report.Results = make([]overlay.CompareResult, 10)

		if lines := verdictScopeLines(report); len(lines) != 0 {
			t.Errorf("scope summary printed %q when every counted package is listed", lines)
		}
	})

	t.Run("no line when there are no verdict counts at all", func(t *testing.T) {
		// The line qualifies the verdict counts (R4.1's WHEN). With no counts
		// printed there is nothing for it to qualify, and verdictSummaryLines is
		// silent on the same report.
		report := &overlay.CompareReport{TotalPackages: 3}

		if lines := verdictScopeLines(report); len(lines) != 0 {
			t.Errorf("scope summary printed %q for a report with no verdicts at all", lines)
		}
	})

	t.Run("never a negative count", func(t *testing.T) {
		// Only a hand-built report can hold more rows than the counters admit,
		// and the answer to it is silence rather than "-2 of 10 have no row".
		report := summaryReport()
		report.Results = make([]overlay.CompareResult, 12)

		if lines := verdictScopeLines(report); len(lines) != 0 {
			t.Errorf("scope summary printed %q for more rows than counted packages", lines)
		}
	})

	t.Run("the count is the rows missing, not the Bentoo-only packages", func(t *testing.T) {
		// THE measurement that decides how this line is computed. On an unfiltered
		// run the two agree — 318 counted, 234 rows, 84 unlisted, and the report
		// holds NotInRemoteCount == 84 — because the only packages runCompare keeps
		// out of Results are the Bentoo-only ones (it never sets IncludeNotInRemote).
		//
		// Under a filter they diverge, and NotInRemoteCount is then simply wrong:
		// the measured --only-redundant run printed 74 rows against the same 318
		// verdicts, so 244 packages have no row while NotInRemoteCount still reads
		// 84. filterCompareResults narrows Results after the counting (D7), so what
		// is unlisted is a property of the rows, not of one Status counter.
		lines := verdictScopeLines(liveSummaryReport(74))
		if len(lines) != 1 {
			t.Fatalf("scope summary has %d lines, want exactly 1: %q", len(lines), lines)
		}
		line := lines[0]

		if !strings.Contains(line, "244 of 318") {
			t.Errorf("scope summary %q does not report the 244 packages the --only-redundant run left out of its 74 rows", line)
		}
		if strings.Contains(line, "84") {
			t.Errorf("scope summary %q reports NotInRemoteCount (84); under a filter that is not how many rows are missing", line)
		}
	})

	t.Run("an empty report is all unlisted", func(t *testing.T) {
		// The --only-outdated run measured above: every package is up-to-date, so
		// FormatReport is never reached and not one table prints, while the summary
		// still reports 318 verdicts. This is the case the line is most needed for.
		lines := verdictScopeLines(liveSummaryReport(0))
		if len(lines) != 1 {
			t.Fatalf("scope summary has %d lines, want exactly 1: %q", len(lines), lines)
		}
		if !strings.Contains(lines[0], "318 of 318") {
			t.Errorf("scope summary %q does not say that all 318 counted packages are unlisted", lines[0])
		}
	})

	// Story 047, sub-task 5.5 (S047-R8.2): "the three pre-existing lines are
	// unchanged and hold no new line" stood here and was RETIRED with
	// comparisonSummaryLines.
	//
	// Its point was NOT the bytes — the subtest above already pinned those — but
	// the SEPARATION: the scope line R4 adds must not grow out of
	// comparisonSummaryLines, so it was re-asserted over a report whose scope
	// line is non-empty. That separation survives in a stronger form. The scope
	// sentence is now a SECTION of the report's own summary, built by
	// internal/common/report from the payload, and `func verdictScopeLines` below
	// is asserted on its own; two builders cannot leak into each other when one
	// of them no longer exists. TestCompareRunSummaryCountsTheRunAndNotTheLists
	// in internal/common/report is where the summary's own counts are pinned.

}
