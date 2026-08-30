package report

// Authored for story 046, sub-task 10.4 — S046-R8.3.
//
// The rule: no comment and no string under internal/common/report, render/
// included, may cite a requirement number that story 046 does not have without
// saying which story does. A number 046 never issued is unresolvable: the same
// token exists in a dozen other stories in this repository, so a reader meeting
// it has no way to find the sentence it points at. .epic/ is not committed
// (`git ls-files .epic/` returns nothing), which is what makes this a code
// problem rather than a bookkeeping one — the comment is the whole record.
//
// A bare number that DOES resolve inside 046 is not refused here. It is not
// resolved either, and the section titled "RESOLVING IS NOT ATTRIBUTING" below
// is the correction — read it before trusting a green run of this guard.
//
// The disambiguation already exists and is used several hundred times across
// the repository: an S0NN- prefix naming the owning story. This guard requires
// it exactly where a bare citation cannot be resolved, and nowhere else.
//
// # RESOLVING IS NOT ATTRIBUTING — amended by sub-task 11.2 (S046-R8.3)
//
// This file was written with a second claim beside the rule, and the second
// claim was wrong. It read: "A bare number that resolves inside 046 is fine —
// it is unambiguous where it stands." That holds in a package story 046 wrote
// from nothing. This is not one. Story 044 wrote most of it and issues
// S044-R1.1 through S044-R10.5, so every number 046 has, 044 has too, with a
// different sentence behind it. A bare citation here that citationFault accepts
// has therefore been matched on its NUMBER, not on its MEANING.
//
// It is not hypothetical, and the collision is not confined to code 046
// inherited. section.go is a file story 046 CREATED, and line 84 of it says a
// repeated reason is "printed once (R7.2), while the report itself still
// carries the whole string (R7.4)". Those two are S044-R7.2 and S044-R7.4:
// 046's own R7.2 is the presentation-field guard and its R7.4 is "adding a kind
// edits no renderer". Neither is what the comment is about, and this guard
// called both of them resolved.
//
// citationFault is UNCHANGED by the amendment. A bare-but-resolvable citation
// is still not a hard failure, because 356 of them are in this package and
// failing all 356 would fail the package rather than fix it. What changed is
// that the class stopped being silent: citation_debt_test.go counts it, prints
// the per-file breakdown on every run, and fails when it grows.
// unattributedCitationDebt at the foot of this file is the ceiling, and the
// evidence that the ceiling bites is recorded beside it. Deciding which story
// each of the 356 meant is story 047's work — done mechanically it would write
// confident falsehoods, and a wrong prefix is worse than a bare number because
// it looks resolved.
//
// # RED ON ARRIVAL — 79 of them
//
// This guard is not green on the day it is written. It finds 79 unresolvable
// citations across 21 files under this package, all of them numbers issued by
// story 044 and carried over with the code: 12 distinct requirements, R-9.3
// eighteen times, R-2.5 thirteen, R-6.4 seven, and so on down. Sub-task 10.4's
// work is to prefix exactly those, and this file is the list.
//
//	$ go test ./internal/common/report/ -run TestCitationsResolve
//	--- FAIL: TestCitationsResolve (0.01s)
//	    citation_test.go:457: autoupdate_check.go:48 cites R-9.3 — story 046 has
//	        no R-9.3, and the citation is bare, so it names nothing a reader can
//	        resolve.
//	        [... 78 more, one per occurrence ...]
//	    citation_test.go:491: 79 of the 471 citations read across 37 files
//	        cannot be resolved
//
// # EVIDENCE THAT IT STILL FAILS ONCE IT IS GREEN (S046-R8.3)
//
// The paragraph above stops being checkable the moment 10.4 finishes: from then
// on this guard passes, and a green test proves nothing about what it would
// catch. S046-R8.3 asks for the evidence a green guard cannot supply, and the
// story artifacts holding it (.draft/red-evidence.yaml) are not committed — so
// a pointer to that file is a pointer to nothing for anyone who cloned this. It
// is written down here instead, the way boundary_fields_test.go writes down its
// own.
//
// Measured on 2026-08-29. The finished state was simulated first, so that each
// mutation could be seen on its own rather than as an eightieth line in a list
// of seventy-nine: every offending citation in the package was prefixed with
// S044- in place, which took the guard to green — the sweep is not failing
// vacuously, it goes green exactly when the rule holds. Three mutations were
// then applied against that green state, one at a time, each run and reverted
// immediately, and the whole package was restored with `git checkout -- ` and
// the restore verified by an empty `git status --porcelain`.
//
// A note on the transcripts: a requirement number is written below with a
// hyphen after the R — R-9.3 for what the run printed without that hyphen —
// because this file is swept by its own rule and quoting a bare citation
// verbatim would make it violate the thing it checks. The hyphen is the only
// edit; every other character is as it was printed.
//
// Mutation 1 — a bare citation, in a comment, in mode.go:
//
//	mode.go:118 cites R-9.3 — story 046 has no R-9.3, and the citation is bare,
//	    so it names nothing a reader can resolve.
//	        Remedy: prefix the citation with the story that owns the requirement,
//	        in the S0NN- form used throughout this repository. The story files
//	        under .epic/ are not committed, so this comment is the only record of
//	        what the code was answering.
//	1 of the 470 citations read across 37 files cannot be resolved
//
// Mutation 2 — this story's own prefix on a number this story does not have,
// in model.go. It is not a repetition of the first: the prefix is the remedy's
// own invention, so the rule that motivated it cannot name what would collide
// with it, and prefixing every offender with the story being worked on is the
// cheapest way to satisfy mutation 1 while asserting something false.
//
//	model.go:89 cites S046-R-9.3 — the citation claims story 046 owns R-9.3, and
//	    story 046 has no R-9.3.
//	        [the Remedy paragraph above, word for word]
//	1 of the 470 citations read across 37 files cannot be resolved
//
// Mutation 3 — a citation inside a string rather than a comment, in
// mode_test.go's failure message. Five of the 79 found on arrival live in that
// position, and a sweep that read only comments would have left them.
//
//	mode_test.go:123 cites R-6.4 — story 046 has no R-6.4, and the citation is
//	    bare, so it names nothing a reader can resolve.
//	        [the Remedy paragraph above, word for word]
//	1 of the 470 citations read across 37 files cannot be resolved
//
// Each names the file, the line, the number and what to do about it, which is
// what makes the failure answerable by someone who did not write this file.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// story046Requirements is story 046's requirement set, written out rather than
// read from .epic/stories/046-report-across-the-cli/story.md.
//
// The file is not committed, so a guard that parsed it would pass vacuously —
// or fail spuriously — for everyone who cloned this repository. Copying the set
// into the guard is what gives the rule an existence independent of the story
// directory, which is the same reason the citations themselves have to be
// resolvable from the code alone.
//
// A copied set drifts, and this one did. R-3.6 was added to story.md by its
// third refinement, months after sub-task 10.4 transcribed the table, and the
// omission was found by sub-task 11.2 rather than by anything here: no bare
// citation of it exists in this package, so the guard had nothing to be wrong
// about yet. It would have been wrong the moment one appeared — a real
// requirement reported as a number story 046 never issued. Anyone adding a
// requirement to story 046 adds it here too, and the count below is 34.
// (Written R-3.6, with the hyphen, for the reason the header gives: this file
// is swept by its own rule.)
var story046Requirements = map[string]bool{
	"R1.1": true, "R1.2": true, "R1.3": true, "R1.4": true, "R1.5": true, "R1.6": true,
	"R2.1": true, "R2.2": true, "R2.3": true, "R2.4": true,
	"R3.1": true, "R3.2": true, "R3.3": true, "R3.4": true, "R3.5": true, "R3.6": true,
	"R4.1": true, "R4.2": true, "R4.3": true, "R4.4": true,
	"R5.1": true, "R5.2": true, "R5.3": true,
	"R6.1": true, "R6.2": true, "R6.3": true,
	"R7.1": true, "R7.2": true, "R7.3": true, "R7.4": true,
	"R8.1": true, "R8.2": true, "R8.3": true, "R8.4": true,
}

