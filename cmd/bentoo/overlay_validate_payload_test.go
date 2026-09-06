package main

// Authored for story 046, sub-task 16.3 — R4.4, R7.3, R7.4, R8.3.
//
// Written from the contract: design.md's Testing Strategy names the unit level
// it owes — "each payload's Sections derivation" — and names this payload as
// the fourth of four it did not have. `validatePayload` was proven only from
// the export path (overlay_validate_export_test.go, sub-task 10.1), which reads
// the sections through two renderers and a file. A test at that level cannot
// say whether a change moved the DERIVATION or the RENDERING, and the split
// between those two is the one D1 and D2 exist to draw. This file pins the
// derivation alone: no renderer, no file, no envelope.
//
// # What "the content is story 047's" means for the assertions below
//
// This payload's Sections declares an ABSENCE. Story 046's Out of Scope keeps
// `overlay validate`'s report CONTENT in story 047, so the block names no
// package, no gate and no finding, and carries no table. Every assertion here
// is therefore about the declaration and about the two properties that make it
// honest — it is the same for every run, and it says something. When 047
// replaces this block with the real ones, TestValidatePayloadSectionsDeclareNo
// Table and the identity tests are what it is expected to change, deliberately
// and in a reviewable diff. That is the point of pinning them: today's answer
// is a decision, and a decision that no test states can be undone by accident.
//
// # Wording is checked by ASPECT, never pinned
//
// omissionGaps (overlay_validate_export_test.go) reads R2.3's two obligations —
// state WHAT was omitted and WHY — against several spellings each, because
// which word an implementer reaches for is not a requirement. This file reuses
// it rather than asserting a sentence, so a rewrite that changes nothing an
// operator can see does not fail.
//
// # Red: by MUTATION, and the transcripts are below (R8.3)
//
// Every assertion in this file was green the day it was written: sub-task 10.1
// already fixed the behaviour, and what 16.3 closes is a COVERAGE gap, not a
// defect. A green test proves nothing about what it would catch, so R8.3 asks
// for the evidence it cannot supply — the same argument boundary_fields_test.go
// and contract_payload_matrix_test.go make for their own guards, and the reason
// the evidence is written HERE rather than pointed at: `git ls-files .epic/`
// returns nothing, so a pointer to the story's register is a pointer to nothing
// for anyone who cloned this.
//
// Measured on 2026-09-02 against HEAD 0fd2f2e. Seven mutations, each applied
// ALONE to cmd/bentoo/overlay_validate_report.go, run with
// `go test ./cmd/bentoo/ -run TestValidatePayloadSections -v`, and reverted
// immediately with `git checkout --` — the restore verified by md5sum against
// the pre-mutation file (760386107c220356125bf63da29850eb, unchanged after all
// seven). Every test below is killed by at least one of them; the PASSES named
// alongside are the point of the ones that survive.
//
// M1 — Sections returns nil. The pre-10.1 state, restored.
//
//	--- FAIL: TestValidatePayloadSectionsKeepTheirOrder
//	    the validate payload's sections are
//	      got: []
//	     want: ["Overlay Validation"]
//	--- FAIL: TestValidatePayloadSectionsLeadNamesItsSubject
//	    the payload produced no section at all
//	--- FAIL: TestValidatePayloadSectionsStateTheOmissionAndItsReason (all four runs)
//	    the declaration never names WHOSE content is absent ...
//	    the declaration never states that content is ABSENT ...
//	    the declaration never states WHY it is absent ...
//	--- FAIL: TestValidatePayloadSectionsStayNonVacuousOnTheEmptyRun
//	    a validation over a clean overlay produced NO section at all — an export
//	    in a section-reading syntax is then a file of zero bytes, and the operator
//	    cannot tell an empty report from a report that was never written (R2.3)
//	--- PASS: TestValidatePayloadSectionsAreTheSameForEveryRun
//
// That last line is why this file has two identity tests instead of one. "nil
// for every run" satisfies sameness PERFECTLY — all four runs agree, because
// none of them says anything — and it is the exact defect 10.1 fixed. Only the
// vacuity half catches it.
//
// M2 — Lead blanked to nil, the block otherwise untouched.
//
//	--- FAIL: TestValidatePayloadSectionsLeadNamesItsSubject
//	    the block "Overlay Validation" has no lead — the title alone names a
//	    subject and states nothing about it
//	--- FAIL: TestValidatePayloadSectionsStayNonVacuousOnTheEmptyRun
//	    the block "Overlay Validation" has no lead
//	--- PASS: TestValidatePayloadSectionsStateTheOmissionAndItsReason
//
// The pass is the second reason the lead is asserted separately: R2.3's two
// obligations survive in the Notes alone, so a check over the whole block would
// accept a document that puts its only sentence after the space where the report
// is not.
//
// M3 — the declaration DERIVED from the payload: the block returned only when
// the run holds a result, nil otherwise.
//
//	--- FAIL: TestValidatePayloadSectionsAreTheSameForEveryRun/a_validation_that_found_nothing
//	    a validation that found nothing describes itself differently from a run
//	    that found three results:
//	      got: []report.Section(nil)
//	     want: []report.Section{report.Section{Title:"Overlay Validation", ...}}
//	--- FAIL: TestValidatePayloadSectionsAreTheSameForEveryRun/a_selector_that_matched_nothing
//	--- FAIL: TestValidatePayloadSectionsStateTheOmissionAndItsReason (the two empty runs)
//	--- FAIL: TestValidatePayloadSectionsStayNonVacuousOnTheEmptyRun
//
// The two runs that break are the two with nothing in Results — the case least
// able to survive a derivation, and the one where a zero-byte file most
// resembles a clean result.
//
// M4 — a table added to the block: Headers{"Package", "Gate", "Outcome"}.
//
//	--- FAIL: TestValidatePayloadSectionsDeclareNoTable
//	    the block "Overlay Validation" declares the columns ["Package" "Gate"
//	    "Outcome"] — a column name is CONTENT, and this story's Out of Scope keeps
//	    overlay validate's content in story 047 (3 columns over 0 rows)
//
// M5 — a second block appended, titled "Findings", with a lead of its own.
//
//	--- FAIL: TestValidatePayloadSectionsKeepTheirOrder
//	    the validate payload's sections are
//	      got: ["Overlay Validation" "Findings"]
//	     want: ["Overlay Validation"]
//	--- FAIL: TestValidatePayloadSectionsStayNonVacuousOnTheEmptyRun
//	    the block "Findings" carries no note — the WHY of R2.3, and the route to
//	    the document that does hold the run, are both stated there
//	--- PASS: TestValidatePayloadSectionsDeclareNoTable
//
// M6 — the receiver named and the overlay path put in the lead.
//
//	--- FAIL: TestValidatePayloadSectionsNameNothingFromTheRun
//	    the block names "/var/db/repos/bentoo", which is this run's overlay — the
//	    declaration is a fact about the SPLIT between stories 046 and 047, not
//	    about the run, and the content it declares absent is not this story's to
//	    write (R7.3)
//	--- PASS: TestValidatePayloadSectionsAreTheSameForEveryRun
//
// The pass is the third hostile half, and it is not a technicality: the four
// fixtures share one overlay, so a block leaking THAT field stays identical
// across all of them. A leak is invisible to a sameness check by construction,
// which is why R7.3's boundary is asserted on its own.
//
// M7 — the block's title varied on opts.ShowAll.
//
//	--- FAIL: TestValidatePayloadSectionsIgnoreSectionOptions/ShowAll
//	    SectionOptions{ShowAll:true SkipPlan:false} changed the block: a
//	    declaration with no unit to list has nothing to shorten or expand
//	--- FAIL: TestValidatePayloadSectionsIgnoreSectionOptions/ShowAll_and_SkipPlan

