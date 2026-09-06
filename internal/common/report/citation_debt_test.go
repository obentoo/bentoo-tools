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
//
// # AMENDED BY SUB-TASK 16.5 — TWO NUMBERS, AND A COMPARISON THAT COMPARED
//
// Two defects, both of them a published number saying less than it appeared to.
//
// The first: a story numbers its DECISIONS as well as its requirements, and
// citationPattern read only the second form. 101 bare decision citations
// across 29 files — every one of them colliding with story 044's design
// numbering exactly as the requirement citations collide with its requirement
// numbering — were counted by nothing. They are counted now, against a ceiling
// of their own rather than folded into the requirement one, because the two
// are repaid by different reading and a single total cannot say which half
// moved.
//
// The second: TestCitationDebtSweepAgreesWithTheGuardsSubject compared len()
// of two maps keyed by path. That is a FILE count. Its own doc comment claimed
// it stopped the debt falling without a fix, and a sweep whose every per-file
// count collapsed to one would have passed it with the file list intact. It
// compares the summed values now, and the key sets, and does both per class.

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
		// The second numbering, added by sub-task 16.5. The predicate is one
		// predicate on purpose: what makes a citation unattributed does not
		// depend on which of a story's two numberings it points at, and a
		// second predicate differing only in a letter is how one class stops
		// being checked. These four are the same four cases as above, in the
		// other form.
		{bareDecision("4"), true, "bare, and 046 issues it — and so does 044, with a different sentence"},
		{bareDecision("7"), true, "bare, and the tree's own prose says this one means story 044's"},
		{"S044-" + bareDecision("7"), false, "prefixed: the citation says which design owns the sentence"},
		{bareDecision("11"), false, "bare and unresolvable — 046 issues ten, so this is a hard failure, not debt"},
	} {
		if got := citationIsUnattributed(tc.citation); got != tc.want {
			t.Errorf("citationIsUnattributed(%q) = %v, want %v — %s", tc.citation, got, tc.want, tc.why)
		}
	}
}

