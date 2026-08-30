package report

// Authored for story 046, sub-task 11.2 — S046-R8.3.
//
// The rule 10.4 adopted: "leave bare citations that DO resolve to a 046
// requirement untouched — inside this story's own code they are unambiguous."
// That premise is false in this package, and this file is where the falsehood
// is counted instead of assumed away.
//
// Story 046 issues R1.1 through R8.4. Story 044 — which wrote most of this
// package — issues R1.1 through S044-R10.5. Every number 046 has, 044 also has,
// with a different sentence behind it. So a bare citation here that passes
// citationFault by membership in story046Requirements has been matched on its
// NUMBER, not on its MEANING, which is the exact confusion 10.4's own ToDo
// names as "the defect this sub-task closes".
//
// It is not hypothetical. section.go is a file story 046 created, and its bare
// R7.2 and R7.4 mean 044's "do not print it a second time" and "preserve every
// reason in full" — 046's R7.2 is the presentation-field guard and its R7.4 is
// "adding a kind edits no renderer". Neither is what the comment is about.
//
// # What this file does, and what it deliberately does not
//
// It does NOT prefix anything. Attributing these citations means reading each
// comment and deciding which story's sentence it meant; done mechanically it
// would produce confident falsehoods, which is worse than a bare number
// because it looks resolved. That sweep is story 047's.
//
// It publishes the debt as a number that cannot grow, which is the shape
// Constraint 8 asks for and the shape TestWidthDebt already uses one directory
// over. The value of a ceiling here is precise: before it, a green
// TestCitationsResolve asserted this package's citations were resolvable, and
// the half it could not check was invisible BECAUSE the guard was green.
//
// # RED ON ARRIVAL
//
// unattributedCitationDebt, citationIsUnattributed and unattributedCitations
// do not exist.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestCitationIsUnattributedSeparatesMatchedFromMeant is the unit half: the
// counter's rule, stated on inputs rather than on the tree.
//
// The third case is the one that makes the class real. A prefixed citation is
// attributed whether or not 046 owns the number, because the prefix is the
// answer to "which story" — that is what prefixing is FOR, and re-litigating
// it here would make the counter disagree with citationFault.
func TestCitationIsUnattributedSeparatesMatchedFromMeant(t *testing.T) {
	for _, tc := range []struct {
		citation string
		want     bool
		why      string
	}{
		{"R7.2", true, "bare, and 046 has it — matched on the number, and 044 has R7.2 too"},
		{"R1.1", true, "bare, and 046 has it — same collision, at the top of both sets"},
		{"S044-R7.2", false, "prefixed: the citation says which story owns the sentence"},
		{"S046-R7.2", false, "prefixed: attributed, even though the prefix is this story's own"},
		{bareID("9.3"), false, "bare and unresolvable — already a hard failure, not debt"},
	} {
		if got := citationIsUnattributed(tc.citation); got != tc.want {
			t.Errorf("citationIsUnattributed(%q) = %v, want %v — %s", tc.citation, got, tc.want, tc.why)
		}
	}
}

// TestCitationDebtDoesNotGrow is the ceiling.
//
// It logs the per-file breakdown on every run, passing or failing: a number
// with no address is a number nobody can start on, and story 047 inherits this
// list rather than a total.
func TestCitationDebtDoesNotGrow(t *testing.T) {
	perFile := unattributedCitations(t)

	total := 0
	files := make([]string, 0, len(perFile))
	for path, n := range perFile {
		total += n
		files = append(files, path)
	}
	sort.Strings(files)

	var breakdown strings.Builder
	for _, path := range files {
		fmt.Fprintf(&breakdown, "\n        %-58s %d", path, perFile[path])
	}
	t.Logf("unattributed citations remaining in this package: %d, over %d files%s",
		total, len(files), breakdown.String())

	if total > unattributedCitationDebt {
		t.Errorf("unattributed citations rose to %d, above the pinned %d.\n"+
			"    A bare citation in this package resolves by NUMBER, and story 044's numbering\n"+
			"    covers story 046's entirely — so a new one names a sentence the reader cannot\n"+
			"    find. Prefix it with the story that owns it, in the S0NN- form.\n"+
			"    Per file:%s", total, unattributedCitationDebt, breakdown.String())
	}
}

// unattributedCitationsFromWalk is the sweep this file's ceiling reads, written
// here so the test states its own subject: every .go file under this package,
// render/ included, comments and string literals alike — the same subject
// TestCitationsResolve sweeps, because a narrower one would let the debt hide
// in the difference.
//
// The guard's own requirement table is the one exclusion. Those string literals
// DEFINE story 046's set; counting them as citations OF it would pin a ceiling
// against the definition and make the number meaningless.
func unattributedCitationsFromWalk(t *testing.T) map[string]int {
	t.Helper()

	fileSet := token.NewFileSet()
	perFile := map[string]int{}

	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if filepath.Base(path) == "citation_test.go" {
			return nil
		}

		file, parseErr := parser.ParseFile(fileSet, path, nil, parser.ParseComments|parser.SkipObjectResolution)
		if parseErr != nil {
			return fmt.Errorf("parsing %s: %w", path, parseErr)
		}

		count := func(text string) {
			for _, ref := range citationsIn(text) {
				if citationIsUnattributed(ref.text) {
					perFile[path]++
				}
			}
		}
		for _, group := range file.Comments {
			for _, comment := range group.List {
				count(comment.Text)
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if literal, ok := node.(*ast.BasicLit); ok && literal.Kind == token.STRING {
				count(literal.Value)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("sweeping the package for citations: %v", err)
	}
	return perFile
}

// TestCitationDebtSweepAgreesWithTheGuardsSubject pins the two sweeps to the
// same tree. If the ceiling ever reads fewer files than TestCitationsResolve,
// the debt can fall without anything being fixed.
func TestCitationDebtSweepAgreesWithTheGuardsSubject(t *testing.T) {
	if got, want := unattributedCitations(t), unattributedCitationsFromWalk(t); len(got) != len(want) {
		t.Errorf("the ceiling sweeps %d files, this file's independent sweep %d — "+
			"the two must read the same subject or the number can fall without a fix", len(got), len(want))
	}
}