import (
	"reflect"
	"strings"
	"testing"

	"github.com/obentoo/bentoolkit/internal/autoupdate/validate"
	"github.com/obentoo/bentoolkit/internal/common/report"
)

// validatePayload is a report.Payload, checked at compile time rather than
// inferred from the fact that validateEnvelope assigns it. The envelope's field
// is the interface, so a Sections that grew a parameter or moved to a pointer
// receiver would be caught here with the reason attached, instead of in a type
// error thirty lines into a call chain.
//
// It is also the reason this type belongs in the render package's payload
// sweeps (R7.4): a payload no guard inspects is a payload the contract is not
// checked over.
var _ report.Payload = validatePayload{}

// validatePayloadFor wraps one validation run in the payload position, exactly
// as validateEnvelope does.
func validatePayloadFor(rep validate.Report) validatePayload {
	return validatePayload{Report: rep}
}

// emptyValidateReport is a run over a clean overlay: it looked, and there was
// nothing to say about anything it found.
//
// It is the fixture a derivation falls silent on, which is why it appears in
// every test below rather than only in the one named after it.
func emptyValidateReport() validate.Report {
	return validate.Report{Overlay: "/var/db/repos/bentoo"}
}

// unmatchedValidateReport is the third shape: the selector named something the
// overlay does not hold, so the run evaluated no ebuild AND has a reason for it.
// It is neither the empty run nor the populated one, and a block that agreed
// with those two could still disagree with this.
func unmatchedValidateReport() validate.Report {
	return validate.Report{Overlay: "/var/db/repos/bentoo", UnmatchedSelector: "app-misc/nosuch"}
}

