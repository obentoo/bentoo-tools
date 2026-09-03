package report

// Authored for story 046, sub-task 1.2 — R7.2, R7.3, R8.3.
//
// Written from the contract: design.md D2 states the obligation in one
// sentence — boundary_test.go "grows a second guard alongside its import check:
// a field-level assertion that no type in the package — Section and Table
// included — declares a width, a colour, a style or an escape sequence. The
// import guard is green on arrival by design (its own comment says so); this
// one is not, and that is the point of adding it."
//
// The last clause is the one part that did not survive contact with the
// package: this guard is green on arrival too. What that changes, and what it
// does not, is the subject of the second section below.
//
// # Why this is a separate file from boundary_test.go
//
// The Tests field for 1.2 names boundary_test.go. This file is authored beside
// it rather than over it: boundary_test.go already holds the import guard and
// its evidence note, and replacing that file would delete a passing guard to
// add one. Same package, same sweep, one rule per file.
//
// # It IS green on arrival, and R8.3 is why that is not enough
//
// D2 expected this guard to fail on the day it was written where the import
// guard could not. It does not. Nothing in the model declares a width, a colour
// or a border — that is what sub-task 1.1 was careful about — so the sweep
// passes on its first run exactly as boundary_test.go's does, and saying
// otherwise here would leave a claim the very next `go test` contradicts.
//
// The design's point survives the correction: a guard is worth nothing until it
// has been seen to fail. Only the moment moves. It is shown by MUTATION rather
// than by the first run, which is what R8.3 asks for, and the two mutations are
// written out below for the reason boundary_test.go gives at more length — the
// story artifacts holding them (.draft/red-evidence.yaml) are not committed, so
// a pointer to that file is a pointer to nothing for anyone who cloned this.
//
// Measured on 2026-08-28. Each mutation was applied on its own, run, and
// reverted immediately, with the file restored byte for byte and the restore
// verified by checksum.
//
//	$ go test ./internal/common/report/ -run TestPackageDeclaresNoPresentationField -v
//
// Adding a `Width int` field, json tag "width", to Tally in model.go:
//
//	boundary_fields_test.go:307: model.go: Tally.Width declares presentation: the word "width" is a rendering concern, not a fact about a run.
//	        how wide a value is drawn is measured in render from lipgloss.Width, never carried by the model
//	        Remedy: the value belongs in render.Options or in the renderer that computes it. internal/common/report describes WHAT a run found; it does not know how a run looks (R7.1, R7.2).
//
// Adding a `BorderChar string` field to Section in section.go. The second
// mutation is not a repetition of the first: model.go is where the model has
// always lived, while Section and Table arrived with sub-task 1.1, and D2 names
// them explicitly. Mutating only Tally would leave "Section and Table included"
// asserted and unmeasured.
//
//	boundary_fields_test.go:307: section.go: Section.BorderChar declares presentation: the word "border" is a rendering concern, not a fact about a run.
//	        a border character is drawn by render/style.go and by nothing else
//	        <the Remedy line above, word for word — fieldViolation closes every message with it>
//
// Both name the TYPE and the FIELD, which is the whole of R7.2, and both say
// where the value belongs instead, which is what a maintainer meeting the
// failure actually needs.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

