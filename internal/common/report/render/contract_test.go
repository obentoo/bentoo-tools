package render

// Authored for story 046, sub-task 6.3 — R7.4.
//
// Written from the contract: story.md's Success Metrics state the claim this
// file checks — "Renderer files edited when a command is added: 0. Adding
// snapshot run after overlay manifest must not modify render; if it does, D1's
// contract did not hold and the design is revisited before story 047 depends on
// it." design.md's Components block says the same as a rule: render's
// dependencies are "report, lipgloss, bubbletea. It imports no producer."
//
// RED ON ARRIVAL, and not by mutation: every renderer in this package takes
// report.Report today — text.go names it eight times, fullscreen.go four,
// json.go twice, inline.go once. report.Report IS the payload under its old
// name (D1: "report.Report becomes report.AutoupdateCheck"), so a renderer
// holding it is a renderer that knows a payload. The guard measures the
// distance still to travel, and it goes green exactly when task 2 has finished
// moving the renderers onto sections.
//
// The sweep skips _test.go files. A test may name a payload — the golden files
// for the manifest and snapshot reports have to build one — and this file names
// every one of them itself, in the list below.
//
// # Evidence that it still fails, taken after task 2 turned it green (R8.3)
//
// The paragraph above is true of the day this file was written and stops being
// checkable the moment the guard passes: a green test proves nothing about what
// it would catch. R8.3 asks for the evidence a green guard cannot supply, and
// the story artifacts holding it (.draft/red-evidence.yaml) are not committed —
// `git ls-files .epic/` returns nothing — so a pointer to that file is a pointer
// to nothing for anyone who cloned this. It is written down here instead, the
// same way boundary_fields_test.go writes down its own.
//
// Measured on 2026-08-29, against sub-task 6.3. One mutation, applied on its
// own, run, and reverted immediately, with text.go restored byte for byte and
// the restore verified by `git diff --stat` producing no output.
//
// Mutation: a function taking a payload was added to text.go, above Plain —
//
//	func mutationProbe(_ report.ManifestRun) {}
//
// Observed:
//
//	--- FAIL: TestNoPayloadReference (0.00s)
//	    contract_test.go:105: text.go:126 names the payload type report.ManifestRun
//	    — a renderer that knows a payload is a renderer every new command has to
//	    edit (R7.4).
//	            Remedy: take []report.Section (or report.Run, for the export) and
//	            let the payload say how it describes itself.
//
// The message names the file, the line and the remedy, which is what makes the
// failure actionable by someone who did not write this file.
//
// # The fourth payload's entry, and evidence that it fires (sub-task 16.3)
//
// payloadTypes listed four names and none of them was `overlay validate`'s. The
// list read as though it covered every payload and covered three, which is the
// hole 16.3 closes; the type doc below says why the entry names validate.Report
// rather than validatePayload, and why the qualifier had to become part of the
// match.
//
// An added entry is a new hand-written line in a guard, so it owes the same
// evidence the guard itself owes. Measured on 2026-09-02 against HEAD 0fd2f2e.
// Three mutations, each applied ALONE to text.go, run, and reverted immediately
// with `git checkout --` — every restore verified by md5sum
// (b9a53d3e29b634d49989a1f5c8619d0b, unchanged after all three) and by
// `git diff --stat` producing no output.
//
// M1 — the fourth payload's domain half named by a renderer. The import and a
// function taking it were added above Plain, the same shape 6.3's mutation used:
//
//	func mutationProbe(_ validate.Report) {}
//
//	--- FAIL: TestNoPayloadReference (0.00s)
//	    contract_test.go: text.go:126 names the payload type validate.Report
//	    — a renderer that knows a payload is a renderer every new command has
//	    to edit …
//	--- FAIL: TestRenderImportsNoProducer (0.00s)
//	    contract_test.go: text.go imports the producer
//	    ".../internal/autoupdate/validate" — render turns sections into text
//	    and knows nothing about what made them …
//
// Both fail, and that is not redundancy: the import guard names the IMPORT and
// its rule, this one names the REFERENCE and the payload's remedy. Before this
// entry existed only the first fired, so the one diagnostic an implementer got
// pointed at the dependency rather than at the payload they were reaching for.
//
// M2 — a renderer SWITCHING on it rather than naming it, which is the failure
// the case-clause sweep exists for:
//
//	switch p.(type) {
//	case validate.Report:
//
//	--- FAIL: TestNoRendererSwitchesOnAPayloadType (0.00s)
//	    contract_payload_matrix_test.go: text.go:128 switches on
//	    validate.Report — a renderer deciding by payload type is edited by
//	    every command added after it …
//
// M3 — the regression half, because 16.3 restructured this list from bare names
// to qualified ones and a restructure that quietly stopped matching would be
// the same hole one layer down. Case clauses on report.ManifestRun and on
// report.Payload:
//
//	--- FAIL: TestNoRendererSwitchesOnAPayloadType (0.00s)
//	    text.go:127 switches on report.ManifestRun …
//	    text.go:129 switches on report.Payload …
//	--- FAIL: TestNoPayloadReference (0.00s)
//	    text.go:127 names the payload type report.ManifestRun …
//
// TestNoPayloadReference reports the ManifestRun case and NOT the Payload one,
// which is the scoping working: report.Payload is forbidden only where a
// renderer SWITCHES on it, because naming the interface is what passing a
// payload looks like.
//
// The GUARD files' own line numbers are omitted from all three transcripts, for
// the reason contract_payload_matrix_test.go's header sets out at length: a
// header that records a coordinate into a guard file is inside the file it
// points at, so writing the number down moves the line it names. The 6.3
// transcript higher up still says contract_test.go:105, which has not been that
// assertion for many edits, and that is the measurement rather than an
// anecdote. text.go's numbers ARE kept: they name the probe each mutation
// inserted, which exists only in the mutated file anyway. The verbatim numbers
// the runs printed are in .draft/red-evidence.yaml, a file that does not move
// what it describes.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// payloadTypes are the domain halves a renderer must not know, written as the
// SELECTOR a renderer would have to spell to name one — package AND type, not a
// bare type name.
//
// The five payloads this CLI ships, and the entry each contributes:
//
//	autoupdate.check   report.AutoupdateCheck
//	overlay.manifest   report.ManifestRun
//	snapshot.run       report.SnapshotRun
//	overlay.validate   validate.Report
//	overlay.compare    report.CompareRun
//
// report.Report is the sixth entry and not a sixth payload: it is the first one
// under its previous name. D1 renames it to AutoupdateCheck and moves it into
// the payload position, so a renderer still holding a Report is holding a
// payload the rename has not caught up with yet.
//
// # Why these are qualified, as of sub-task 16.3
//
// This was a list of bare names, matched against a `report.` qualifier
// hardcoded in the sweep below, and that shape could not spell the fourth
// payload at all: `overlay validate`'s domain half is `validate.Report`, which
// is not in the report package. The list therefore covered three payloads while
// reading as though it covered four — the defect 16.3 closes, and the reason
// the count is now asserted (TestNoRendererSwitchesOnAPayloadType) rather than
// left to be read off the literal.
//
// # Why the fourth entry names the PRODUCER type and not the payload
//
// The payload is `validatePayload`, and it lives in `package main` under
// cmd/bentoo — it embeds validate.Report, and internal/common/report may not
// import internal/autoupdate (boundary_test.go's forbiddenImports), so the type
// that puts a validation run in the payload position has to sit at the adapter.
// No Go file can import main, so no renderer can spell `validatePayload`, and
// an entry for it could never fire. A guard that cannot fail is precisely what
// S046-R8.3 refuses, so it is not listed.
//
// What a renderer CAN reach is the type `validatePayload` embeds, and that is
// the same payload's reachable half: a renderer holding a validate.Report is a
// renderer that knows what a validation run found, and every field of one is a
// selector away from it. TestRenderImportsNoProducer catches the import that
// would be needed; this catches the reference, with the payload's own remedy
// attached rather than the import rule's.
//
// The paragraph above used to close by predicting the next entry: "When story
// 047 gives the validate run a model of its own in the report package, the
// entry to ADD here is `report.ValidateRun`". Story 047 gave `overlay validate`
// no model. It added a FIFTH KIND instead — `overlay.compare`, whose payload is
// report.CompareRun, declared in the report package and listed below — and this
// list was widened for that rather than for the name it had been told to
// expect. run.go carries the correction in full, on KindOverlayValidate; the
// entry naming validate.Report stays exactly as it was, because that payload is
// still in `package main` and still unspellable by any renderer.
//
// The point the prediction was making survives it, and is worth restating: the
// name that belongs here is whatever a renderer could actually SPELL. It is the
// payload type when the payload is in the report package (report.CompareRun),
// and the reachable half when it is not (validate.Report). Naming the wrong one
// is how this list reached four audits covering three payloads.
// It is not derivable from everyKind and everyKind is not derivable from it —
// payloadKindCount's doc, in contract_payload_matrix_test.go, says why, and the
// count both lists are censused against is declared there.
var payloadTypes = []string{
	"report.Report",
	"report.AutoupdateCheck",
	"report.ManifestRun",
	"report.SnapshotRun",
	"validate.Report",
	// The fifth payload, added by story 047's sub-task 2.4. Like the three
	// report.* entries above it, and unlike validate.Report, it names the
	// payload type ITSELF: CompareRun is declared in the report package, so a
	// renderer really could spell it and this entry really can fire.
	"report.CompareRun",
}

