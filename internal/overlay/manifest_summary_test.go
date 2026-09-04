package overlay

// Authored for story 046, sub-task 17.3 — S046-R5.2, S046-R8.3.
//
// Written from the contract: S046-R5.2 — "WHEN a finding produced inside a
// library is rendered, THE SYSTEM SHALL render it in every mode and include it
// in the export, without the library having chosen how it looks" — and
// design.md D5, which names this exact expression as the defect: "a producer
// today ends its run by handing the Reporter a formatted sentence —
// BatchDone(\"%d ok, %d failed\") — which is a report squeezed through a
// progress channel, where nothing can count it, export it or re-render it.
// Under this design the producer returns its facts and the caller builds the
// payload; BatchDone receives a summary the payload produced rather than one
// the library invented."
//
// Sub-task 5.1 (manifest_result_test.go) moved the SOURCE of the two numbers:
// they come from `func (ManifestResult) Ok` / `func (ManifestResult) Failed`
// instead of counters kept for the sentence alone. It did not move who chooses
// the WORDING. The format string is still declared in this package, so the one
// thing D5 asks for — the library returns facts and does not decide how they
// look — is the one thing still outstanding.
//
// # This test states a DISAGREEMENT it does not settle
//
// manifest.go argues, at length and in good faith, for keeping the call: the
// summary is composed from the values the caller is about to receive, so "the
// sentence is one more reader of the numbers rather than the only place they
// exist", and Unchanged Behavior 3 says the Reporter's consumers do not move.
// That argument is not stupid. D5 says the opposite in as many words, and
// nothing in .draft/deviations.yaml records a decision either way — `grep
// BatchDone` over it returns nothing.
//
// So this guard exists to make the disagreement VISIBLE and FAILING, for a
// human to decide, and it has two honest endings:
//
//   - the design wins: RegenerateManifests stops composing the sentence. The
//     caller in cmd/bentoo builds it from the returned ManifestResult — which
//     already carries Ok, Failed and NotEvaluated — and passes it, or calls
//     BatchDone itself. This test goes green with no change to it.
//   - the code wins: the retention is deliberate, and it is recorded in
//     .draft/deviations.yaml against D5, with manifest.go's reasoning as the
//     rationale. Then THIS FILE IS DELETED. A guard kept beside an accepted
//     deviation is a guard someone will silence, and a silenced guard is worse
//     than none.
//
// What is not an ending: leaving both the design sentence and the code as they
// are, with no record. That is the state at HEAD, and it is the state this file
// refuses to let stay quiet.
//
// # RED ON ARRIVAL
//
// One call site in this package: RegenerateManifests, in manifest.go, hands
// BatchDone a fmt.Sprintf over a format literal it declares itself.
//
// # It reads LITERALS, not text
//
// The detector walks the argument expression of each BatchDone call through the
// AST. It never matches source text, and it therefore never fires on a COMMENT
// — including the 13-line comment above the call, which quotes the format
// string while arguing for it, and including this header, which quotes it
// twice. That distinction is not decoration: render/width.go taught this repo
// the other way round, where a grep-shaped rule reported the comment explaining
// a removed construct as the construct.
//
// # Why the sweep does not require a BatchDone call to exist
//
// The vacuity guard is on the FILES, not on the call sites: manifestSummarySources
// fatals when it scans none, the way render/contract_test.go does. It cannot
// also fatal on zero calls, because "RegenerateManifests no longer calls
// BatchDone at all" is one of the two shapes the fix may legitimately take, and
// a guard that failed on its own remedy would be unpassable. The classifier is
// pinned separately, in both directions, by TestManifestSummaryFaultClassifier —
// which is what keeps a sweep that classifies NOTHING from passing over a
// package full of composed sentences.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// summaryRemedy is the sentence a maintainer meeting this failure needs. It
// says where the wording belongs, not merely that it is in the wrong place.
const summaryRemedy = "Remedy: the caller composes the sentence from the returned ManifestResult — it already carries Ok, Failed and NotEvaluated.\n" +
	"    A package under internal/overlay establishes the facts of a run (S046-R5.1) and does not choose how they look (S046-R5.2):\n" +
	"    a sentence built here is a finding that cannot be re-rendered in another mode, counted, or exported, because it left as text.\n" +
	"    If the retention is deliberate instead, record it in .draft/deviations.yaml against design.md D5 and delete this file."

// manifestSummaryFault classifies the argument handed to BatchDone: it answers
// whether the SENTENCE was composed inside this package, and why.
//
// Two signals, both structural:
//
//   - a non-empty string literal anywhere in the argument. Wording is wording
//     whether it arrives as a format string, as a concatenation operand, or as
//     a bare sentence.
//   - a call into fmt. Composition is composition even when the format string
//     is held in a variable, which the literal rule alone would miss.
//
// What is deliberately ACCEPTED: an identifier, a field, a method call on a
// value the caller supplied, and the empty literal "". A library may still
// close the batch — cmd/bentoo/overlay_autoupdate.go calls BatchDone("") today
// — because the rule is about who chose the WORDS, not about who ends the run.
func manifestSummaryFault(arg ast.Expr) string {
	fault := ""

	ast.Inspect(arg, func(node ast.Node) bool {
		if fault != "" {
			return false
		}
		switch n := node.(type) {
		case *ast.BasicLit:
			if n.Kind != token.STRING {
				return true
			}
			text, err := strconv.Unquote(n.Value)
			if err != nil {
				text = n.Value
			}
			if strings.TrimSpace(text) != "" {
				fault = fmt.Sprintf("it is built from the wording %s declared here", n.Value)
				return false
			}
		case *ast.CallExpr:
			sel, ok := n.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "fmt" {
				return true
			}
			fault = fmt.Sprintf("it is composed here by fmt.%s", sel.Sel.Name)
			return false
		}
		return true
	})

	return fault
}

