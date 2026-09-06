package report

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// forbiddenImports carries the two rules this package must not cross, and the
// seven imports that would cross one of them. Each ENTRY travels with its own
// remedy — entry rather than rule, since sub-task 16.4 — because a shared
// message would send most readers the wrong way: presentation belongs in the
// render package, but a dependency on a producer is not fixed by moving it
// there, since render sits under internal/common too and its own guard forbids
// the same three producers. That one is fixed by taking the primitive fact
// instead and converting at the adapter that already owns the kind — and the
// adapter differs per producer, which is why the direction rule now carries two
// remedies rather than one.
//
// # The three producers, and why this list held one of them until sub-task 16.4
//
// design.md's Architecture draws the producers as a subgraph of three:
// internal/autoupdate (with its validate half), internal/overlay's manifest and
// internal/snapshot's runner. S046-R7.3 fails this suite when the model imports
// a presentation library OR a producer package, so all three belong here. This
// list held internal/autoupdate alone, while the sibling guard over
// internal/common/report/render — TestRenderImportsNoProducer — already carried
// all three. Two guards over one rule, disagreeing about its subject, is what
// made the short list a defect rather than a scoping choice; and the model's
// guard was green either way, which is why nobody re-read it for four audits.
// The transcripts in TestPackageImportsNoPresentation's own doc below are what
// stands in for the failure the added entries cannot produce on their own.
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
	{"github.com/obentoo/bentoolkit/internal/overlay", "dependency direction (D2)", producerRemedy},
	{"github.com/obentoo/bentoolkit/internal/snapshot", "dependency direction (D2)", producerRemedy},
}

const (
	presentationRemedy = "internal/common/report describes WHAT a run found; it does not know how a run looks. " +
		"Move the formatting to internal/common/report/render, which is the package that may import this."

	directionRemedy = "a package under internal/common must not depend on internal/autoupdate — that inverts the " +
		"dependency direction. Moving it to internal/common/report/render does NOT fix this; render sits under " +
		"internal/common too. Take the primitive fact instead and convert at the adapter " +
		"(cmd/bentoo/overlay_autoupdate_report.go), the way GateFact takes a plain string cause rather than a DeclineCause."

	// producerRemedy is the direction rule's second remedy, and it exists
	// because the first one names an address. directionRemedy sends its reader
	// to cmd/bentoo/overlay_autoupdate_report.go, which is the right seam for
	// exactly one of the three producers; reusing it for the other two would be
	// the shared-message defect the type doc above describes, and it would send
	// two readers in three to a file that has nothing to do with what they
	// imported.
	producerRemedy = "a package under internal/common must not depend on a producer — internal/overlay establishes " +
		"what a manifest run found and internal/snapshot what a snapshot run found, and importing either inverts the " +
		"dependency direction exactly as internal/autoupdate does. Moving it to internal/common/report/render does " +
		"NOT fix this; render sits under internal/common too, and its own guard (TestRenderImportsNoProducer) forbids " +
		"the same three producers. Take the primitive facts instead and convert at the adapter that already owns the " +
		"kind — cmd/bentoo/overlay_manifest_report.go for internal/overlay, cmd/bentoo/snapshot_report.go for " +
		"internal/snapshot — the way GateFact takes a plain string cause rather than a DeclineCause."
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
//
// # The two producers the list did not hold, and evidence that they fire (16.4)
//
// S046-R7.3 fails this suite when the model imports a presentation library OR a
// producer package, and design.md's Architecture draws three producers.
// forbiddenImports held one of them — internal/autoupdate — while the sibling
// guard over internal/common/report/render, TestRenderImportsNoProducer,
// already listed all three. The list is now seven entries and the two guards
// name the same three producers.
//
// Nothing here imports either of the added paths, and nothing did: at the
// commit this was measured against, no non-test file under
// internal/common/report imported any obentoo/bentoolkit path at all. So the
// widening changed no run's verdict and could not have — which is both why the
// gap survived four audits and why S046-R8.3 declines to take the new entries
// on the word of whoever wrote them. A guard nobody can make fail is a guard
// nobody checks.
//
// What this guard fails on is an IMPORT, and it counts nothing: a blank import
// with no call behind it fails it exactly as forty calls would. That is the
// same sentence internal/overlay/boundary_test.go and
// internal/snapshot/boundary_test.go carry about the sibling rule one layer
// down, and the mutations below are blank imports for that reason — the weakest
// form the violation can take, no symbol used and nothing else in the file
// changed.
//
// Measured on 2026-09-02 against HEAD 0fd2f2e. Two mutations, each applied
// ALONE to model.go, run with the command above, and reverted immediately with
// `git checkout --`. Both reverts verified by md5sum against the pre-mutation
// file — 307ec0194356bf4f63578be766efee3b, identical before the first mutation
// and after the second — and by `git status --porcelain` not listing model.go.
//
// M1 — `import _ "github.com/obentoo/bentoolkit/internal/overlay"` in model.go:
//
//	boundary_test.go: model.go imports "github.com/obentoo/bentoolkit/internal/overlay", which crosses the dependency direction (D2) rule.
//	    <producerRemedy, printed in full — the const declared above>
//	--- FAIL: TestPackageImportsNoPresentation (0.00s)
//
// M2 — `import _ "github.com/obentoo/bentoolkit/internal/snapshot"`, on its own:
//
//	boundary_test.go: model.go imports "github.com/obentoo/bentoolkit/internal/snapshot", which crosses the dependency direction (D2) rule.
//	    <producerRemedy, printed in full — the const declared above>
//	--- FAIL: TestPackageImportsNoPresentation (0.00s)
//
// M3 — the half that measures the ENTRY rather than the sweep, taken on a
// pristine copy of 0fd2f2e unpacked into /tmp with `git archive`. Both
// mutations above were applied there against the five-entry list as it shipped,
// and the guard PASSED both times: the model could import internal/overlay or
// internal/snapshot and this suite would agree it was clean. The sweep, the
// parser and the message were all already correct — the only thing missing was
// the line. This file was then copied in over that same tree, mutation still
// applied, and reproduced M1 and M2 above byte for byte. That is the defect and
// its repair measured on one tree, one variable apart, and it is what makes the
// two added entries live rather than decorative.
//
// Both remedies are named rather than pasted, for the reason the 2026-08-28
// transcript above gives about its own: the remedy is a const in this file, and
// a second copy of it here is the copy that goes stale. What the runs establish
// beyond naming the import is that each message CARRIED the PRODUCERS' remedy —
// the manifest adapter and the snapshot adapter — and not internal/autoupdate's,
// which is the shared-message defect the type doc above warns about and the
// reason the direction rule has two remedies.
//
// The guard file's own line number is omitted from all three transcripts. A
// header that records a coordinate into a guard file sits inside the file it
// points at, so writing the number down moves the line it names. That is not
// hypothetical here: THIS section displaced the 2026-08-28 transcript's
// `boundary_test.go:117`, which was still exact the day it was written and is
// now the assertion's old address. That record is left as its author wrote it,
// cited as the measurement rather than corrected — the convention
// internal/common/report/render/contract_test.go adopted for the same reason.
// model.go carries no number in either transcript because the sweep prints
// none: it names the file and the import, which is the whole of what the rule
// is about. The verbatim numbers the runs printed are in
// .draft/red-evidence.yaml, a file that does not move what it describes.
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