// thisStoryPrefix is the prefix that claims a citation belongs to story 046.
const thisStoryPrefix = "S046-"

// citationPattern reads a requirement citation, with its story prefix when it
// has one.
//
// The leading character class is the boundary, and it is load-bearing in both
// directions. Without it the pattern would find a citation inside an identifier
// that merely ends in R followed by digits, and — worse — it would read the
// tail of a prefixed citation as a bare one, turning every correctly
// disambiguated reference in the tree into a violation.
//
// Both number parts are greedy for the converse reason: a citation is the whole
// number it is written as. Reading a prefix of it would collapse two distinct
// requirements into one and report a violation against a requirement nobody
// cited.
var citationPattern = regexp.MustCompile(`(?:^|[^0-9A-Za-z_])((?:S[0-9]{3}-)?R[0-9]+\.[0-9]+)`)

// citationRef is one citation and where it sits inside the text it was read
// from, counted in lines from the start of that text.
//
// The offset is not decoration. A doc comment in this package runs to fifty
// lines, and a failure that named only the comment's first line would send its
// reader to a paragraph and leave them to search it.
type citationRef struct {
	text string
	line int
}

// citationsIn reads every citation in a block of text.
func citationsIn(text string) []citationRef {
	var refs []citationRef
	for _, match := range citationPattern.FindAllStringSubmatchIndex(text, -1) {
		start, end := match[2], match[3]
		refs = append(refs, citationRef{
			text: text[start:end],
			line: strings.Count(text[:start], "\n"),
		})
	}
	return refs
}