// cleanValidateReport is a run that evaluated an ebuild and every gate passed —
// the case whose report an operator is most likely to mistake for "we did not
// look" if the document says nothing.
func cleanValidateReport() validate.Report {
	rep := mixedReport()
	rep.Results = rep.Results[1:2]
	return rep
}

// validateSectionTitles is the block titles in the order they are meant to be
// read. Order is asserted as a SEQUENCE, never as a set: blocks that say the
// same things in a different order answer a different question.
func validateSectionTitles(blocks []report.Section) []string {
	titles := make([]string, 0, len(blocks))
	for _, b := range blocks {
		titles = append(titles, b.Title)
	}
	return titles
}

// validateSectionSaid is everything a reader of these blocks is shown, in the
// order they are shown it — titles, leads, table headers, cells, row details and
// notes. Nothing a block carries is left out, so a check for a leaked package
// name cannot pass because the name leaked into a field this helper forgot.
func validateSectionSaid(blocks []report.Section) string {
	var said []string
	for _, b := range blocks {
		said = append(said, b.Title)
		said = append(said, b.Lead...)
		said = append(said, b.Rows.Headers...)
		for _, row := range b.Rows.Rows {
			said = append(said, row.Cells...)
			said = append(said, row.Detail)
		}
		said = append(said, b.Notes...)
	}
	return strings.Join(said, "\n")
}

// TestValidatePayloadSectionsKeepTheirOrder pins what the payload produces: one
// block, named, and that one is the declaration of an absence.
//
// The COUNT is part of the assertion. A second block appearing here would be
// content — this story ships none, and story 047 replacing this block should
// have to change this line to do it.
func TestValidatePayloadSectionsKeepTheirOrder(t *testing.T) {
	got := validateSectionTitles(validatePayloadFor(mixedReport()).Sections(report.SectionOptions{}))
	want := []string{"Overlay Validation"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("the validate payload's sections are\n  got: %q\n want: %q", got, want)
	}
}

// TestValidatePayloadSectionsLeadNamesItsSubject pins the lead specifically —
// the sentences a reader meets before anything else.
//
// A block whose declaration lived only in its Notes would pass a check over the
// whole block and still put the one thing this document has to say after the
// space where the report is not. The lead is where a reader looks for what they
// are about to read, so the subject is asserted THERE, while R2.3's two
// obligations are checked over the block as a whole below.
func TestValidatePayloadSectionsLeadNamesItsSubject(t *testing.T) {
	blocks := validatePayloadFor(mixedReport()).Sections(report.SectionOptions{})
	if len(blocks) == 0 {
		t.Fatal("the payload produced no section at all")
	}

	block := blocks[0]
	if len(block.Lead) == 0 {
		t.Fatalf("the block %q has no lead — the title alone names a subject and states nothing about it", block.Title)
	}

	lead := strings.ToLower(strings.Join(block.Lead, "\n"))
	if !strings.Contains(lead, "validat") {
		t.Errorf("the lead never names whose report is absent, so the reader is left guessing which one they are not holding:\n%s", strings.Join(block.Lead, "\n"))
	}
}

