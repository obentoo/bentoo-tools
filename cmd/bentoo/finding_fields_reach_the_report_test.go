package main

// Every fact a Finding carries must reach the operator, or say why it does not.
//
// This is the rule story 047 was missing. The story moved `overlay compare` from
// printing to producing Finding values, and the guards that existed watched the
// PRODUCER: internal/overlay/boundary_test.go fails when that package imports the
// terminal printer, which is the half of the split that was easy to get wrong on
// purpose. Nothing watched the other half. Finding.Effect, Finding.Origin and
// Finding.Proposal arrived on this side and `func comparePackageFindings` read
// only Detail, so S032-R5.2, S032-R5.3, S032-R5.4 and the maintainer's own
// declared reason rendered as silence. No test failed, because a field nobody
// reads breaks nothing that anybody asserts.
//
// # The list here is the COMPLEMENT, and that is the whole design
//
// A registry of the fields that must be read would be wrong the first time a
// producer adds one and forgets it — the objection `func comparePackageFindings`
// already states about a list of finding kinds. This names only the fields
// deliberately NOT read, each with its reason, and derives the rest from the
// struct. A new field on Finding fails this test until somebody either reads it
// or writes down why the report does not.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// findingFieldsNotCarriedHere is every exported Finding field this package does
// not read, and why.
//
// They share one shape: the PRODUCER already put the fact into the sentence the
// row carries as its Reason, so a second read here would be a second answer to
// one question — and the one that drifted would contradict the sentence printed
// beside it. That is the argument the comment above `func comparePkgFacts` makes
// about Authorship, applied to the rest.
var findingFieldsNotCarriedHere = map[string]string{
	"Version": "the row's own columns are built from CompareResult.LocalVersion, which is the " +
		"version the comparison was made at; the finding's copy is the same fact for a consumer holding findings alone",
	"Upstream": "as Version, from CompareResult.RemoteVersion",
	"Entry": "the producer spells it into Detail — `patched — declared by <entry>` — and a second copy would " +
		"print the registry key twice on one line",
	"Added": "`func compareDiffCell` builds the diff cell from CompareResult.DiffAdded, and " +
		"internal/overlay/compare_diff_counts_fence_test.go permits exactly two functions to name those fields",
	"Removed":    "as Added, from CompareResult.DiffRemoved",
	"Classified": "the run-level share is `func compareClassificationNote`'s, over CompareReport and CompareResult.Classified; a second denominator here could disagree with it",
	"Authorship": "it does not cross at all — see the comment above `func comparePkgFacts`: `func compareFindings` already spells the proved case into the sentence the row carries",
	"ProvedBy": "the same sentence names the proving file (\"our ebuild references <file>, which ::gentoo does not ship\"), " +
		"so it reaches the operator as prose rather than twice",
}

func TestEveryFindingFieldReachesTheReportOrSaysWhyNot(t *testing.T) {
	declared := exportedFindingFields(t)
	read := findingFieldsReadHere(t)

	for _, field := range declared {
		_, exempt := findingFieldsNotCarriedHere[field]
		switch {
		case read[field] && exempt:
			t.Errorf("Finding.%s is listed as not carried here, and IS read. Delete the entry: a "+
				"reason nobody can act on is worse than none, and the next reader will believe it.", field)
		case !read[field] && !exempt:
			t.Errorf("Finding.%s crosses the boundary and NOTHING in this package reads it, so whatever "+
				"it establishes reaches no operator. Read it where the report is built, or add it to "+
				"findingFieldsNotCarriedHere with the reason the report does not need it. This is the "+
				"defect story 047 shipped four times over: the field arrived, no line was built from it, "+
				"and no test failed.", field)
		}
	}

	// A reason for a field that no longer exists is a registry gone stale, which
	// is the failure mode this list is otherwise written to avoid.
	for field := range findingFieldsNotCarriedHere {
		if !slices.Contains(declared, field) {
			t.Errorf("findingFieldsNotCarriedHere names %q, which is not a field of overlay.Finding. "+
				"Drop the entry.", field)
		}
	}
}