// splitCitation separates a citation's story prefix from its requirement id.
// An empty prefix means the citation was written bare.
func splitCitation(citation string) (prefix, id string) {
	if hyphen := strings.IndexByte(citation, '-'); hyphen >= 0 {
		return citation[:hyphen+1], citation[hyphen+1:]
	}
	return "", citation
}

// citationFault answers why a citation cannot be resolved, or "" when it can.
//
// Three outcomes, and the third is the one a reader is most likely to miss:
//
//   - Bare, and 046 has the requirement: not a fault here — and not resolved
//     either. Story 044 issues the same numbers, so the match is on the digits
//     rather than on the sentence. It stays outside the hard-failure class
//     because 356 of them are in this package and refusing all 356 at once
//     would fail the package rather than fix it; sub-task 11.2 gave the class a
//     counted ceiling instead. citationIsUnattributed at the foot of this file
//     is that class, and the header section "RESOLVING IS NOT ATTRIBUTING" is
//     why it exists.
//   - Bare, and 046 does not: the fault this guard exists for.
//   - Prefixed S046-, and 046 does not have the requirement: also a fault. The
//     prefix is the fix's own invention, so nothing in the rule above forbids
//     spelling it wrong, and the cheapest way to silence a bare-citation guard
//     is to prefix every offender with the story being worked on rather than
//     the story that owns the requirement. That produces a citation which is
//     confidently, checkably false — worse than the bare one, because it looks
//     resolved.
//
// A prefix naming any OTHER story passes unchecked. This guard knows one
// story's requirement set and cannot audit S044- or S033- without reading
// artifacts that are not committed; the prefix is the disambiguation, and
// checking that it points at a real sentence is a reader's job.
func citationFault(citation string) string {
	prefix, id := splitCitation(citation)

	switch {
	case prefix == "" && !story046Requirements[id]:
		return fmt.Sprintf("story 046 has no %s, and the citation is bare, so it names nothing a reader can resolve", id)
	case prefix == thisStoryPrefix && !story046Requirements[id]:
		return fmt.Sprintf("the citation claims story 046 owns %s, and story 046 has no %s", id, id)
	}
	return ""
}

