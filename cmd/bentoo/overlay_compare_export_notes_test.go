package main

// Authored for story 047, sub-task 7.1 — S047-R1.3, S047-R6.3.
//
// # The defect this file exists for
//
// `func presentCompareReport` in overlay_compare_report.go built the terminal's
// sections and then appended the run's notes TO THOSE SECTIONS, while
// `func renderExport` in overlay_autoupdate_ui.go re-derives its own blocks from
// run.Payload. The two never met. Every run-level sentence and every finding
// beyond a package's first therefore reached the screen and no export, in any of
// the three formats — including the per-package findings that close issue #33,
// the prune advice, the classification share, the baseline coverage line and the
// realign notice.
//
// The export was strictly LESS complete than the terminal, which is the inverse
// of what S047-R1.3 ("--export SHALL write the complete report — every package,
// every reason in full") and S047-R6.3 ("carry every explanation in full") ask
// for, and it was invisible: the file is simply shorter, and nothing in it says
// what is missing.
//
// # Why nothing in the suite caught it
//
// Every test either asserted on the sections the terminal builds, or on the
// payload's fields, or on a golden rendered from a fixture that carried no note.
// Not one followed a sentence from the terminal THROUGH renderExport. That is
// the single assertion below, and it is a comparison of two populations rather
// than a check that a field exists: a field can exist and still be dropped by
// the path that serialises it.
//
// # Recorded evidence that it fails when its rule is broken (S047-R8.1)
//
// Mutation: the historical defect restored at the seam it lived on. Four lines
// at the top of `func renderExport` in overlay_autoupdate_ui.go put the export
// back on the payload shape it had before this sub-task — CompareRun.Notes
// cleared and every ComparePkg.FurtherFindings cleared — while the terminal
// path was left exactly as it is. All three arms failed, on both populations:
//
//	the markdown export dropped a sentence the terminal printed: "dev-libs/foo: inherit differs …"
//	the json export dropped a sentence the terminal printed: "1 of the 2 packages compared …"
//	the plain export dropped a sentence the terminal printed: "Realignment verdicts: 1 of the 2 …"
//
// The whole of ./cmd/bentoo was run against the mutation and THIS WAS THE ONLY
// TEST THAT FAILED, which is the claim above measured rather than asserted.
// Reverted, and the revert verified byte-identical by md5sum.
//
// # What this test does NOT catch, said plainly
//
// Its subject is the terminal's own notes, so a change that removes a sentence
// from BOTH paths leaves the two in agreement and passes here. That is the
// correct division and not a gap: what each note SAYS is pinned by
// compare_run_test.go and overlay_compare_run_summaries_test.go, and what this
// file exists for is the asymmetry those cannot see — one path saying more than
// the other.

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/obentoo/bentoolkit/internal/common/report"
	"github.com/obentoo/bentoolkit/internal/overlay"
)

// compareNotesFixture is a run with something to say on BOTH axes: a run-level
// sentence no row carries, and a package with a second finding its one-line
// reason cell has no room for.
//
// Both are built through `func buildCompareReport` and `func compareRunNotes`
// rather than written into a payload literal, so the test covers the adapter
// that fills the fields as well as the renderers that read them.
func compareNotesFixture(t *testing.T) report.Run {
	t.Helper()

	kept, findings := comparePkgWithFindings("dev-libs", "foo",
		"undeclared divergence — ours differs",
		"inherit differs from ::gentoo — gstreamer-meson")
	rep := &overlay.CompareReport{
		TotalPackages:    2,
		ComparedPackages: 2,
		// Two different run-level sentences, on two different conditions: one
		// counted straight off the report, one derived from the flags a
		// renderer never learns. A fixture firing only one of them would leave
		// half of `func compareRunNotes` untravelled.
		NoBaselineCount:  1,
		RealignAsked:     2,
		RealignNoVerdict: 1,
		Results:          []overlay.CompareResult{kept},
		Findings:         findings,
	}

	notes := compareRunNotes(rep, true, true, false)
	if len(notes) == 0 {
		t.Fatalf("the fixture produced no run-level note, so this test would assert over an empty population")
	}
	return buildCompareReport(rep, "gentoo", nil, notes...)
}

// flatten collapses every run of whitespace to a single space.
//
// A note is wrapped to the device by the plain writer and re-flowed by nothing
// in markdown, so a fixed sentence may be split across lines. Order and content
// survive wrapping; a byte offset does not, which is why the comparison is made
// on normalised text rather than on the raw output.
func flatten(s string) string { return strings.Join(strings.Fields(s), " ") }

// jsonStrings is every string value anywhere in a JSON document.
//
// The document is DECODED rather than searched as text: encoding/json escapes
// some characters on the way out, so a substring match against the raw bytes
// would be checking the encoder as well as the report, and could fail for a
// reason that has nothing to do with a missing note.
func jsonStrings(t *testing.T, data []byte) []string {
	t.Helper()

	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("the JSON export did not parse: %v\n%s", err, data)
	}

	var out []string
	var walk func(any)
	walk = func(v any) {
		switch value := v.(type) {
		case string:
			out = append(out, value)
		case []any:
			for _, item := range value {
				walk(item)
			}
		case map[string]any:
			for _, item := range value {
				walk(item)
			}
		}
	}
	walk(doc)
	return out
}