// forbiddenPayloadSelectors is the set the two sweeps match against: every
// entry in payloadTypes, plus whatever the caller adds for its own rule.
//
// It exists because the two guards kept two literals of the same list, and the
// two drifted — the case-clause sweep held four names while the matrix held
// three kinds, "and they are not the same four". One list, read twice, cannot
// do that.
func forbiddenPayloadSelectors(extra ...string) map[string]bool {
	set := make(map[string]bool, len(payloadTypes)+len(extra))
	for _, name := range payloadTypes {
		set[name] = true
	}
	for _, name := range extra {
		set[name] = true
	}
	return set
}

// renderSourceFiles lists this package's non-test Go files, so a sweep that
// found none fails loudly instead of passing vacuously.
func renderSourceFiles(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading the package directory: %v", err)
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.Join(".", name))
	}

	if len(files) == 0 {
		t.Fatal("found no non-test Go file — the sweep would pass without inspecting anything")
	}
	return files
}

// TestNoPayloadReference is the claim, checked rather than assumed: nothing
// under render names a payload type, so adding a fifth kind is a new file in
// the report package and not an edit here.
//
// It reads the SELECTOR — report.X — rather than searching for the bare word,
// because "Report" appears in prose, in package names and in the word
// "Reporter" all over this repository, and a substring sweep would report
// violations that are sentences.
func TestNoPayloadReference(t *testing.T) {
	forbidden := forbiddenPayloadSelectors()

	fset := token.NewFileSet()

	for _, path := range renderSourceFiles(t) {
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}

		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}
			// The qualifier is part of the match rather than hardcoded to
			// `report`: the fourth payload's reachable half is
			// validate.Report, and a sweep that only looked at one package
			// could not see it (16.3).
			qualified := pkg.Name + "." + selector.Sel.Name
			if forbidden[qualified] {
				position := fset.Position(selector.Pos())
				t.Errorf("%s:%d names the payload type %s — a renderer that knows a payload is a renderer every new command has to edit (R7.4).\n"+
					"    Remedy: take []report.Section (or report.Run, for the export) and let the payload say how it describes itself.",
					path, position.Line, qualified)
			}
			return true
		})
	}
}

// TestRenderImportsNoProducer is the same rule one level up: the renderers may
// depend on the model and on presentation libraries, and on no producer at all.
//
// It is green on arrival — render imports no producer today — and it is here
// because the cheapest way to give a renderer a payload's facts is to import
// the package that makes them, which would satisfy TestNoPayloadReference while
// breaking exactly the same rule.
func TestRenderImportsNoProducer(t *testing.T) {
	producers := []string{
		"github.com/obentoo/bentoolkit/internal/autoupdate",
		"github.com/obentoo/bentoolkit/internal/overlay",
		"github.com/obentoo/bentoolkit/internal/snapshot",
	}

	fset := token.NewFileSet()

	for _, path := range renderSourceFiles(t) {
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}

		for _, spec := range file.Imports {
			imported := strings.Trim(spec.Path.Value, `"`)
			for _, producer := range producers {
				if imported == producer || strings.HasPrefix(imported, producer+"/") {
					t.Errorf("%s imports the producer %q — render turns sections into text and knows nothing about what made them (R7.4)", path, imported)
				}
			}
		}
	}
}
