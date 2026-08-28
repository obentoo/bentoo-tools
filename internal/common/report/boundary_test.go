package report

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// forbiddenImports carries the two rules this package must not cross. Each
// rule travels with its own remedy, because they are different rules with
// different fixes and a shared message would send half the readers the wrong
// way: presentation belongs in the render package, but a dependency on
// internal/autoupdate is not fixed by moving it there — render sits under
// internal/common too. That one is fixed by taking the primitive fact instead.
var forbiddenImports = []struct {
	prefix string
	rule   string
	remedy string
}{
	{"github.com/charmbracelet/lipgloss", "presentation (R1.2)", presentationRemedy},
	{"github.com/charmbracelet/bubbletea", "presentation (R1.2)", presentationRemedy},
	{"github.com/charmbracelet/bubbles", "presentation (R1.2)", presentationRemedy},
	{"golang.org/x/term", "presentation (R1.2)", presentationRemedy},
	{"github.com/obentoo/bentoolkit/internal/autoupdate", "dependency direction (D2)", directionRemedy},
}

const (
	presentationRemedy = "internal/common/report describes WHAT a run found; it does not know how a run looks. " +
		"Move the formatting to internal/common/report/render, which is the package that may import this."

	directionRemedy = "a package under internal/common must not depend on internal/autoupdate — that inverts the " +
		"dependency direction. Moving it to internal/common/report/render does NOT fix this; render sits under " +
		"internal/common too. Take the primitive fact instead and convert at the adapter " +
		"(cmd/bentoo/overlay_autoupdate_report.go), the way GateFact takes a plain string cause rather than a DeclineCause."
)

// TestPackageImportsNoPresentation makes R1.2 mechanical instead of
// conventional. The model describes a run; it does not know how a run looks.
//
// GREEN ON ARRIVAL by design: this test cannot fail on the day the package is
// written, because the package has no reason to import lipgloss yet. It is a
// guard, not evidence. What proves it works is mutation — add a forbidden
// import, watch it fail, revert.
//
// # The mutation, written out rather than pointed at
//
// What follows is that mutation as it was actually run. It is written here,
// in the repository, because the story artifacts this package was developed
// against are not committed: a note saying "see .draft/red-evidence.yaml"
// names a file nobody who cloned this can open, and evidence a reader cannot
// reach is the same as no evidence (story 046, R8.3).
//
// Measured on 2026-08-28. Each rule was mutated on its own, because the two
// carry separate remedies and one shared message is exactly the instrument bug
// worth catching. Both times the import was added to model.go with a
// package-level `var _` so the package still compiled, the run below was made,
// and model.go was then restored byte for byte and the restore verified by
// checksum.
//
//	$ go test ./internal/common/report/ -run TestPackageImportsNoPresentation -v
//
// presentation (R1.2), via `import "github.com/charmbracelet/lipgloss"`:
//
//	boundary_test.go:117: model.go imports "github.com/charmbracelet/lipgloss", which crosses the presentation (R1.2) rule.
//	    internal/common/report describes WHAT a run found; it does not know how a run looks. Move the formatting to internal/common/report/render, which is the package that may import this.
//
// dependency direction (D2), via `import "github.com/obentoo/bentoolkit/internal/autoupdate"`:
//
//	boundary_test.go:117: model.go imports "github.com/obentoo/bentoolkit/internal/autoupdate", which crosses the dependency direction (D2) rule.
//	    <directionRemedy, printed in full — the const declared above>
//
// The second remedy is named instead of pasted for the reason a paste would
// exist at all: it is a const in this file, and a second copy of it here is the
// copy that goes stale. What the run establishes is that the message CARRIED
// its remedy, which is the half R7.3 asks for beyond naming the import — and
// that the two rules produced different remedies rather than a shared one.
//
// Quoting the forbidden paths above does not trip the sweep, and cannot: it
// reads the parsed import declarations, never the file's text, so a path in a
// comment is invisible to it. That is also why the evidence can live here at
// all, next to the rule it is evidence for.
//
// Why a test and not a comment: internal/common/tui states the identical rule
// in a comment, and the check path grew four hard-coded widths anyway.
//
// The sweep covers _test.go files too. A test in this package that needed
// lipgloss would mean the model had acquired presentation through the back
// door, and there is no legitimate reason for one to.
func TestPackageImportsNoPresentation(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading the package directory: %v", err)
	}

	fset := token.NewFileSet()
	scanned := 0

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		path := filepath.Join(".", entry.Name())
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		scanned++

		for _, spec := range file.Imports {
			imported := strings.Trim(spec.Path.Value, `"`)
			for _, forbidden := range forbiddenImports {
				if imported == forbidden.prefix || strings.HasPrefix(imported, forbidden.prefix+"/") {
					t.Errorf("%s imports %q, which crosses the %s rule.\n%s",
						path, imported, forbidden.rule, forbidden.remedy)
				}
			}
		}
	}

	// A sweep that scanned nothing passes vacuously and would keep passing
	// after someone renamed the files out from under it.
	if scanned == 0 {
		t.Fatal("scanned no .go files — the sweep passed without inspecting anything")
	}
}
