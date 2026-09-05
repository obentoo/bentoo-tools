package overlay

import (
	"strconv"
	"strings"
	"testing"
)

// Story 047, sub-task 5.5 — S047-R8.2: TestFormatReportVerdict stood here and
// was RETIRED, subtest by subtest, because sub-task 4.2 deletes FormatReport.
//
// It was a BOTH test — real facts asserted through printed text — so each of its
// six subtests is accounted for individually rather than the whole being waved
// through as "the output changed".
//
//  1. "the table carries a Verdict column". FORMATTING. The verdict is no longer
//     a COLUMN: it is the SECTION. testdata/TestCompareGoldenPlain.golden pins
//     all four verdict sections by name, in order, which is a stronger claim
//     than a header substring. The Status column survives it as STATE, pinned in
//     the same golden's every table, and by TestCompareRunTablesNameTheirColumns
//     in internal/common/report.
//  2. "every package's row states its verdict". FORMATTING, in its new shape:
//     membership of a section IS the statement. Covered by
//     TestBuildCompareReport/"each verdict lands in its own list" in cmd/bentoo,
//     which asserts the routing on the payload rather than on a row's substring,
//     and by TestCompareGoldenPlain, which shows the rows under their sections.
//  3. "packages are grouped by verdict". Same coverage, and the golden is the
//     stricter form: contiguity was inferred from string offsets here, and there
//     it is structural — a row is in a section or it is not.
//  4. "a redundant package is named a removal candidate". Covered positively by
//     testdata/TestCompareGoldenPlain.golden ("The recommendation to remove
//     covers the 1 somebody read, and none of the rest") and by all three arms
//     of TestCompareRunRedundantLeadNamesTheRefusedVersionPair.
//  5. "no removal is suggested when nothing is redundant". This one had NO
//     cover, and it is not a formatting claim — it is the negative half of the
//     recommendation, which is the sentence an operator deletes packages on. It
//     was REWRITTEN into internal/common/report, where the renderer that now
//     owns the sentence lives, as
//     TestCompareRunEmptyRedundantRecommendsNothing. Note that the old
//     assertion could not have survived a move unchanged even in principle: it
//     was `!strings.Contains(out, "remov")`, and the empty section's own lead
//     now reads "...so nothing here is recommended for removal", which contains
//     it. Deleting it as a prose collision would have retired a real guard on
//     the strength of a substring.
//  6. "every existing summary line survives" — "Outdated: 1", "Up-to-date: 3",
//     "Other: 2", "Total: 6". The per-STATUS run counters do not reach the new
//     payload at all: `func buildCompareReport` sums outdated + newer +
//     up-to-date into CompareRun.InBoth by decision, so the report states how
//     many packages exist in both repositories and no longer how the versions
//     divided. That is a deliberate narrowing of the RENDERED summary and not a
//     lost fact: the four counters are still computed and still asserted
//     directly on CompareReport by TestCompare, TestCompareNewerInLocal and
//     TestCompareWithAllResults in compare_test.go, by
//     TestCompareInterrupted..., by compare_nilmap_test.go, and by
//     TestSplitLeavesVerdicts' survival table in
//     compare_redundant_split_test.go. What no test now covers is that they
//     reach an operator — see the note in /tmp/047-triage.md, Group A.
//
// verdictReportFixture went with it. It was that test's fixture and nothing
// else read it, so keeping it would leave an unused builder for a report shape
// no assertion is taken over — which is how a fixture comes to be trusted as
// coverage it is not. compare_redundant_split_test.go keeps its own.

// lineContaining is deliberately GONE. It returned the first line of the report
// containing a package's name, which stopped being that package's row the moment
// a section note contained an ordinary English word that is also a fixture's
// package name ("...whether a copy holds work of ours" and app-misc/ours). Both
// callers now use tableRows (compare_redundant_split_test.go), which matches the
// first column of a table row and therefore cannot match prose.

