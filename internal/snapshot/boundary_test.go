package snapshot

// Authored for story 046, sub-task 7.3 — R5.3, R8.3.
//
// Written from the contract: R5.3 states the rule and the message it owes —
// "IF a package under internal/overlay or internal/snapshot imports the
// terminal printer, THEN THE SYSTEM SHALL fail its test suite naming the
// package and the import." design.md D7 gives the count it is holding the line
// against: the 46 output.* calls this story moves out of internal/.
//
// The shape is internal/common/report/boundary_test.go's, reused deliberately:
// a rule stated in a comment is a rule that gets broken, and that file says so
// about the identical rule in internal/common/tui, which grew four hard-coded
// widths anyway.
//
// GREEN ON ARRIVAL, unlike its twin in internal/overlay: no file in this
// package imports the printer today, so this half of the rule is a guard rather
// than a repair. R8.3 is answered by mutation — an import of
// internal/common/output was reinstated in internal/snapshot/runner.go, the
// failure was recorded in .draft/red-evidence.yaml, and the mutation reverted.
//
// It exists anyway because R5.3 names BOTH trees, and because internal/snapshot
// is where the next printing library would be written: a rule enforced on one
// half is a rule that moves to the other half.
//
// # Why _test.go files are exempt
//
// Sub-task 7.3's own validation says so: "grep -rn 'common/output'
// internal/overlay internal/snapshot returns nothing outside tests". A test may
// legitimately name the printer — asserting that a finding carries no colour
// means knowing what colour looks like — and the rule is about what the LIBRARY
// does, not about what a test may read.

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// printerImport is the terminal printer this package may not reach for.
//
// It is matched as a package path rather than as a name: internal/common/output
// is a printer, and the point of the rule is the DIRECTION — a library under
// internal/overlay decides what it found, and a caller in cmd/bentoo decides
// how it looks.
const printerImport = "github.com/obentoo/bentoolkit/internal/common/output"

const printerRemedy = "a package under internal/snapshot establishes findings; it does not print them (R5.1). " +
	"Return the finding to the caller as a value and let cmd/bentoo render it — that is what lets the same " +
	"finding be shown in three modes, exported and counted (R5.2). Moving the call to another package under " +
	"internal/ does NOT fix this; the rule is about the layer, not the file."

// TestNoPrinterImport is the guard. It sweeps this package's non-test Go files
// and fails naming the FILE and the IMPORT, because "internal/overlay imports
// the printer" sends a reader to a directory of forty files and leaves them to
// search it.
func TestNoPrinterImport(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading the package directory: %v", err)
	}

	fset := token.NewFileSet()
	scanned := 0

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		path := filepath.Join(".", name)
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		scanned++

		for _, spec := range file.Imports {
			imported := strings.Trim(spec.Path.Value, `"`)
			if imported == printerImport || strings.HasPrefix(imported, printerImport+"/") {
				t.Errorf("internal/snapshot: %s imports %q — the library is choosing how a finding looks (R5.3).\n    %s",
					path, imported, printerRemedy)
			}
		}
	}

	// A sweep that scanned nothing passes vacuously and would keep passing
	// after the files were renamed out from under it.
	if scanned == 0 {
		t.Fatal("scanned no non-test .go file — the sweep passed without inspecting anything")
	}
}
