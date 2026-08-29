package overlay

// Authored for story 046, sub-task 7.2 — R5.1, R5.2.
//
// Written from the contract: R5.1 asks that a finding established for the
// operator be returned to the caller, and R5.2 that it then render in every
// mode and reach the export "without the library having chosen how it looks".
// design.md D7 counts what this file is moving: the 16 output.* calls in
// annotate_baseline.go.
//
// # What is actually wrong today, and what this therefore asserts
//
// annotate_baseline.go does not write to stdout — it builds strings with
// output.Sprintf and hands them back. So "nothing was printed" is true before
// this sub-task starts, and a test that asked only that would pass over the
// defect untouched. What is wrong is that the COLOUR is chosen inside the
// library: by the time the caller holds the sentence, the appearance is
// decided, the identifier is embedded in prose, and the finding can be printed
// to a terminal and to nothing else — not to JSON, not to Markdown, not to a
// count.
//
// So the assertions are: the findings are VALUES with their identifiers in
// fields, they carry no escape sequence, and the annotation stays silent.
//
// The contract is 7.1's, unchanged: type Finding{Atom, Detail} reached through
// CompareReport.Findings. Baseline annotation appends to the same list rather
// than opening a second one — two lists of findings would be two orders, and a
// renderer would have to invent which comes first.
//
// Red on arrival: CompareReport has no Findings.
//
// Borrowed, never re-declared: annotateFixtureTrees, annotateReviewOpts and
// annotateCompare (annotate_baseline_test.go), captureOverlayStdout (the helper
// authored with 7.1, capture_stdout_findings_test.go — materialize that file
// first).

import (
	"strings"
	"testing"
)

// TestAnnotateBaselineReturnsItsFindings pins R5.1 over a run that had
// something to find: the overlay ebuild and the ::gentoo one differ on three
// axes and the overlay side carries a divergence declaration, so the annotation
// establishes several facts about one package.
func TestAnnotateBaselineReturnsItsFindings(t *testing.T) {
	overlayRoot, prov, pkgs := annotateFixtureTrees(t, map[string]bool{"media-libs/gst-plugins-qt6": true})
	opts := annotateReviewOpts(overlayRoot)
	report := annotateCompare(t, pkgs, prov, opts)

	AnnotateBaseline(report, prov, opts)

	if len(report.Findings) == 0 {
		t.Fatal("the baseline annotation returned no finding — everything it established stayed inside a formatted sentence (R5.1)")
	}

	compared := map[string]bool{}
	for _, result := range report.Results {
		compared[result.Category+"/"+result.Package] = true
	}

	for _, finding := range report.Findings {
		if strings.TrimSpace(finding.Atom) == "" {
			t.Errorf("a finding carries no atom: %+v — an identifier inside prose cannot be counted, filtered or exported (R5.1)", finding)
			continue
		}
		if !compared[finding.Atom] {
			t.Errorf("finding %+v names %q, which is not one of the packages reviewed: %v", finding, finding.Atom, compared)
		}
	}
}

// TestBaselineFindingsChooseNoAppearance pins R5.2. A finding that arrives with
// its colour already chosen can be printed to a terminal and to nothing else —
// which is exactly the three renderings and the export this story exists to
// make possible.
func TestBaselineFindingsChooseNoAppearance(t *testing.T) {
	overlayRoot, prov, pkgs := annotateFixtureTrees(t, map[string]bool{"media-libs/gst-plugins-qt6": true})
	opts := annotateReviewOpts(overlayRoot)
	report := annotateCompare(t, pkgs, prov, opts)

	AnnotateBaseline(report, prov, opts)

	for _, finding := range report.Findings {
		if strings.ContainsRune(finding.Detail, 0x1b) {
			t.Errorf("finding %q carries an escape sequence in its detail: %q — the library has chosen how it looks (R5.2)", finding.Atom, finding.Detail)
		}
		if strings.ContainsRune(finding.Atom, 0x1b) {
			t.Errorf("finding atom %q carries an escape sequence — an identifier is a fact, not a rendering", finding.Atom)
		}
	}
}

// TestAnnotationWritesNothingToStdout is the regression half: the day someone
// reaches for output.Info.Printf in this file again, this fails without anyone
// having to read the diff.
func TestAnnotationWritesNothingToStdout(t *testing.T) {
	overlayRoot, prov, pkgs := annotateFixtureTrees(t, map[string]bool{"media-libs/gst-plugins-qt6": true})
	opts := annotateReviewOpts(overlayRoot)
	report := annotateCompare(t, pkgs, prov, opts)

	out := captureOverlayStdout(t, func() { AnnotateBaseline(report, prov, opts) })

	if strings.TrimSpace(out) != "" {
		t.Errorf("the baseline annotation wrote to stdout (R5.1):\n%s", out)
	}
	if len(report.Findings) == 0 {
		t.Fatal("the annotation established nothing — silence with no findings is not the requirement, it is the same loss by another route")
	}
}

// TestUnexaminedBaselineIsAFindingToo is the hostile half of the same rule, and
// the one a suite usually leaves out.
//
// "We could not look" and "we looked and everything matched" produce the SAME
// output if the run-level outcome is not itself a finding: no per-package rows
// either way. A report that renders the first as the second tells the operator
// that nothing differs from ::gentoo when nothing was compared against it —
// which is the failure story 034 added BaselineSkipped for, and which this
// story must not lose while moving the sentence out of the library.
func TestUnexaminedBaselineIsAFindingToo(t *testing.T) {
	// A provider with no ::gentoo tree behind it at all: baselineTreeOf fails,
	// so nothing is examined.
	prov := &fakeProvider{versions: map[string][]string{}}
	report := &CompareReport{
		Results: []CompareResult{
			{Category: "media-libs", Package: "gst-plugins-qt6", LocalVersion: "1.29.2"},
		},
	}

	AnnotateBaseline(report, prov, CompareOptions{})

	if report.BaselineSkipped == "" {
		t.Fatal("the run-level skip was not recorded at all — the review reports as silence that it never ran")
	}
	if len(report.Findings) == 0 {
		t.Fatal("a review that examined NOTHING returned no finding, exactly as a review that examined everything and found nothing would — the two are indistinguishable to every renderer and to the export (R5.1)")
	}
	for _, finding := range report.Findings {
		if strings.ContainsRune(finding.Detail, 0x1b) {
			t.Errorf("the skip finding carries an escape sequence: %q (R5.2)", finding.Detail)
		}
	}
}