// MERGE FRAGMENT — story 034, sub-task 8.1 (report every count with its denominator).
//
// Target file: internal/overlay/compare_report_verdict_test.go  (APPEND, package overlay)
// That is the compare RENDERER's test file — it already owns the report-shaped
// fixtures and the assertions on what FormatReport prints, which is exactly the
// surface this sub-task changes. Every symbol added here is prefixed
// `classification`, so nothing in it collides with `verdictReportFixture` and
// friends.
//
// Pinned contract:
//
//	func classificationLines(r CompareResult) []string        // per package
//	func runClassificationLines(rep *CompareReport) []string  // per run, with the denominator
//
// Two pure line builders, following the precedent
// overlay_compare_summary_test.go states for the summary: assert on the builder
// rather than on captured log output, because logger binds its io.Writer at
// first use and exposes no setter. Splitting the decision from the emission is
// cheaper than capturing a stream and leaves the emission trivial enough to
// read. FormatReport calls them; the tests below check both the lines and the
// fact that FormatReport shows them.
//
// The numbers in the fixtures — 31, 47, 22, 100 — are chosen so that no count is
// a SUBSTRING of another or of the total. With 3, 7, 2 and 12, an assertion that
// "the ours column does not mention 2" passes on a line reading "of 12" and
// fails on nothing.

const (
	classificationVersionMove  = 31
	classificationOurs         = 47
	classificationUnclassified = 22
	classificationTotal        = classificationVersionMove + classificationOurs + classificationUnclassified
)

// classificationResult is one package's classified difference.
func classificationResult(versionMove, ours, unclassified int, reduced bool) CompareResult {
	return CompareResult{
		Category:      "media-libs",
		Package:       "gst-plugins-qt6",
		LocalVersion:  "1.29.2",
		RemoteVersion: "1.26.11",
		Baseline:      Baseline{Repo: "gentoo", Version: "1.26.11", Path: "/x.ebuild", Distance: 1, Found: true},
		Classified: Classified{
			VersionMove:  versionMove,
			Ours:         ours,
			Unclassified: unclassified,
			Reduced:      reduced,
			Span:         1,
		},
	}
}

// classificationLineWith returns the first line mentioning `keyword`, and fails
// the test when none does — a missing line and an empty line are different
// failures and only one of them is this sub-task's.
func classificationLineWith(t *testing.T, lines []string, keyword string) string {
	t.Helper()
	for _, l := range lines {
		if strings.Contains(strings.ToLower(l), keyword) {
			return l
		}
	}
	t.Fatalf("no line mentions %q; the classification is reported without one of its three classes:\n%s", keyword, strings.Join(lines, "\n"))
	return ""
}

// TestClassificationCountsSumToTheTotal is R2.4 and R2.5 read together: the
// three counts and the total are one arithmetic, and a report that shows two of
// the three lets the third be inferred wrongly.
//
// _Requirements: R2, R2.4, R2.5_
func TestClassificationCountsSumToTheTotal(t *testing.T) {
	res := classificationResult(classificationVersionMove, classificationOurs, classificationUnclassified, true)
	lines := classificationLines(res)

	if len(lines) == 0 {
		t.Fatal("classificationLines produced nothing for a classified result; the per-package counts reach no report (R2.4)")
	}
	joined := strings.Join(lines, "\n")

	for label, want := range map[string]int{
		"version move": classificationVersionMove,
		"ours":         classificationOurs,
		"unclassified": classificationUnclassified,
		"total":        classificationTotal,
	} {
		if !strings.Contains(joined, strconv.Itoa(want)) {
			t.Errorf("the per-package classification never states the %s count (%d):\n%s", label, want, joined)
		}
	}

	sum := res.Classified.VersionMove + res.Classified.Ours + res.Classified.Unclassified
	if sum != classificationTotal {
		t.Fatalf("the fixture itself does not add up (%d != %d); every assertion here would be about the wrong arithmetic", sum, classificationTotal)
	}
}