// presentationWords is the vocabulary a field name may not be built from: the
// six things R7.1 enumerates — escape sequence, colour, column width, padding,
// border character, terminal dimension — plus the words a renderer reaches for
// when it has already made the decision.
//
// Each word travels with what it costs, because a maintainer meeting this
// failure needs to know why the field is wrong and not merely that it is: the
// remedy is never "rename it", it is "the value belongs in render.Options or in
// the renderer that computes it".
//
// # What is deliberately NOT here
//
//   - "cells". Row.Cells is a record's VALUES, not a display measurement, and
//     it is the name render.row already uses. Forbidding it would forbid the
//     section vocabulary D2 moves in.
//   - "column". A Table's column count is structure — how many values a row
//     holds — while a column's WIDTH is presentation, and "width" is already
//     forbidden. Banning the noun would ban the structure with it.
//
// Both exclusions are stated rather than left as gaps, so a later reader can
// tell a considered omission from an oversight.
var presentationWords = map[string]string{
	"width":     "how wide a value is drawn is measured in render from lipgloss.Width, never carried by the model",
	"height":    "a terminal dimension: it describes the device, not the run",
	"color":     "a colour is chosen by a renderer, and only in the modes that have one",
	"colour":    "a colour is chosen by a renderer, and only in the modes that have one",
	"style":     "a style is presentation by definition",
	"padding":   "padding is the space a renderer puts around a value",
	"margin":    "a margin is the space a renderer puts around a block",
	"border":    "a border character is drawn by render/style.go and by nothing else",
	"escape":    "an escape sequence in the model is the defect this package exists to remove (R7.1)",
	"ansi":      "an escape sequence in the model is the defect this package exists to remove (R7.1)",
	"bold":      "weight is a decoration a renderer applies",
	"italic":    "weight is a decoration a renderer applies",
	"underline": "weight is a decoration a renderer applies",
	"glyph":     "a glyph is a rendering of a fact, not the fact",
	"icon":      "an icon is a rendering of a fact, not the fact",
	"emoji":     "an icon is a rendering of a fact, not the fact",
	"terminal":  "a terminal dimension describes the device the report is printed to",
	"screen":    "the model describes runs; it does not describe screens (R7)",
	"viewport":  "a viewport is the fullscreen renderer's own state",
	"indent":    "indentation is how a renderer shows nesting",
	"ellipsis":  "an ellipsis marks a cut a renderer made; the model keeps the value whole (R7.1)",
	"truncate":  "shortening a value is always a rendering decision, never a loss of data",
	"shorten":   "shortening a value is always a rendering decision, never a loss of data",
	"theme":     "a theme is a set of rendering choices",
	"palette":   "a palette is a set of rendering choices",
	"spinner":   "a spinner is a live-region animation, not a fact a run established",
}

// splitWords breaks a Go identifier into its lower-cased camel-case words, so
// the vocabulary is matched against WORDS rather than substrings.
//
// The distinction is load-bearing in both directions. A substring match on
// "width" would also fire on nothing useful, but a substring match on "pad"
// would fire on "Padded" AND on "Update" — and "NotEvaluated" contains "valuate"
// the same way. Splitting first means BorderChar is caught by its first word
// while Updated is not caught at all.
//
// Initialisms are kept whole: ANSIEscape splits to ansi + escape, not to a
// + n + s + i.
func splitWords(name string) []string {
	var words []string
	var current []rune

	flush := func() {
		if len(current) > 0 {
			words = append(words, strings.ToLower(string(current)))
			current = nil
		}
	}

	runes := []rune(name)
	for i, r := range runes {
		switch {
		case r == '_':
			flush()
		case unicode.IsUpper(r):
			// A capital starts a new word unless it continues an initialism:
			// the previous rune is upper and the next one is not lower.
			prevUpper := i > 0 && unicode.IsUpper(runes[i-1])
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if !prevUpper || nextLower {
				flush()
			}
			current = append(current, r)
		default:
			current = append(current, r)
		}
	}
	flush()

	return words
}

// presentationWordIn answers which forbidden word a field name is built from,
// and why that word is forbidden. An empty word means the name is clean.
func presentationWordIn(fieldName string) (word, why string) {
	for _, w := range splitWords(fieldName) {
		if why, forbidden := presentationWords[w]; forbidden {
			return w, why
		}
	}
	return "", ""
}

// fieldViolation is the sentence R7.2 asks for: it names the TYPE and the FIELD,
// because "a presentation field was found in internal/common/report" sends the
// reader to a package and leaves them to search it.
func fieldViolation(typeName, fieldName, word, why string) string {
	return fmt.Sprintf(
		"%s.%s declares presentation: the word %q is a rendering concern, not a fact about a run.\n"+
			"    %s\n"+
			"    Remedy: the value belongs in render.Options or in the renderer that computes it. "+
			"internal/common/report describes WHAT a run found; it does not know how a run looks (R7.1, R7.2).",
		typeName, fieldName, word, why)
}