// TestCitationDebtDoesNotGrow is the ceiling — TWO ceilings since sub-task
// 16.5, one per numbering, each pinned and reported on its own.
//
// It logs the per-file breakdown on every run, passing or failing: a number
// with no address is a number nobody can start on, and story 047 inherits
// these lists rather than two totals.
//
// EITHER rise fails. Nothing here adds the two together, and the omission is
// the design: the classes are repaid by different reading — one comment
// checked against story.md, the other against design.md — so a single total
// would let thirty requirement citations repaid pay for thirty decision
// citations added and print a debt that had not moved.
//
// WHAT A COUNTED CEILING DOES NOT DO. It stops the debt rising. It attributes
// nothing. After a green run here, all 457 of these citations still name their
// number and not their story, and a reader meeting any one of them still
// cannot tell whether the sentence behind it is story 046's or story 044's.
// The ceiling's whole claim is that the population is not growing while it
// waits to be read; treating a green run as evidence that the citations
// resolve is the mistake this file exists because someone made once already.
func TestCitationDebtDoesNotGrow(t *testing.T) {
	debt := unattributedCitations(t)

	// The requirement label is empty so that the line this prints is the line
	// sub-task 11.2's recorded transcript quotes, to the byte. A ceiling whose
	// evidence describes a message it no longer prints is evidence of nothing.
	for _, ceiling := range []struct {
		class  citationClass
		label  string
		pinned int
		why    string
	}{
		{
			class:  requirementCitation,
			label:  "",
			pinned: unattributedCitationDebt,
			why: "A bare citation in this package resolves by NUMBER, and story 044's numbering\n" +
				"    covers story 046's entirely — so a new one names a sentence the reader cannot\n" +
				"    find. Prefix it with the story that owns it, in the S0NN- form.",
		},
		{
			class:  decisionCitation,
			label:  "design-decision ",
			pinned: unattributedDecisionDebt,
			why: "A bare decision token resolves by NUMBER exactly as a requirement citation does:\n" +
				"    story 044's design issues the same ten numbers with different sentences behind\n" +
				"    them, and this package cites both — render/fullscreen_sections_test.go already\n" +
				"    means S044-D7 while writing the token bare. Prefix it with the story that owns\n" +
				"    it, in the S0NN- form.",
		},
	} {
		perFile := debt[ceiling.class]
		total := debt.total(ceiling.class)

		files := make([]string, 0, len(perFile))
		for path := range perFile {
			files = append(files, path)
		}
		sort.Strings(files)

		var breakdown strings.Builder
		for _, path := range files {
			fmt.Fprintf(&breakdown, "\n        %-58s %d", path, perFile[path])
		}
		t.Logf("unattributed %scitations remaining in this package: %d, over %d files%s",
			ceiling.label, total, len(files), breakdown.String())

		if total > ceiling.pinned {
			t.Errorf("unattributed %scitations rose to %d, above the pinned %d.\n"+
				"    %s\n"+
				"    Per file:%s", ceiling.label, total, ceiling.pinned, ceiling.why, breakdown.String())
		}
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
//
// It DUPLICATES the walk in citation_test.go on purpose, and the duplication is
// the check rather than an oversight in it. Sub-task 16.5 unified everything
// that could be unified without destroying the test: one pattern, one
// predicate, one class function, one debt type — so the two sweeps now differ
// only in the walk itself, which is the single thing a cross-check must not
// share with its subject. Folding this into eachCitationInPackage would leave
// TestCitationDebtSweepAgreesWithTheGuardsSubject comparing a function with
// itself, which passes on any tree and reports nothing.
func unattributedCitationsFromWalk(t *testing.T) citationDebt {
	t.Helper()

	fileSet := token.NewFileSet()
	debt := citationDebt{}
	filesScanned := 0
	storyFilesScanned := 0

	read := func(path string) error {
		file, parseErr := parser.ParseFile(fileSet, path, nil, parser.ParseComments|parser.SkipObjectResolution)
		if parseErr != nil {
			return fmt.Errorf("parsing %s: %w", path, parseErr)
		}
		filesScanned++

		count := func(text string) {
			for _, ref := range citationsIn(text) {
				if citationIsUnattributed(ref.text) {
					debt.add(classOf(ref.text), path)
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
	}

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
		return read(path)
	})
	if err != nil {
		t.Fatalf("sweeping the package for citations: %v", err)
	}

	// The half sub-task 16.6 added, read here too because this sweep's whole
	// job is to be the ceiling's subject checked against a second reader. The
	// LIST is shared — it is data, and two copies of it would drift into two
	// different subjects, which is the divergence this file exists to catch —
	// while the READING stays duplicated, which is the one thing a cross-check
	// must not share with the thing it checks.
	for _, path := range storyCreatedFilesOutsideThePackage {
		if err := read(path); err != nil {
			t.Fatalf("sweeping the files this story created outside this package: %v", err)
		}
		storyFilesScanned++
	}

	// A walk that read nothing returns an empty debt, and an empty debt agrees
	// with anything the cross-check compares it to. The guard counts FILES and
	// not hits for the reason unattributedCitations gives: zero hits is what a
	// finished repayment looks like, and story 047's success must not read as
	// this sweep's failure.
	if filesScanned == 0 {
		t.Fatal("scanned no Go file — the independent sweep returned its counts without looking at anything")
	}
	if storyFilesScanned == 0 {
		t.Fatalf("read none of the %d files this story created outside this package — this sweep would then agree with a ceiling that had stopped reading them too, and the cross-check would pass over a subject two sizes smaller than the one the ceilings were measured on",
			len(storyCreatedFilesOutsideThePackage))
	}
	return debt
}

// TestCitationDebtSweepAgreesWithTheGuardsSubject pins the two sweeps to the
// same tree, in BOTH of the ways they can part company.
//
// The sentence this check has always claimed: if the ceiling ever reads a
// narrower subject than TestCitationsResolve, the debt can fall without
// anything being fixed. Until sub-task 16.5 the check did not cover it. It read
//
//	if got, want := ...; len(got) != len(want)
//
// on two maps keyed by PATH, so what it compared was a FILE count. A file count
// is not a debt: let every per-file count fall from twelve to one and the key
// set is untouched, both lengths still read 36, and the check passes while the
// published number drops by three hundred. That mutation is recorded — with the
// old comparison passing it and this one failing it — in citation_test.go's
// evidence section for sub-task 16.5.
//
// WHICH OF THE TWO THE SENTENCE NOW COVERS: both, per class.
//
//   - The KEY SET — WHICH files carry debt. A file one sweep stops reading is
//     a file whose debt leaves one of the two totals. Checked in both
//     directions, because a file only the ceiling reads is as much a
//     divergence as one only the independent walk reads.
//   - The SUMMED VALUES — HOW MUCH debt those files carry. This is the half
//     len() could not see, and it is the half a narrowing sweep moves first:
//     narrowing what is COUNTED inside a file leaves the file list intact.
//
// Per class rather than over the pair, because the two classes have separate
// ceilings. A requirement total that agreed while the decision totals diverged
// would leave half of what this suite publishes unchecked, which is the same
// defect one level down.
func TestCitationDebtSweepAgreesWithTheGuardsSubject(t *testing.T) {
	got, want := unattributedCitations(t), unattributedCitationsFromWalk(t)

	for _, class := range []citationClass{requirementCitation, decisionCitation} {
		gotFiles, wantFiles := got[class], want[class]

		for path := range gotFiles {
			if _, read := wantFiles[path]; !read {
				t.Errorf("the ceiling counts %s debt in %s and this file's independent sweep counts none there — "+
					"the two must read the same subject or the number can fall without a fix", class, path)
			}
		}
		for path := range wantFiles {
			if _, read := gotFiles[path]; !read {
				t.Errorf("this file's independent sweep counts %s debt in %s and the ceiling counts none there — "+
					"the ceiling is the narrower of the two, which is how a published debt falls without a fix", class, path)
			}
		}

		if gotTotal, wantTotal := got.total(class), want.total(class); gotTotal != wantTotal {
			t.Errorf("the ceiling counts %d unattributed %s citations and this file's independent sweep %d, "+
				"across file lists that agree — the debt one of them publishes is not the debt that is there, "+
				"and comparing the number of files would not have noticed", gotTotal, class, wantTotal)
		}
	}
}