// citationViolation is the sentence a maintainer meets. It names the file, the
// line, the citation and what to do, because a guard whose message stops at
// "unresolvable citation" is a guard that gets deleted rather than answered.
func citationViolation(path string, line int, citation, fault string) string {
	return fmt.Sprintf(
		"%s:%d cites %s — %s.\n"+
			"    Remedy: prefix the citation with the story that owns the requirement, in the S0NN- form used\n"+
			"    throughout this repository. The story files under .epic/ are not committed, so this comment is\n"+
			"    the only record of what the code was answering.",
		path, line, citation, fault)
}

// bareID spells a citation out of parts, so that a fixture in this file is not
// itself a citation this file's own sweep would find.
//
// The guard reads string literals as well as comments, and it does not exempt
// itself — an exemption would be a hole exactly where someone tempted to hide a
// citation would put one. Writing the hostile fixtures whole would therefore
// make this file fail against itself. Assembling them at run time keeps the
// fixture and the rule in the same file without either one lying about the
// other.
func bareID(number string) string { return "R" + number }

// TestCitationRule pins the classifier, hostile cases first.
//
// The order is the point. Written after the tree had been cleaned, a case
// asking "does a good citation pass?" would pass against a classifier that
// approves everything, and a sweep built on that classifier would report a
// clean package forever. The cases that must FIRE come first, and each names
// the way the rule could be satisfied wrongly.
func TestCitationRule(t *testing.T) {
	cases := []struct {
		name     string
		citation string
		fault    bool
		why      string
	}{
		// Hostile: the rule must fire. The number below belongs to story 044
		// (S044-R9.3, written here with the prefix this guard asks for) and it
		// is the most-cited of the borrowed numbers in this package.
		{
			name:     "bare citation of a requirement 046 does not have",
			citation: bareID("9.3"),
			fault:    true,
			why:      "a number 046 never issued, written as though it were 046's",
		},
		// Hostile: the same, for a number that exists in many stories at once.
		{
			name:     "bare citation of a number many stories share",
			citation: bareID("2.5"),
			fault:    true,
			why:      "sixteen story files in this repository define one; bare, it names none of them",
		},
		// Hostile, the converse: the rule must NOT fire. Over-firing here would
		// condemn every correct citation in the package and cost a rewrite of
		// twenty-five files that buys nothing.
		{
			name:     "bare citation of a requirement 046 does have",
			citation: "R7.2",
			fault:    false,
			why:      "not refused: 046 has R7.2, so this class is counted as debt rather than failed",
		},
		// Hostile, the third element: the prefix is a spelling the fix invents,
		// so the rule that motivated it cannot name what would collide with it.
		// An S046- prefix on a number 046 does not have is the collision — it
		// silences a bare-citation sweep while asserting something false.
		{
			name:     "this story's prefix on a requirement this story does not have",
			citation: thisStoryPrefix + bareID("9.3"),
			fault:    true,
			why:      "prefixing every offender with the story being worked on is the cheapest wrong fix",
		},
		// Correct disambiguation, and the reason the guard exists.
		{
			name:     "the owning story's prefix",
			citation: "S044-" + bareID("9.3"),
			fault:    false,
			why:      "the reader can find the sentence: story 044 issued it",
		},
		{
			name:     "another story's prefix",
			citation: "S033-" + bareID("9.5"),
			fault:    false,
			why:      "this guard knows one story's set and does not audit another's",
		},
		{
			name:     "this story's prefix on one of its own requirements",
			citation: thisStoryPrefix + "R5.1",
			fault:    false,
			why:      "the form already used twenty-three times in this repository",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fault := citationFault(testCase.citation)

			if testCase.fault && fault == "" {
				t.Errorf("%s was accepted — %s", testCase.citation, testCase.why)
			}
			if !testCase.fault && fault != "" {
				t.Errorf("%s was rejected as %q — %s", testCase.citation, fault, testCase.why)
			}
		})
	}
}