// TestValidatePayloadSectionsStateTheOmissionAndItsReason is R2.3 at the level
// the requirement is written for: what was omitted, and why.
//
// It reuses omissionGaps rather than restating the aspects, so the export test
// and this one cannot drift into two readings of one requirement — and it runs
// the check over the SECTIONS instead of over a rendered file, which is the
// difference this file exists to make: a failure here is the derivation's, not
// a renderer's.
func TestValidatePayloadSectionsStateTheOmissionAndItsReason(t *testing.T) {
	for name, rep := range map[string]validate.Report{
		"a validation that found nothing":   emptyValidateReport(),
		"a selector that matched nothing":   unmatchedValidateReport(),
		"a validation whose gates all pass": cleanValidateReport(),
		"a validation with a failed gate":   mixedReport(),
	} {
		t.Run(name, func(t *testing.T) {
			said := validateSectionSaid(validatePayloadFor(rep).Sections(report.SectionOptions{}))
			for _, gap := range omissionGaps(said) {
				t.Errorf("the declaration %s\n%s", gap, said)
			}
		})
	}
}

// TestValidatePayloadSectionsDeclareNoTable pins the column names this payload's
// tables declare: none, because it declares an absence and a column name is
// content.
//
// Columns() is asserted rather than len(Headers) because a table can carry rows
// wider than its header, and a row that appeared here would be content arriving
// without a column name to announce it.
func TestValidatePayloadSectionsDeclareNoTable(t *testing.T) {
	for _, block := range validatePayloadFor(mixedReport()).Sections(report.SectionOptions{}) {
		if columns := block.Rows.Columns(); columns != 0 {
			t.Errorf("the block %q declares the columns %q — a column name is CONTENT, and this story's Out of Scope keeps overlay validate's content in story 047 (%d columns over %d rows)",
				block.Title, block.Rows.Headers, columns, len(block.Rows.Rows))
		}
	}
}

// TestValidatePayloadSectionsAreTheSameForEveryRun is the first hostile half of
// the identity rule this block rests on, and the one that fires WRONGLY when the
// declaration is derived: four runs that found four different things, one of
// which found nothing at all, must describe themselves identically.
//
// The rule is not "the empty run gets a block too". It is that the block states
// a fact about THE SPLIT between story 046 and story 047, which is equally true
// of a run that failed three gates and a run over a clean overlay. A block
// derived from the payload instead would fall silent exactly on the empty run —
// the case least able to survive it, and the one where a zero-byte file most
// resembles a clean result.
//
// Compared with DeepEqual over the whole block rather than over titles: a lead
// or a note that varied with the run would be the same defect one field lower.
func TestValidatePayloadSectionsAreTheSameForEveryRun(t *testing.T) {
	want := validatePayloadFor(mixedReport()).Sections(report.SectionOptions{})

	for name, rep := range map[string]validate.Report{
		"a validation that found nothing":   emptyValidateReport(),
		"a selector that matched nothing":   unmatchedValidateReport(),
		"a validation whose gates all pass": cleanValidateReport(),
	} {
		t.Run(name, func(t *testing.T) {
			got := validatePayloadFor(rep).Sections(report.SectionOptions{})
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s describes itself differently from a run that found three results:\n  got: %#v\n want: %#v", name, got, want)
			}
		})
	}
}