// TestClassificationUnclassifiedIsInNeitherColumn is R2.3. A difference the
// reduction and the model both declined to attribute is unclassified, and
// pushing it into either bucket is the failure this requirement exists to
// prevent: "ours" would invite a realignment of something nobody read, and
// "version move" would subtract deliberate work as noise.
//
// _Requirements: R2, R2.3, R2.5_
func TestClassificationUnclassifiedIsInNeitherColumn(t *testing.T) {
	lines := classificationLines(classificationResult(classificationVersionMove, classificationOurs, classificationUnclassified, true))

	versionLine := classificationLineWith(t, lines, "version")
	oursLine := classificationLineWith(t, lines, "ours")
	unclassifiedLine := classificationLineWith(t, lines, "unclassified")

	if !strings.Contains(unclassifiedLine, strconv.Itoa(classificationUnclassified)) {
		t.Errorf("the unclassified line is %q, want it to state %d", unclassifiedLine, classificationUnclassified)
	}
	if strings.Contains(versionLine, strconv.Itoa(classificationUnclassified)) {
		t.Errorf("the version-move line is %q and carries the unclassified count %d; a difference nobody attributed must not be counted as version noise (R2.3)", versionLine, classificationUnclassified)
	}
	if strings.Contains(oursLine, strconv.Itoa(classificationUnclassified)) {
		t.Errorf("the ours line is %q and carries the unclassified count %d; a difference nobody attributed must not be counted as ours (R2.3)", oursLine, classificationUnclassified)
	}
}

// TestClassificationWithNoModelReportsEverythingUnclassified: with no reviewer,
// the deterministic reduction is all that ran. Whatever it could not subtract is
// unclassified — not "ours by elimination", which is the tempting shortcut and
// the one that would propose realigning 492 lines of deliberate slotting work.
//
// _Requirements: R2, R2.3, R2.5, R4.4_
func TestClassificationWithNoModelReportsEverythingUnclassified(t *testing.T) {
	// Reduced=false is D3's third preference: no usable third point, so nothing
	// was subtracted and no model spoke.
	res := classificationResult(0, 0, classificationTotal, false)
	lines := classificationLines(res)

	versionLine := classificationLineWith(t, lines, "version")
	oursLine := classificationLineWith(t, lines, "ours")
	unclassifiedLine := classificationLineWith(t, lines, "unclassified")

	if !strings.Contains(unclassifiedLine, strconv.Itoa(classificationTotal)) {
		t.Errorf("the unclassified line is %q, want all %d differences reported there", unclassifiedLine, classificationTotal)
	}
	for _, line := range []string{versionLine, oursLine} {
		if strings.Contains(line, strconv.Itoa(classificationTotal)) {
			t.Errorf("%q claims %d differences although nothing was attributed; with no model and no third point, 'ours by elimination' is a guess wearing a count", line, classificationTotal)
		}
	}
}

// TestRunClassificationStatesItsDenominator is the requirement's whole point. A
// share without its denominator — "63% attributed" — is unreadable: 63% of
// twelve differences in one package and 63% of forty thousand across the overlay
// are different claims, and a classification whose reach is invisible is
// indistinguishable from a guess.
//
// _Requirements: R2, R2.5_
func TestRunClassificationStatesItsDenominator(t *testing.T) {
	report := &CompareReport{
		TotalPackages:    2,
		ComparedPackages: 2,
		Results: []CompareResult{
			classificationResult(classificationVersionMove, classificationOurs, classificationUnclassified, true),
			classificationResult(0, 0, classificationTotal, false),
		},
	}

	lines := runClassificationLines(report)
	if len(lines) == 0 {
		t.Fatal("runClassificationLines produced nothing; the run-level share is the line 8.1 exists to add")
	}
	joined := strings.Join(lines, "\n")

	// The denominator: every difference the run examined, across both packages.
	wantExamined := 2 * classificationTotal
	if !strings.Contains(joined, strconv.Itoa(wantExamined)) {
		t.Errorf("the run-level lines never state how many differences were EXAMINED (%d):\n%s\na share without its denominator is the failure R2.5 exists to prevent", wantExamined, joined)
	}
	// The numerator that matters: the second package contributed its whole diff
	// to the unclassified column, so the run's unclassified share is large and
	// must be visible rather than averaged away.
	wantUnclassified := classificationUnclassified + classificationTotal
	if !strings.Contains(joined, strconv.Itoa(wantUnclassified)) {
		t.Errorf("the run-level lines never state the unclassified count (%d of %d):\n%s", wantUnclassified, wantExamined, joined)
	}
}