// TestCitationScannerReadsWholeCitations pins the reader the classifier is fed
// from, hostile cases first.
//
// Every case here is a way for two different citations to arrive at the same
// answer. A scanner that collapses them would either invent violations against
// requirements nobody cited, or read a correctly prefixed reference as a bare
// one and condemn the very form the remedy asks for.
func TestCitationScannerReadsWholeCitations(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		// Hostile: a longer number must not collapse onto the shorter citation
		// it starts with. A two-digit sub-number is its own requirement, and a
		// scanner that read only the first digit would report a violation
		// against a number nobody wrote. The fixture is assembled rather than
		// written out for the reason bareID gives.
		//
		// The pair is the whole hostile case here: these two numbers differ by
		// one character and must stay apart, in both directions — the long one
		// must not be read as the short one, and the short one, cited on its
		// own elsewhere in this table, must still be found.
		{
			name: "a longer sub-number is not the shorter one it starts with",
			text: "see " + bareID("2.55") + " for the detail",
			want: []string{bareID("2.55")},
		},
		// Hostile, the converse: a prefixed citation must not split into a bare
		// one. This is the failure that would turn every correct citation in the
		// tree into a violation the moment the remedy was applied.
		{
			name: "a prefixed citation is one citation, not a bare tail",
			text: "as " + thisStoryPrefix + "R5.1 requires",
			want: []string{thisStoryPrefix + "R5.1"},
		},
		// Hostile: an identifier that merely ends in R plus digits is not a
		// citation. A scanner without a left boundary finds one inside it.
		{
			name: "a citation inside a word is not a citation",
			text: "the CHAR2.5 column and the VAR7.2 binding",
			want: nil,
		},
		{
			name: "a citation at the very start of the text",
			text: bareID("9.3") + " is answered here",
			want: []string{bareID("9.3")},
		},
		{
			name: "adjacent citations are read separately",
			text: "R1.1," + bareID("9.3") + " and " + "S044-" + bareID("5.4"),
			want: []string{"R1.1", bareID("9.3"), "S044-" + bareID("5.4")},
		},
		{
			name: "prose without a citation",
			text: "the report is rendered in four modes",
			want: nil,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var got []string
			for _, ref := range citationsIn(testCase.text) {
				got = append(got, ref.text)
			}

			if strings.Join(got, " ") != strings.Join(testCase.want, " ") {
				t.Errorf("read %v from %q, want %v", got, testCase.text, testCase.want)
			}
		})
	}
}

// TestCitationScannerCountsLinesWithinAText pins the offset, because a doc
// comment is one node and the citation inside it can be forty lines down.
func TestCitationScannerCountsLinesWithinAText(t *testing.T) {
	text := "// first line\n// second line\n// third line cites " + bareID("9.3")

	refs := citationsIn(text)
	if len(refs) != 1 {
		t.Fatalf("read %d citations from the block, want 1", len(refs))
	}
	if refs[0].line != 2 {
		t.Errorf("the citation was reported %d lines into the block, want 2 — a failure that names the block's first line sends its reader to a paragraph", refs[0].line)
	}
}

// TestCitationViolationNamesFileAndCitation pins the message. A guard that
// fails without naming the file and the offending number costs its reader a
// search through twenty-five files, which is how a mechanical rule turns back
// into a convention nobody follows.
func TestCitationViolationNamesFileAndCitation(t *testing.T) {
	citation := bareID("9.3")

	fault := citationFault(citation)
	if fault == "" {
		t.Fatalf("%s is not classified as a fault — its message cannot be checked", citation)
	}

	message := citationViolation("mode.go", 42, citation, fault)

	for _, want := range []string{"mode.go", "42", citation, "Remedy", "S0NN-"} {
		if !strings.Contains(message, want) {
			t.Errorf("the failure message does not name %q:\n%s", want, message)
		}
	}
}

