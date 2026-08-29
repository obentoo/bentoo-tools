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
// all four itself, in the list below.
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

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// payloadTypes are the domain halves a renderer must not know.
//
// report.Report is on the list under its own name because it is the same type:
// D1 renames it to AutoupdateCheck and moves it into the payload position, so a
// renderer still holding a Report is holding a payload the rename has not
// caught up with yet.
var payloadTypes = []string{
	"Report",
	"AutoupdateCheck",
	"ManifestRun",
	"SnapshotRun",
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
	forbidden := make(map[string]bool, len(payloadTypes))
	for _, name := range payloadTypes {
		forbidden[name] = true
	}

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
			if !ok || pkg.Name != "report" {
				return true
			}
			if forbidden[selector.Sel.Name] {
				position := fset.Position(selector.Pos())
				t.Errorf("%s:%d names the payload type report.%s — a renderer that knows a payload is a renderer every new command has to edit (R7.4).\n"+
					"    Remedy: take []report.Section (or report.Run, for the export) and let the payload say how it describes itself.",
					path, position.Line, selector.Sel.Name)
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
