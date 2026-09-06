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
// NEITHER OF THOSE EXISTS ANY MORE. Story 047 moved the rendering out of this
// package: sub-task 4.2 deleted the renderer that called them, the run-level
// builder went with it, and the 4.2 addendum deleted the per-package one. The
// contract this file now holds is `func classificationFinding(r CompareResult)
// (Finding, bool)` — the same counts, the same sentence, as a value — and the
// count-with-its-denominator that reaches an operator is pinned by
// TestCompareRunClassificationShareReachesTheReport in cmd/bentoo.
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

// Story 047, sub-task 4.2 addendum — S047-R8.1: classificationLineWith stood
// here. It picked the first rendered line mentioning a class name, and every
// caller it had is migrated below off the rendered lines and onto the value the
// classification actually establishes. A helper for a rendering nothing produces
// is one more thing that reads as coverage and is not.

// TestClassificationCountsSumToTheTotal is R2.4 and R2.5 read together: the
// three counts and the total are one arithmetic, and a consumer given two of the
// three lets the third be inferred wrongly.
//
// Story 047, sub-task 4.2 addendum (S047-R8.1): it read the digits back out of
// the lines classificationLines built, and that builder lost its last production
// caller when 4.2 deleted renderBaselineFindings. The counts are now read as
// INTEGERS off the Classified the finding carries, which is the stronger form of
// the same claim — `strings.Contains(joined, "22")` passed on any text holding a
// 2 and a 2. The DENOMINATOR keeps one assertion on the sentence, because it is
// stated there and computable everywhere else, and R2.4 is that it be STATED;
// the constants are chosen so no count is a substring of another or of the total.
//
// _Requirements: R2, R2.4, R2.5, S047-R8.1_
func TestClassificationCountsSumToTheTotal(t *testing.T) {
	res := classificationResult(classificationVersionMove, classificationOurs, classificationUnclassified, true)

	if sum := res.Classified.VersionMove + res.Classified.Ours + res.Classified.Unclassified; sum != classificationTotal {
		t.Fatalf("the fixture itself does not add up (%d != %d); every assertion here would be about the wrong arithmetic", sum, classificationTotal)
	}

	finding, classified := classificationFinding(res)
	if !classified {
		t.Fatal("no classification was established for a classified result; the per-package counts reach no consumer (R2.4)")
	}

	for _, class := range []struct {
		label string
		got   int
		want  int
	}{
		{"version move", finding.Classified.VersionMove, classificationVersionMove},
		{"ours", finding.Classified.Ours, classificationOurs},
		{"unclassified", finding.Classified.Unclassified, classificationUnclassified},
	} {
		if class.got != class.want {
			t.Errorf("the classification states %d for the %s count, want %d", class.got, class.label, class.want)
		}
	}

	if total := classifiedTotal(finding.Classified); total != classificationTotal {
		t.Errorf("the denominator is %d, want %d; the three counts and the total they are a share of are one arithmetic (R2.4)", total, classificationTotal)
	}
	// And it has to be SAID. Three counts whose total is invisible let a reader
	// infer the missing one wrongly, which is the whole of R2.5 here.
	if !strings.Contains(finding.Detail, strconv.Itoa(classificationTotal)) {
		t.Errorf("the classification reads %q and never states the %d differences its counts are a share of (R2.5)", finding.Detail, classificationTotal)
	}
}

// TestClassificationUnclassifiedIsInNeitherColumn is R2.3. A difference the
// reduction and the model both declined to attribute is unclassified, and
// pushing it into either bucket is the failure this requirement exists to
// prevent: "ours" would invite a realignment of something nobody read, and
// "version move" would subtract deliberate work as noise.
//
// Story 047, sub-task 4.2 addendum (S047-R8.1): "in neither column" used to mean
// "the count is not a substring of the other two lines". The columns are three
// FIELDS, so the claim is now arithmetic on them — the two attributed classes
// hold the total minus the unclassified, and an unclassified difference is
// therefore counted once, in the third field and in neither of the other two.
// That cannot be satisfied by a rendering, and it cannot pass on a coincidence
// of digits.
//
// _Requirements: R2, R2.3, R2.5, S047-R8.1_
func TestClassificationUnclassifiedIsInNeitherColumn(t *testing.T) {
	finding, classified := classificationFinding(classificationResult(classificationVersionMove, classificationOurs, classificationUnclassified, true))
	if !classified {
		t.Fatal("no classification was established for a classified result; the per-package counts reach no consumer (R2.4)")
	}
	got := finding.Classified

	if got.Unclassified != classificationUnclassified {
		t.Errorf("the unclassified count is %d, want %d", got.Unclassified, classificationUnclassified)
	}
	if got.VersionMove != classificationVersionMove {
		t.Errorf("the version-move count is %d, want %d; if the %d differences nobody attributed have been folded in here, deliberate work is being subtracted as version noise (R2.3)",
			got.VersionMove, classificationVersionMove, classificationUnclassified)
	}
	if got.Ours != classificationOurs {
		t.Errorf("the ours count is %d, want %d; if the %d differences nobody attributed have been folded in here, the report invites a realignment of something nobody read (R2.3)",
			got.Ours, classificationOurs, classificationUnclassified)
	}
	if attributed := got.VersionMove + got.Ours; attributed != classificationTotal-classificationUnclassified {
		t.Errorf("the two attributed classes hold %d of %d differences while %d are unclassified; an unclassified difference is counted in the third class and in NEITHER of the other two (R2.3)",
			attributed, classificationTotal, classificationUnclassified)
	}
}