// eachCitationInPackage reads every citation this package writes and hands each
// one to visit: the file it stands in, the line it stands on, and the citation
// itself. It returns the number of files it parsed.
//
// This is the SUBJECT, written once and shared by both sweeps below. Every .go
// file under internal/common/report, render/ and the _test.go files included —
// a citation in a test's failure message is read by the same person with the
// same question, and five of the 79 violations this guard found on arrival
// lived inside strings, so literals are read alongside comments.
//
// skip is the ONLY difference between the guard and the ceiling, and making it
// an argument is what keeps the two from drifting apart. Before this, each
// sweep declared the subject for itself, so narrowing one of them would have
// gone unnoticed by the other — and a ceiling reading fewer files than the
// guard reports a debt that fell without anything being fixed. A nil skip
// reads everything.
//
// The returned count is what each caller's vacuity guard is built on: a sweep
// that read nothing finds no violation and no debt, and both of those look
// exactly like success.
func eachCitationInPackage(t *testing.T, skip func(path string) bool, visit func(path string, line int, citation string)) int {
	t.Helper()

	fileSet := token.NewFileSet()
	filesScanned := 0

	walkErr := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if skip != nil && skip(path) {
			return nil
		}

		file, parseErr := parser.ParseFile(fileSet, path, nil, parser.ParseComments|parser.SkipObjectResolution)
		if parseErr != nil {
			return fmt.Errorf("parsing %s: %w", path, parseErr)
		}
		filesScanned++

		inspect := func(pos token.Pos, text string) {
			base := fileSet.Position(pos).Line
			for _, ref := range citationsIn(text) {
				visit(path, base+ref.line, ref.text)
			}
		}

		for _, group := range file.Comments {
			for _, comment := range group.List {
				inspect(comment.Pos(), comment.Text)
			}
		}

		ast.Inspect(file, func(node ast.Node) bool {
			if literal, ok := node.(*ast.BasicLit); ok && literal.Kind == token.STRING {
				inspect(literal.Pos(), literal.Value)
			}
			return true
		})

		return nil
	})
	if walkErr != nil {
		t.Fatalf("sweeping the package: %v", walkErr)
	}

	return filesScanned
}

// TestCitationsResolve is the guard itself: every citation written in a comment
// or a string anywhere under internal/common/report, render/ included, must
// name a sentence a reader can find.
//
// It exempts nothing, this file included — an exemption would be a hole exactly
// where someone tempted to hide a citation would put one.
func TestCitationsResolve(t *testing.T) {
	citationsRead := 0
	violations := 0

	filesScanned := eachCitationInPackage(t, nil, func(path string, line int, citation string) {
		citationsRead++
		if fault := citationFault(citation); fault != "" {
			violations++
			t.Errorf("%s", citationViolation(path, line, citation, fault))
		}
	})

	// A sweep that read nothing passes, and would keep passing after the
	// citations were moved out from under it.
	if filesScanned == 0 {
		t.Fatal("scanned no Go file — the sweep passed without looking at anything")
	}
	if citationsRead == 0 {
		t.Fatal("read no citation at all — this package cites requirements in almost every file, so the reader is broken, not the package")
	}

	if violations > 0 {
		t.Logf("%d of the %d citations read across %d files cannot be resolved", violations, citationsRead, filesScanned)
	}
}

// # THE SECOND CLASS — WHAT RESOLVES WITHOUT BEING ATTRIBUTED (sub-task 11.2)
//
// Everything below serves the ceiling in citation_debt_test.go, and it lives in
// THIS file for a reason that is not tidiness. The debt sweep excludes
// citation_test.go by name: story046Requirements above DEFINES story 046's set,
// and counting the definition as a citation OF the set would pin a ceiling
// against the definition. Putting the counter inside the one excluded file
// keeps both sweeps reading an identical subject with a single exclusion rule
// to maintain, and keeps the counter's own explanation — which cites
// requirements in almost every sentence — out of the number it reports. A
// helper in a third file would sit inside the subject it measures.