// manifestSummarySources lists this package's non-test Go files, so a sweep
// that found none fails loudly instead of passing vacuously.
//
// The directory comes from this file's own path, never from the process working
// directory: a guard that reads whatever directory it was started in is a guard
// that can be moved off its subject.
func manifestSummarySources(t *testing.T) []string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate this test's source file — the sweep has no subject")
	}
	pkgDir := filepath.Dir(thisFile)

	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("reading the package directory %q: %v", pkgDir, err)
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.Join(pkgDir, name))
	}

	if len(files) == 0 {
		t.Fatal("found no non-test Go file — the sweep would pass without inspecting anything")
	}
	return files
}

// TestManifestSummaryIsNotComposedByTheLibrary is the guard: no non-test file
// under internal/overlay hands BatchDone a sentence it wrote itself.
//
// _test.go files are exempt, for the reason boundary_test.go gives about the
// same package: a test may legitimately name the wording — asserting that the
// live region still ends on a summary means knowing what one looks like — and
// the rule is about what the LIBRARY does.
func TestManifestSummaryIsNotComposedByTheLibrary(t *testing.T) {
	fset := token.NewFileSet()

	for _, path := range manifestSummarySources(t) {
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			// A parse failure FAILS rather than skips: an unreadable source
			// scanned as if it were empty yields exactly the clean result a
			// compliant package yields.
			t.Fatalf("parsing %s: %v", path, err)
		}

		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "BatchDone" || len(call.Args) != 1 {
				return true
			}

			if fault := manifestSummaryFault(call.Args[0]); fault != "" {
				position := fset.Position(call.Pos())
				t.Errorf("%s:%d passes BatchDone a summary the library invented: %s.\n"+
					"    design.md D5: \"BatchDone receives a summary the payload produced rather than one the library invented\".\n"+
					"    %s",
					filepath.Base(path), position.Line, fault, summaryRemedy)
			}
			return true
		})
	}
}

// TestManifestSummaryFaultClassifier pins the classifier one case per
// direction, so a sweep that classified nothing as composed cannot pass over a
// package full of composed sentences.
func TestManifestSummaryFaultClassifier(t *testing.T) {
	cases := []struct {
		name   string
		expr   string
		reject bool
	}{
		// Composed here — the defect, in each shape it can take.
		{"sprintf over the counts", `fmt.Sprintf("%d ok, %d failed", result.Ok(), result.Failed())`, true},
		{"sprintf with a held format", `fmt.Sprintf(summaryFormat, result.Ok(), result.Failed())`, true},
		{"concatenation", `strconv.Itoa(result.Ok()) + " ok"`, true},
		{"a bare sentence", `"all done"`, true},
		{"sprint of the facts", `fmt.Sprint(result.Ok(), " ok")`, true},

		// Composed elsewhere, or not composed at all — the shapes the fix may
		// legitimately take, and each MUST stay legal or the guard is unpassable.
		{"a value the caller supplied", `summary`, false},
		{"a field of the options", `opts.Summary`, false},
		{"a method on the payload", `payload.Summary()`, false},
		{"a payload asked for its own sentence", `run.Tally().Sentence`, false},
		{"closing the batch with nothing", `""`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tc.expr)
			if err != nil {
				t.Fatalf("parsing the fixture %q: %v", tc.expr, err)
			}

			fault := manifestSummaryFault(expr)
			if tc.reject && fault == "" {
				t.Errorf("%s was accepted — the guard would let the library keep choosing the wording (S046-R5.2)", tc.expr)
			}
			if !tc.reject && fault != "" {
				t.Errorf("%s was rejected as %q — the guard forbids a shape the remedy itself needs, and could never go green", tc.expr, fault)
			}
		})
	}
}

// TestManifestSummaryDetectorReadsLiteralsNotComments is the render/width.go
// lesson, checked rather than asserted in prose: a comment quoting the removed
// sentence — which is what an accepted deviation, or a changelog note, would
// leave behind — is not the defect, and a guard that reported it would send its
// reader to delete the explanation instead of the code.
//
// The fixture is the shape the file would have AFTER the fix: the wording lives
// in the caller, the comment still explains what moved.
func TestManifestSummaryDetectorReadsLiteralsNotComments(t *testing.T) {
	const src = `package overlay

func run(rep reporter, summary string) {
	// The sentence used to be composed here, as fmt.Sprintf("%d ok, %d failed",
	// result.Ok(), result.Failed()). It is the caller's now (D5).
	rep.BatchDone(summary) // was: "%d ok, %d failed"
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture.go", src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing the fixture: %v", err)
	}

	calls := 0
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "BatchDone" || len(call.Args) != 1 {
			return true
		}
		calls++
		if fault := manifestSummaryFault(call.Args[0]); fault != "" {
			t.Errorf("the detector reported a fixed call as %q — it read the comment quoting the old sentence, not the code", fault)
		}
		return true
	})

	if calls != 1 {
		t.Fatalf("found %d BatchDone call(s) in the fixture, want 1 — the check inspected nothing", calls)
	}
}
