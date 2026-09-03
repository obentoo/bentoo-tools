package overlay

// Authored for story 046, sub-task 7.3 — R5.3, R8.3.
//
// Written from the contract: R5.3 states the rule and the message it owes —
// "IF a package under internal/overlay or internal/snapshot imports the
// terminal printer, THEN THE SYSTEM SHALL fail its test suite naming the
// package and the import." design.md D7 sizes the work behind that rule: 40
// calls into internal/common/output, made by the five files under
// internal/overlay that printed directly — compare.go, annotate_baseline.go,
// baseline.go, realign_reviewer.go and status.go. Two earlier figures for that
// same set are retracted and should not be re-derived: 46 came from a grep
// filtered to a fixed list of printer names, which missed three of those files;
// 61 counted every occurrence of output. across the whole internal/overlay and
// internal/snapshot trees, which reached four files past them. Both were
// published as "calls in these five files", which neither was.
//
// That count describes the WORK. It is not what this test fails on. The guard
// fails on an IMPORT — a non-test file in this package importing
// internal/common/output — and counts nothing at all: a blank import with no
// call behind it fails it exactly as forty calls would, which is the mutation
// recorded below.
//
// The shape is internal/common/report/boundary_test.go's, reused deliberately:
// a rule stated in a comment is a rule that gets broken, and that file says so
// about the identical rule in internal/common/tui, which grew four hard-coded
// widths anyway.
//
// RED ON ARRIVAL, and not by mutation: five files in this package imported the
// printer when this was written. This guard goes green exactly when 7.1, 7.2
// and 7.3 have finished moving them.
//
// # Evidence that it still fails, taken after those three turned it green (R8.3)
//
// The paragraph above stops being checkable the moment the guard passes: a green
// test proves nothing about what it would catch. R8.3 asks for the evidence a
// green guard cannot supply, and the story artifacts holding the original Red
// (.draft/red-evidence.yaml) are not committed — `git ls-files .epic/` returns
// nothing — so a pointer to that file is a pointer to nothing for anyone who
// cloned this. It is written down here instead.
//
// Measured on 2026-08-29, against sub-task 7.4. One mutation, applied on its
// own, run, and reverted immediately, with status.go restored byte for byte.
//
// Mutation: the printer import was reinstated in status.go, blank —
//
//	_ "github.com/obentoo/bentoolkit/internal/common/output"
//
// Observed:
//
//	--- FAIL: TestNoPrinterImport (0.00s)
//	    boundary_test.go:81: internal/overlay: status.go imports
//	    ".../internal/common/output" — the library is choosing how a finding
//	    looks (R5.3).
//	            a package under internal/overlay establishes findings; it does
//	            not print them (R5.1). Return the finding to the caller as a
//	            value and let cmd/bentoo render it — that is what lets the same
//	            finding be shown in three modes, exported and counted (R5.2).
//	            Moving the call to another package under internal/ does NOT fix
//	            this; the rule is about the layer, not the file.
//
// A BLANK import is what was reinstated on purpose. It is the weakest form the
// violation can take — no symbol is used, so nothing else in the file changes —
// and the guard still catches it, which is the property worth recording: the
// rule is about the dependency, not about whether anything came of it.
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

const printerRemedy = "a package under internal/overlay establishes findings; it does not print them (R5.1). " +
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
				t.Errorf("internal/overlay: %s imports %q — the library is choosing how a finding looks (R5.3).\n    %s",
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