// TestFindingValuesAreNamedFinding keeps the guard above honest.
//
// It reads `finding.<Field>` and nothing else, so a rename would empty it in
// silence — every field would look unread, every entry would look stale, and the
// test would fail loudly rather than pass hollowly. That is the safe direction,
// but a reader meeting fourteen failures deserves to be told the cause, so this
// states the convention as its own case.
func TestFindingValuesAreNamedFinding(t *testing.T) {
	for _, file := range packageFiles(t) {
		for _, decl := range file.ast.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Type.Params == nil {
				continue
			}

			// Every name whose DECLARED type makes a range over it yield an
			// overlay.Finding. Read off the source rather than resolved, which is
			// why only parameters qualify: their type is written down beside them.
			// `gate.Findings` in overlay_validate.go is []validate.Finding and must
			// not be caught here — a guard that fires on the wrong type teaches
			// readers to silence it.
			sliceParams := map[string]bool{}
			reportParams := map[string]bool{}
			for _, param := range fn.Type.Params.List {
				for _, name := range param.Names {
					switch {
					case isOverlayFinding(param.Type):
						if name.Name != "finding" {
							t.Errorf("%s: %s takes an overlay.Finding named %q. Call it `finding`: "+
								"TestEveryFindingFieldReachesTheReportOrSaysWhyNot reads that name and no other.",
								file.name, fn.Name.Name, name.Name)
						}
					case isOverlayFindingSlice(param.Type):
						sliceParams[name.Name] = true
					case isCompareReportPointer(param.Type):
						reportParams[name.Name] = true
					}
				}
			}

			ast.Inspect(fn, func(n ast.Node) bool {
				rng, ok := n.(*ast.RangeStmt)
				if !ok || rng.Value == nil {
					return true
				}
				if !rangesOverFindings(rng.X, sliceParams, reportParams) {
					return true
				}
				if bound, ok := rng.Value.(*ast.Ident); ok && bound.Name != "finding" {
					t.Errorf("%s: %s ranges over overlay.Finding values bound as %q. Call it `finding`, "+
						"for the same reason.", file.name, fn.Name.Name, bound.Name)
				}
				return true
			})
		}
	}
}

// rangesOverFindings reports whether expr yields overlay.Finding values: the
// slice parameter itself, or the Findings field of a *overlay.CompareReport
// parameter.
func rangesOverFindings(expr ast.Expr, sliceParams, reportParams map[string]bool) bool {
	switch node := expr.(type) {
	case *ast.Ident:
		return sliceParams[node.Name]
	case *ast.SelectorExpr:
		base, ok := node.X.(*ast.Ident)
		return ok && node.Sel.Name == "Findings" && reportParams[base.Name]
	}
	return false
}

// isOverlayFindingSlice reports whether an expression spells []overlay.Finding.
func isOverlayFindingSlice(expr ast.Expr) bool {
	array, ok := expr.(*ast.ArrayType)
	return ok && array.Len == nil && isOverlayFinding(array.Elt)
}

// isCompareReportPointer reports whether an expression spells
// *overlay.CompareReport.
func isCompareReportPointer(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "CompareReport" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "overlay"
}

// isOverlayFinding reports whether an expression spells `overlay.Finding`.
func isOverlayFinding(expr ast.Expr) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Finding" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "overlay"
}

// exportedFindingFields reads the struct itself, so the rule tracks the type
// rather than a copy of it somebody has to remember to update.
func exportedFindingFields(t *testing.T) []string {
	t.Helper()

	path := filepath.Join(repoRoot, "internal", "overlay", "finding.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}

	var fields []string
	ast.Inspect(parsed, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok || spec.Name.Name != "Finding" {
			return true
		}
		structType, ok := spec.Type.(*ast.StructType)
		if !ok {
			return false
		}
		for _, field := range structType.Fields.List {
			for _, name := range field.Names {
				if name.IsExported() {
					fields = append(fields, name.Name)
				}
			}
		}
		return false
	})

	if len(fields) == 0 {
		t.Fatalf("no exported fields found on overlay.Finding in %s; the parse found the wrong thing", path)
	}
	slices.Sort(fields)
	return fields
}

// findingFieldsReadHere collects every `finding.<Field>` this package's
// non-test files read.
func findingFieldsReadHere(t *testing.T) map[string]bool {
	t.Helper()

	read := make(map[string]bool)
	for _, file := range packageFiles(t) {
		ast.Inspect(file.ast, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if base, ok := sel.X.(*ast.Ident); ok && base.Name == "finding" {
				read[sel.Sel.Name] = true
			}
			return true
		})
	}
	return read
}

type parsedFile struct {
	name string
	ast  *ast.File
}

// packageFiles is every non-test .go file in this package.
//
// Test files are excluded on purpose: a field a test reads is a field nothing
// SHIPS, and this rule is about what reaches an operator.
func packageFiles(t *testing.T) []parsedFile {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading the package directory: %v", err)
	}

	fset := token.NewFileSet()
	var files []parsedFile
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", name, err)
		}
		files = append(files, parsedFile{name: name, ast: parsed})
	}

	if len(files) == 0 {
		t.Fatal("no non-test .go files found in this package; the walk found the wrong directory")
	}
	return files
}