// TestPresentationFieldRule pins the classifier the sweep is built on, one case
// per direction. Without it, a sweep that classified NOTHING as presentation
// would pass over a package full of widths and report success.
func TestPresentationFieldRule(t *testing.T) {
	cases := []struct {
		field  string
		reject bool
	}{
		// R7.1's six categories, in eight cases. Width and ColumnWidth both
		// stand for column width — the same forbidden word reached by a
		// one-word name and by a two-word one — and Style is the eighth,
		// which that list does not name: it comes from "the words a renderer
		// reaches for" the doc comment above adds beside them.
		{"Width", true},
		{"ColumnWidth", true},
		{"Color", true},
		{"BorderChar", true},
		{"Padding", true},
		{"TerminalHeight", true},
		{"ANSIEscape", true},
		{"Style", true},

		// Structure, and it must stay legal — these are the fields the section
		// vocabulary is made of.
		{"Title", false},
		{"Lead", false},
		{"Notes", false},
		{"Rows", false},
		{"Headers", false},
		{"Cells", false},
		{"Detail", false},

		// The names a substring match would have taken with it.
		{"Updated", false},
		{"NotEvaluated", false},
		{"Complete", false},
		{"Depth", false},
		{"DistfilesToFetch", false},
	}

	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			word, _ := presentationWordIn(tc.field)
			if tc.reject && word == "" {
				t.Errorf("%s was accepted — the guard would let a %s field into the model", tc.field, tc.field)
			}
			if !tc.reject && word != "" {
				t.Errorf("%s was rejected for the word %q — the guard forbids a field the section vocabulary needs", tc.field, word)
			}
		})
	}
}

// TestPresentationFieldFailureNamesTypeAndField pins R7.2's second half. A
// guard that fails without naming the offending field costs its reader a search
// through every type in the package, which is how a mechanical rule turns back
// into a conventional one.
func TestPresentationFieldFailureNamesTypeAndField(t *testing.T) {
	word, why := presentationWordIn("Width")
	if word == "" {
		t.Fatal("Width is not classified as presentation — the message cannot be checked")
	}

	msg := fieldViolation("Tally", "Width", word, why)

	for _, want := range []string{"Tally", "Width", "render"} {
		if !strings.Contains(msg, want) {
			t.Errorf("the failure message does not name %q:\n%s", want, msg)
		}
	}
}

// TestPackageDeclaresNoPresentationField is the guard itself: every struct
// field declared in this package, Section and Table included once they land.
//
// GREEN ON ARRIVAL, and R8.3 is answered by mutation rather than by hope. The
// two mutations are in the file header, with the message this function produced
// under each one, verbatim — one field in model.go and one in section.go, so
// the evidence covers both files the sweep has to reach.
//
// The sweep covers _test.go files for the same reason the import guard does: a
// fixture in this package carrying a width would be presentation entering
// through the back door, and there is no legitimate reason for one to.
func TestPackageDeclaresNoPresentationField(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading the package directory: %v", err)
	}

	fset := token.NewFileSet()
	fields := 0

	for _, dirEntry := range entries {
		if dirEntry.IsDir() || !strings.HasSuffix(dirEntry.Name(), ".go") {
			continue
		}

		path := filepath.Join(".", dirEntry.Name())
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}

		ast.Inspect(file, func(node ast.Node) bool {
			spec, ok := node.(*ast.TypeSpec)
			if !ok {
				return true
			}
			structType, ok := spec.Type.(*ast.StructType)
			if !ok || structType.Fields == nil {
				return true
			}

			for _, field := range structType.Fields.List {
				for _, name := range field.Names {
					fields++
					if word, why := presentationWordIn(name.Name); word != "" {
						t.Errorf("%s: %s", path, fieldViolation(spec.Name.Name, name.Name, word, why))
					}
				}
			}
			return true
		})
	}

	// A sweep that inspected no field passes vacuously, and would keep passing
	// after the types were moved out from under it.
	if fields == 0 {
		t.Fatal("inspected no struct field — the sweep passed without looking at anything")
	}
}