// TestClassificationRendersNothingAtItsZeroValue is R7.2 held at the renderer,
// and it is the reason these lines can be added to a shipped command at all: a
// plain `compare` fills no Classified, so these builders produce nothing and
// FormatReport prints exactly what it printed yesterday.
//
// Task 6.3's TestAnnotateBaselineZeroValueRendersNothing asserts the same
// property from the other end, over the whole rendering. This one is the local
// half: the builder itself is silent, so the silence does not depend on a caller
// remembering to check.
//
// _Requirements: R7.2, R2.5_
func TestClassificationRendersNothingAtItsZeroValue(t *testing.T) {
	if lines := classificationLines(CompareResult{Category: "media-libs", Package: "gst-plugins-qt6"}); len(lines) != 0 {
		t.Errorf("classificationLines produced %q for an unclassified-and-unexamined result; a run without --realign must render exactly what it renders today", lines)
	}
	empty := &CompareReport{TotalPackages: 1, ComparedPackages: 1, Results: []CompareResult{{
		Category: "media-libs", Package: "gst-plugins-qt6",
	}}}
	if lines := runClassificationLines(empty); len(lines) != 0 {
		t.Errorf("runClassificationLines produced %q for a report with no classification; the run-level share is emitted only for a review run", lines)
	}
}

// TestClassificationReachesTheRenderedReport closes the gap the two builders
// leave: a line builder nothing calls is a line nobody sees.
//
// # Story 047, sub-task 5.5 — it now asks the question of the FINDINGS
//
// It used to call FormatReport and look for the three counts in its text.
// Sub-task 4.2 deletes FormatReport, so the assertion is taken where the counts
// actually leave this package now: `func classificationFinding` in
// annotate_baseline.go builds a FindingClassification carrying Classified as
// three integers, EstablishFindings composes it into CompareReport.Findings,
// and `func comparePackageNotes` in cmd/bentoo turns every finding beyond a
// package's first into a note under that package's section. The counts are
// therefore READ as numbers rather than matched as digits in a sentence, which
// is the stronger form of the same claim — `strings.Contains(rendered, "22")`
// passed on any line that happened to contain a 2 and a 2.
//
// The per-package half is all this test can carry. The RUN-level block —
// runClassificationLines, the share with its denominator — has no consumer once
// FormatReport is gone: it is a summary and deliberately not a finding, so
// EstablishFindings does not compose it and compareRunNotes does not carry it.
// That gap is recorded rather than papered over; asserting the builder here
// would test a function nothing calls and read as coverage.
//
// _Requirements: R2, R2.4, R2.5, S047-R8.2_
func TestClassificationReachesTheRenderedReport(t *testing.T) {
	report := &CompareReport{
		TotalPackages:    1,
		ComparedPackages: 1,
		Results:          []CompareResult{classificationResult(classificationVersionMove, classificationOurs, classificationUnclassified, true)},
	}
	EstablishFindings(report)

	var found *Finding
	for i := range report.Findings {
		if report.Findings[i].Kind == FindingClassification {
			found = &report.Findings[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("no classification finding reached the report; the builders are written and called by nobody:\n%+v", report.Findings)
	}
	if found.Atom != "media-libs/gst-plugins-qt6" {
		t.Errorf("the classification finding names %q; a count with no package to attach it to cannot be shown under a row", found.Atom)
	}
	if got := found.Classified.VersionMove; got != classificationVersionMove {
		t.Errorf("VersionMove = %d, want %d", got, classificationVersionMove)
	}
	if got := found.Classified.Ours; got != classificationOurs {
		t.Errorf("Ours = %d, want %d", got, classificationOurs)
	}
	if got := found.Classified.Unclassified; got != classificationUnclassified {
		t.Errorf("Unclassified = %d, want %d", got, classificationUnclassified)
	}
	// The sentence the operator reads is still built from the same value, so
	// the finding and the line cannot drift apart.
	if lines := renderClassificationLines(*found); len(lines) == 0 {
		t.Error("the classification finding renders no line; the counts reach a consumer and no reader")
	}
}