// TestValidatePayloadSectionsStayNonVacuousOnTheEmptyRun is the converse hostile
// half, and without it the test above is worth nothing: returning nil for every
// run satisfies identity PERFECTLY while stating nothing, and that is not a
// hypothetical — it is what this payload did before sub-task 10.1, and it left
// `overlay validate --export=report.md` writing a file of zero bytes.
//
// So sameness is checked here against the one run that can be mistaken for a
// failure to look, and it is checked for CONTENT: a title, a lead, a note, and
// R2.3's two obligations met in what they say.
func TestValidatePayloadSectionsStayNonVacuousOnTheEmptyRun(t *testing.T) {
	blocks := validatePayloadFor(emptyValidateReport()).Sections(report.SectionOptions{})
	if len(blocks) == 0 {
		t.Fatal("a validation over a clean overlay produced NO section at all — an export in a section-reading syntax is then a file of zero bytes, and the operator cannot tell an empty report from a report that was never written (R2.3)")
	}

	for _, block := range blocks {
		if strings.TrimSpace(block.Title) == "" {
			t.Error("a block with no title cannot be found by a reader scanning the document")
		}
		if len(block.Lead) == 0 {
			t.Errorf("the block %q has no lead", block.Title)
		}
		if len(block.Notes) == 0 {
			t.Errorf("the block %q carries no note — the WHY of R2.3, and the route to the document that does hold the run, are both stated there", block.Title)
		}
	}

	said := validateSectionSaid(blocks)
	for _, gap := range omissionGaps(said) {
		t.Errorf("the empty run's declaration %s\n%s", gap, said)
	}
}

// TestValidatePayloadSectionsNameNothingFromTheRun is the third hostile: a
// declaration that agreed with itself across four runs could still be leaking
// this run's facts into the sentence, and the leak would be invisible to a
// comparison that only asked whether the blocks were equal.
//
// It is R7.3's boundary read at the payload: `validatePayload` EMBEDS
// validate.Report, a producer type, so every field of a validation run is one
// selector away from this method. Naming a package here would be story 047's
// content arriving through the back door, un-designed, in a block an operator
// has already learned to read as "the report is elsewhere".
//
// Versions are deliberately not swept — "1.0" is a substring of too much prose
// to be evidence of anything. Packages, overlay paths, sources, gate reasons and
// finding details are the values that could only have come from the run.
func TestValidatePayloadSectionsNameNothingFromTheRun(t *testing.T) {
	rep := mixedReport()
	said := strings.ToLower(validateSectionSaid(validatePayloadFor(rep).Sections(report.SectionOptions{})))

	type leak struct {
		what  string
		value string
	}

	leaks := []leak{{"this run's overlay", rep.Overlay}}

	for _, result := range rep.Results {
		leaks = append(leaks, leak{"a package this run evaluated", result.Package})
		for _, source := range result.Sources {
			leaks = append(leaks, leak{"a source this run read", source})
		}
		for _, g := range result.Gates {
			if g.Reason != "" {
				leaks = append(leaks, leak{"a gate's reason from this run", g.Reason})
			}
			for _, f := range g.Findings {
				leaks = append(leaks, leak{"a finding from this run", f.Detail})
			}
		}
	}

	for _, l := range leaks {
		if l.value == "" {
			continue
		}
		if strings.Contains(said, strings.ToLower(l.value)) {
			t.Errorf("the block names %q, which is %s — the declaration is a fact about the SPLIT between stories 046 and 047, not about the run, and the content it declares absent is not this story's to write (R7.3)", l.value, l.what)
		}
	}
}

// TestValidatePayloadSectionsIgnoreSectionOptions pins the unnamed parameter as
// a decision rather than an oversight. SectionOptions says whether a report
// lists every unit or counts them, and a declaration with no unit to list has
// nothing to shorten or expand — so all four combinations must produce the same
// block.
//
// SkipPlan is included even though this payload has no plan: an option that
// silently began removing a block here would remove the only block there is.
func TestValidatePayloadSectionsIgnoreSectionOptions(t *testing.T) {
	payload := validatePayloadFor(mixedReport())
	want := payload.Sections(report.SectionOptions{})

	for name, opts := range map[string]report.SectionOptions{
		"ShowAll":              {ShowAll: true},
		"SkipPlan":             {SkipPlan: true},
		"ShowAll and SkipPlan": {ShowAll: true, SkipPlan: true},
	} {
		t.Run(name, func(t *testing.T) {
			if got := payload.Sections(opts); !reflect.DeepEqual(got, want) {
				t.Errorf("SectionOptions%+v changed the block: a declaration with no unit to list has nothing to shorten or expand\n  got: %#v\n want: %#v", opts, got, want)
			}
		})
	}
}