// requirementTableFile is the one file the debt sweep does not read: this one,
// where the requirement table lives. citation_debt_test.go's independent walk
// names the same basename, and TestCitationDebtSweepAgreesWithTheGuardsSubject
// fails if the two subjects ever part company.
const requirementTableFile = "citation_test.go"

// isRequirementTable is that exclusion as eachCitationInPackage takes it.
func isRequirementTable(path string) bool { return filepath.Base(path) == requirementTableFile }

// citationIsUnattributed answers whether a citation was matched on its NUMBER
// rather than on its MEANING: written bare, and let through by citationFault
// only because story 046 issues that number too — as does story 044, which
// wrote most of this package.
//
// It is written THROUGH citationFault rather than beside it so that the two
// classes cannot drift into overlapping. Whatever citationFault rejects is a
// hard failure and is never counted as debt; what this counts is exactly the
// residue citationFault lets through bare. A prefixed citation is attributed by
// definition, whichever story the prefix names — that is what a prefix is FOR,
// and re-deciding it here would put the counter and the guard at odds.
func citationIsUnattributed(citation string) bool {
	prefix, _ := splitCitation(citation)
	return prefix == "" && citationFault(citation) == ""
}

// unattributedCitationDebt is the ceiling, and it is a MEASUREMENT rather than
// a target: 356 bare citations over 36 of the 37 files the sweep reads, taken
// on 2026-08-29 against the tree exactly as it is committed. Nothing was picked
// to make the number fit — it is what the sweep printed on the day it landed,
// and classify_test.go is the single swept file with none.
//
// It may not rise. That trigger is the thing story 044 lacked when it recorded
// three typed column widths as debt "to be repaid when one of them is next
// touched": none were touched, and the count had reached 18 by the time anyone
// looked (cmd/bentoo/width_debt_test.go carries that history, and this ceiling
// is built in its shape). A number that can only fall is the hand-off
// Constraint 8 asks for — story 047 can CHECK it instead of believing a
// description of it.
//
// It will not fall by itself. Repaying it means reading 356 comments and
// deciding which story's sentence each one meant; done mechanically it would
// write confident falsehoods, and a wrong prefix is worse than a bare number
// because it looks resolved. Sub-task 11.2 therefore prefixes nothing and hands
// the reading to story 047.
const unattributedCitationDebt = 356