// TestClassificationWithNoModelReportsEverythingUnclassified: with no reviewer,
// the deterministic reduction is all that ran. Whatever it could not subtract is
// unclassified — not "ours by elimination", which is the tempting shortcut and
// the one that would propose realigning 492 lines of deliberate slotting work.
//
// Story 047, sub-task 4.2 addendum (S047-R8.1): asserted on the value rather
// than on the lines classificationLines built, for the reason above it. The
// reach keeps one assertion, because three zeroes with no reach beside them read
// as a package there was nothing to classify for rather than as one nothing
// could be attributed for — which is the distinction R2.5 exists to keep.
//
// _Requirements: R2, R2.3, R2.5, R4.4, S047-R8.1_
func TestClassificationWithNoModelReportsEverythingUnclassified(t *testing.T) {
	// Reduced=false is D3's third preference: no usable third point, so nothing
	// was subtracted and no model spoke.
	finding, classified := classificationFinding(classificationResult(0, 0, classificationTotal, false))
	if !classified {
		t.Fatal("no classification was established although the reduction ran; a package whose every difference is unclassified is still classified (R2.4)")
	}
	got := finding.Classified

	if got.Unclassified != classificationTotal {
		t.Errorf("%d differences are reported unclassified, want all %d of them", got.Unclassified, classificationTotal)
	}
	if got.VersionMove != 0 || got.Ours != 0 {
		t.Errorf("the classification attributes %d differences to a version move and %d to us although nothing was subtracted and no model spoke; 'ours by elimination' is a guess wearing a count",
			got.VersionMove, got.Ours)
	}
	if want := baselineReachProse(got); !strings.Contains(finding.Detail, want) {
		t.Errorf("the classification reads %q and does not state its reach %q; a total with no reach beside it cannot be told from one nobody looked for (R2.5)", finding.Detail, want)
	}
}

// TestClassificationRendersNothingAtItsZeroValue is R7.2 held at the builder,
// and it is the reason these lines can be added to a shipped command at all: a
// plain `compare` fills no Classified, so the builder produces nothing and the
// run reports exactly what it reported yesterday.
//
// Task 6.3's TestAnnotateBaselineZeroValueRendersNothing asserts the same
// property from the other end, over the whole rendering. This one is the local
// half: the builder itself is silent, so the silence does not depend on a caller
// remembering to check.
//
// Story 047, sub-task 4.2: the run-level half of this test asserted the same
// silence for the run-level line builder. Both the builder and the renderer that
// was its only caller are deleted, so a report with no classification now has no
// run-level line to be silent about. The per-package half below is the whole of
// what this package still builds.
//
// Story 047, sub-task 4.2 addendum (S047-R8.1): the per-package half asked
// classificationLines for an empty slice, and that builder went the same way.
// The silence is now asserted where it is DECIDED — classificationFinding
// returns false for a zero-value Classified — and again at the boundary, since
// a gate nobody consults is not a gate: EstablishFindings must compose no
// classification for a package nothing was classified for.
//
// _Requirements: R7.2, R2.5, S047-R8.1_
func TestClassificationEstablishesNothingAtItsZeroValue(t *testing.T) {
	unexamined := CompareResult{Category: "media-libs", Package: "gst-plugins-qt6"}

	if finding, classified := classificationFinding(unexamined); classified {
		t.Errorf("a classification was established for an unclassified-and-unexamined result: %+v; a run without --realign must report exactly what it reported yesterday", finding)
	}

	report := &CompareReport{TotalPackages: 1, ComparedPackages: 1, Results: []CompareResult{unexamined}}
	EstablishFindings(report)
	for _, f := range report.Findings {
		if f.Kind == FindingClassification {
			t.Errorf("a classification finding reached the report for a package nothing was classified for: %+v", f)
		}
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
// and `func comparePkgFacts` in cmd/bentoo carries every finding beyond a
// package's first on ComparePkg.FurtherFindings, which
// `func (r CompareRun) Sections` says under that package's own block. The counts are
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
	// The sentence a reader is given is built from the same value, so the counts
	// and the sentence cannot drift apart.
	//
	// Story 047, sub-task 4.2 addendum (S047-R8.1): this asked
	// renderClassificationLines, which had no production caller left after 4.2, for
	// a non-empty slice. Detail is the field that sentence is now rendered from —
	// by `func comparePkgFacts` in cmd/bentoo — so the claim is made there.
	if found.Detail == "" {
		t.Error("the classification finding carries no sentence; the counts reach a consumer and no reader")
	}
}