// compareRenderedLead is the sentence `func comparePkgNotes` in
// internal/common/report/compare_run.go writes above a block's further
// findings. It is spelled out here because that constant is unexported one
// package over, and exporting prose to save a line would publish one package's
// wording as another's API.
const compareRenderedLead = "Beside the reason on each row, the run established more about these packages:"

// jsonPayloadForm is the text a rendered note is expected to appear as in the
// JSON export, and whether the JSON is expected to carry it at all.
//
// The JSON export is the PAYLOAD serialised, not the section list, so the two
// documents state the same facts in different shapes and a verbatim comparison
// would be measuring the shape. Two translations, and both are narrow:
//
//   - A package's further finding is rendered as "<atom>: <finding>" because a
//     reader meeting prose under a table must be told which row it belongs to.
//     A JSON reader is not: the finding sits inside that package's own object,
//     beside its "package" key, so the prefix is the block's doing and the text
//     after it is the fact. The prefix is stripped only when it names a package
//     the document actually carries.
//   - The lead above them carries no fact at all — its whole job is to say that
//     the sentences under it belong to the rows above, which in JSON is stated
//     by nesting rather than by a sentence. It is the ONE thing excluded, it is
//     named as a constant so the exclusion cannot silently widen, and excluding
//     it costs the check nothing: the findings it introduces are still asserted
//     one by one, and a payload that dropped them fails here whether or not the
//     lead was skipped.
func jsonPayloadForm(note string, packages []string) (string, bool) {
	if note == compareRenderedLead {
		return "", false
	}
	for _, pkg := range packages {
		if strings.HasPrefix(note, pkg+": ") {
			return strings.TrimPrefix(note, pkg+": "), true
		}
	}
	return note, true
}

// TestCompareNotesReachEveryExport is the report's completeness comparison: a
// sentence the terminal prints beside its tables is a sentence every exported
// format carries (S047-R1.3, S047-R6.3).
//
// # The population is taken from the TERMINAL, not written out here
//
// Listing the expected sentences would pin this test to today's wording and let
// tomorrow's note be added, printed and dropped by the export with the suite
// green — which is precisely how the defect survived. The terminal's own blocks
// are the subject, so a note nobody thought to add here is still covered on the
// day it is written.
//
// # The terminal is asked WITHOUT --all, and the exports with it
//
// That is the real asymmetry `func exportContent` in overlay_autoupdate_check.go
// establishes: an export never holds a row back. Taking the smaller population
// as the subject is therefore the honest direction — everything the narrower
// render says must appear in the wider one, and a note that only the wider one
// carries is not a loss.
func TestCompareNotesReachEveryExport(t *testing.T) {
	run := compareNotesFixture(t)

	var said []string
	for _, section := range run.Sections(report.SectionOptions{}) {
		said = append(said, section.Notes...)
	}
	if len(said) < 2 {
		t.Fatalf("the terminal printed %d note(s): %q. This test needs both a run-level sentence and a package's further finding, or it is asserting over almost nothing", len(said), said)
	}

	var sawLead, sawPackage bool
	for _, note := range said {
		sawLead = sawLead || strings.Contains(note, "established more about these packages")
		sawPackage = sawPackage || strings.Contains(note, "gstreamer-meson")
	}
	if !sawLead || !sawPackage {
		t.Fatalf("the terminal's notes are %q, want the extra per-package finding among them — that is the population the export was dropping", said)
	}

	payload := compareComparePayload(t, run)
	var packages []string
	for _, list := range [][]report.ComparePkg{payload.Redundant, payload.NeedsRebase, payload.Keep, payload.Unknown} {
		for _, pkg := range list {
			packages = append(packages, pkg.Package)
		}
	}

	for _, tc := range []struct {
		name   string
		format exportFormat
	}{
		{"markdown", exportMarkdown},
		{"json", exportJSON},
		{"plain", exportPlain},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := renderExport(&buf, run, tc.format); err != nil {
				t.Fatalf("renderExport returned an error: %v", err)
			}

			var carries func(string) bool
			if tc.format == exportJSON {
				values := jsonStrings(t, buf.Bytes())
				carries = func(note string) bool {
					for _, value := range values {
						if strings.Contains(flatten(value), flatten(note)) {
							return true
						}
					}
					return false
				}
			} else {
				flat := flatten(buf.String())
				carries = func(note string) bool { return strings.Contains(flat, flatten(note)) }
			}

			for _, note := range said {
				if tc.format == exportJSON {
					want, expected := jsonPayloadForm(note, packages)
					if !expected {
						continue
					}
					note = want
				}
				if !carries(note) {
					t.Errorf("the %s export dropped a sentence the terminal printed: %q\n"+
						"    An export is the COMPLETE report, and a file shorter than the screen says nothing about\n"+
						"    what is missing from it (S047-R1.3, S047-R6.3). The usual cause is a note attached to the\n"+
						"    sections one path built rather than carried in the payload both paths read.\n"+
						"    Export:\n%s", tc.name, note, buf.String())
				}
			}
		})
	}
}