// # EVIDENCE THAT THE CEILING BITES, AND THAT IT LEFT THE GUARD ALONE (S046-R8.3)
//
// The pin above is green on the day it lands, and a green ceiling proves
// nothing about what it would catch — the same hole this file's header
// identifies for TestCitationsResolve and closes there. The artifacts holding
// the Red (.draft/red-evidence.yaml) are not committed, so a pointer to them is
// a pointer to nothing for anyone who cloned this. It is written here instead.
//
// Measured on 2026-08-29. Two mutations, applied to mode.go one at a time from
// a copy taken first, each run and reverted immediately, the restore verified
// by `git status --porcelain` reporting no change to mode.go and by a rerun
// going green. Both transcripts are re-wrapped to this comment's width and the
// long breakdowns are elided where marked; no other character is edited, and
// the single exception to that is named at the foot.
//
// Mutation 1 — ONE bare citation of a number story 046 DOES issue, added as a
// comment at the end of mode.go. This is the class the ceiling counts:
//
//	=== RUN   TestCitationDebtDoesNotGrow
//	    citation_debt_test.go:97: unattributed citations remaining in this package: 357, over 36 files
//	        [... 7 rows unchanged ...]
//	        mode.go                                                    12
//	        [... 28 rows unchanged ...]
//	    citation_debt_test.go:101: unattributed citations rose to 357, above the pinned 356.
//	        A bare citation in this package resolves by NUMBER, and story 044's numbering
//	        covers story 046's entirely — so a new one names a sentence the reader cannot
//	        find. Prefix it with the story that owns it, in the S0NN- form.
//	        Per file:
//	        [the same 36 rows, mode.go again at 12]
//	--- FAIL: TestCitationDebtDoesNotGrow (0.01s)
//
// mode.go stood at 11 before the mutation and at 12 after, and that is what
// makes the breakdown the answerable half: the total says the debt grew, the
// list says which file to open. The elided rows are identical in both listings
// and identical to the ones a green run prints.
//
// Mutation 2 — ONE bare citation of a number story 046 does NOT have, in the
// same position. It is not a repetition of the first. It is the demonstration
// that the two classes stayed disjoint once this file grew a second one: the
// ceiling must NOT move, and the hard guard must refuse the citation exactly as
// it did before sub-task 11.2 touched anything. Both tests were run together,
// in one command, against one mutated tree:
//
//	=== RUN   TestCitationDebtDoesNotGrow
//	    citation_debt_test.go:97: unattributed citations remaining in this package: 356, over 36 files
//	        [the 36 rows, every one as a green run prints it]
//	--- PASS: TestCitationDebtDoesNotGrow (0.01s)
//	=== RUN   TestCitationsResolve
//	    citation_test.go:546: mode.go:272 cites R-9.3 — story 046 has no R-9.3, and the
//	        citation is bare, so it names nothing a reader can resolve.
//	        Remedy: prefix the citation with the story that owns the requirement, in the
//	        S0NN- form used throughout this repository. The story files under .epic/ are
//	        not committed, so this comment is the only record of what the code was
//	        answering.
//	    citation_test.go:560: 1 of the 499 citations read across 38 files cannot be resolved
//	--- FAIL: TestCitationsResolve (0.01s)
//
// Read together, the two runs say what neither says alone: an unresolvable
// citation still fails the package and is NOT counted as debt, and an
// unattributed one is counted and does NOT fail the package. Neither mutation
// moved the other verdict. Mutation 2's message is word for word the one this
// file's header recorded for its own first mutation, before the ceiling
// existed — a different line of mode.go, the same sentence — which is the
// evidence that sub-task 11.2 left the hard guard's behaviour where it was.
//
// The one exception: R-9.3 above is written with a hyphen the run did not
// print. TestCitationsResolve sweeps this file too, so quoting that number bare
// would make the transcript violate the rule it is transcribing — the device is
// this file's own, from the header. Mutation 1's transcript needed no device at
// all, and that is a property of the ceiling worth noticing: its failure names
// files and counts, never a requirement number, so it can be quoted anywhere.

// unattributedCitations counts, per file, the citations this package resolves
// on their number alone.
//
// It reads eachCitationInPackage's subject — the guard's own — less exactly one
// file. A narrower subject would let the debt hide in the difference, which is
// why the exclusion is an argument here rather than a second walk, and why
// TestCitationDebtSweepAgreesWithTheGuardsSubject checks this count against an
// independent sweep written in citation_debt_test.go.
//
// The exclusion is the whole file rather than the table alone. Every citation
// in citation_test.go is either the definition of 046's set or a fixture for
// the rule, and neither is a claim this package's code makes about a
// requirement.
func unattributedCitations(t *testing.T) map[string]int {
	t.Helper()

	perFile := map[string]int{}
	filesScanned := eachCitationInPackage(t, isRequirementTable, func(path string, _ int, citation string) {
		if citationIsUnattributed(citation) {
			perFile[path]++
		}
	})

	// A sweep that read nothing reports a debt of zero, and zero is exactly
	// what a finished repayment looks like — so the vacuity guard counts FILES
	// and not hits. Guarding on hits would turn story 047's success into a
	// failure. That the reader itself still works is pinned by
	// TestCitationScannerReadsWholeCitations and by TestCitationsResolve's own
	// "read no citation at all" check, which share citationsIn with this sweep.
	if filesScanned == 0 {
		t.Fatal("scanned no Go file — the debt sweep reported its count without looking at anything")
	}

	return perFile
}
