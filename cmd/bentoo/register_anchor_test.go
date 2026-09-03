package main

// Authored for story 046, sub-task 17.2 — S046-R8.3.
//
// The deviation register is read by a later task, and it points at the code
// with `file.go:NN` anchors. A line number is true for exactly one commit: any
// edit above it moves what the register claims to be describing, and nothing
// tells the register. Task 16 is still open and follows six of these anchors,
// so a stale one does not merely age — it sends the next reader to the wrong
// line and lets them conclude the register was wrong about the code.
//
// The guard therefore asks two things of every anchor in the register:
//
//   - RESOLUTION — the named file is exactly one file in this tree, and the
//     named line exists in it;
//   - SUPPORT — where the register writes a checkable token BESIDE the anchor
//     (an identifier in parentheses, a requirement citation, another .go path
//     it says that line names), that token is at or beside the anchored line.
//
// Support is what makes this more than a bounds check. An anchor may name a
// line that exists and still describe a closing brace.
//
// The register lives under `.epic/`, which Constraint 6 keeps out of the
// repository, so on a fresh clone this SKIPS with a named reason rather than
// passing silently — the shape anchor_test.go and width_debt_subject_test.go
// already use.
//
// File resolution reuses goFileIndex/resolveAnchoredFile from anchor_test.go,
// in this package: one walk of the tree, one definition of "names a single
// file", and no second copy to drift.
//
// Every name carries the TestRegisterAnchor prefix.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const deviationRegister = "../../.epic/stories/046-report-across-the-cli/.draft/deviations.yaml"

// supportWindow is how far from the anchored line a supporting token may sit
// and still count as "beside" it. Three lines absorbs a wrapped statement or a
// doc line immediately above the subject; it does not absorb an anchor that has
// drifted by five, nine or twenty-two lines, which is what the register carries
// today.
const supportWindow = 3

var (
	// An anchor: `overlay_validate.go:650`, `cmd/bentoo/root.go:73`. A range
	// (`:141-142`) is checked at its first line.
	registerAnchor = regexp.MustCompile(`([A-Za-z0-9_/.-]+\.go):(\d+)`)

	// A requirement citation the register writes beside an anchor: `R6.3`,
	// `S044-R3.8`, `S046-R3.7`.
	registerCitation = regexp.MustCompile(`\b(S\d{3}-)?R\d+\.\d+\b`)

	// A .go path with NO line number — the register naming a file it says the
	// anchored line mentions ("width_debt_test.go:161 lists render/style.go").
	registerBarePath = regexp.MustCompile(`\b[A-Za-z0-9_/.-]+\.go\b`)
)

// anchorClaim is one `file:line` the register writes, with the tokens written
// beside it and where they were written.
type anchorClaim struct {
	registerLine int    // line in deviations.yaml, so a failure can be found
	named        string // the file as the register spells it
	line         int    // the line the register claims
	text         string // the register line, for the failure message
	support      []supportToken
}

type supportToken struct {
	token string
	kind  string // "identifier", "citation", "path"
}

// readRegister returns the register's lines, or skips when it is not present.
func readRegister(t *testing.T) []string {
	t.Helper()

	raw, err := os.ReadFile(deviationRegister)
	if err != nil {
		t.Skipf("register anchor guard skipped: %s could not be read (%v). The register lives under "+
			".epic/, which Constraint 6 keeps out of the repository, so a clone without it declares a "+
			"no-op rather than passing silently.", deviationRegister, err)
	}
	return strings.Split(string(raw), "\n")
}

// namingVerb is the register's own vocabulary for "the line NAMES this": it is
// the only construction under which a second .go path on the line is read as a
// claim about the anchored line. `says` is deliberately absent — "inline.go:141
// says X; mode.go: ..." starts a second subject rather than describing the
// first — and a clause break or a negation between the two cancels the reading
// ("ManifestUpdate lives in rename.go:104, NOT in manifest.go").
var (
	namingVerb   = regexp.MustCompile(`\b(lists|names|carries|mentions|quotes|declares|cites|contains)\b`)
	clauseBreak  = regexp.MustCompile(`;|--|\bnot\b`)
	bracesOnly   = regexp.MustCompile(`^[\s})\]]*$`)
	identifierRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// supportFor collects the checkable tokens the register writes beside one
// anchor on one line.
//
// It is deliberately narrow. Every token it returns is one the register BINDS
// to the anchor by its own syntax — a name in the parentheses that hold the
// anchor and nothing else, a requirement number in the same sentence, a file
// path the sentence says that line names. Prose that merely shares a line is
// not a claim about a line number and is not read as one.
func supportFor(text, named string, at []int) []supportToken {
	var tokens []supportToken
	seen := map[string]bool{}
	add := func(token, kind string) {
		if token == "" || seen[token] {
			return
		}
		seen[token] = true
		tokens = append(tokens, supportToken{token: token, kind: kind})
	}

	before, after := text[:at[0]], text[at[1]:]

	// `presentCheckReport (overlay_autoupdate_check.go:736)` — the register's
	// appositive form, used throughout to mean "the thing declared there". The
	// parentheses must hold the anchor ALONE: a list of four call sites
	// (`Fullscreen (text.go:93, text.go:130, …)`) is a different claim, and a
	// trailing note (`root.go:73 (PersistentPreRunE, Flag only)`) names an
	// enclosing hook rather than a declaration.
	if strings.HasSuffix(strings.TrimRight(before, " "), "(") && strings.HasPrefix(strings.TrimLeft(after, " "), ")") {
		head := strings.TrimSpace(strings.TrimSuffix(strings.TrimRight(before, " "), "("))
		head = strings.TrimRight(head, ",")
		if ident := lastIdentifier(head); ident != "" {
			add(ident, "declaration")
		}
	}
	// `fullscreen.go:191 (newInterruptibleModel)` — the same claim, reversed,
	// and held to the same rule: the parentheses hold the name alone.
	if rest := strings.TrimLeft(after, " "); strings.HasPrefix(rest, "(") {
		if closeAt := strings.Index(rest, ")"); closeAt > 1 {
			if inner := strings.TrimSpace(rest[1:closeAt]); identifierRe.MatchString(inner) {
				add(inner, "declaration")
			}
		}
	}

	for _, hit := range registerCitation.FindAllString(text, -1) {
		add(hit, "citation")
	}

	// Another .go file the sentence says the anchored line names. Anchored
	// paths are blanked first so an anchor is never read as support for itself
	// or for its neighbour.
	stripped := registerAnchor.ReplaceAllString(after, " ")
	if verb := namingVerb.FindStringIndex(stripped); verb != nil {
		for _, hit := range registerBarePath.FindAllStringIndex(stripped, -1) {
			if hit[0] < verb[1] {
				continue // the verb has to sit BETWEEN the anchor and the path
			}
			if clauseBreak.FindStringIndex(stripped[:hit[0]]) != nil {
				continue // a second clause is a second subject
			}
			path := stripped[hit[0]:hit[1]]
			if path == named || strings.HasSuffix(named, path) || strings.HasSuffix(path, named) {
				continue
			}
			add(path, "path")
		}
	}

	return tokens
}

var identifierTail = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*$`)

func lastIdentifier(text string) string { return identifierTail.FindString(text) }

// declarationLines returns the line each top-level name is declared on.
func declarationLines(t *testing.T, cache map[string]map[string]int, rel string) map[string]int {
	t.Helper()

	if lines, seen := cache[rel]; seen {
		return lines
	}
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, filepath.Join(repoRoot, rel), nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing %s, which the register anchors: %v", rel, err)
	}
	lines := map[string]int{}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			lines[d.Name.Name] = fileSet.Position(d.Pos()).Line
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					lines[s.Name.Name] = fileSet.Position(s.Pos()).Line
				case *ast.ValueSpec:
					for _, ident := range s.Names {
						lines[ident.Name] = fileSet.Position(ident.Pos()).Line
					}
				}
			}
		}
	}
	cache[rel] = lines
	return lines
}

// registerClaims reads every anchor in the register.
func registerClaims(t *testing.T) []anchorClaim {
	t.Helper()

	var claims []anchorClaim
	for index, text := range readRegister(t) {
		for _, at := range registerAnchor.FindAllStringSubmatchIndex(text, -1) {
			named := text[at[2]:at[3]]
			line, err := strconv.Atoi(text[at[4]:at[5]])
			if err != nil {
				continue
			}
			claims = append(claims, anchorClaim{
				registerLine: index + 1,
				named:        named,
				line:         line,
				text:         strings.TrimSpace(text),
				support:      supportFor(text, named, at),
			})
		}
	}
	return claims
}

// nearAnchor reports whether token appears within supportWindow lines of the
// anchored line, and returns what is actually AT that line — the half of the
// message that sends a reader somewhere.
func nearAnchor(lines []string, at int, token string) (bool, string) {
	atText := ""
	if at >= 1 && at <= len(lines) {
		atText = strings.TrimSpace(lines[at-1])
	}
	for probe := at - supportWindow; probe <= at+supportWindow; probe++ {
		if probe < 1 || probe > len(lines) {
			continue
		}
		if strings.Contains(lines[probe-1], token) {
			return true, atText
		}
	}
	return false, atText
}

// fileLines reads a tree file, cached across claims.
func fileLines(t *testing.T, cache map[string][]string, rel string) []string {
	t.Helper()

	if lines, seen := cache[rel]; seen {
		return lines
	}
	raw, err := os.ReadFile(repoRoot + "/" + rel)
	if err != nil {
		t.Fatalf("reading %s, which the register anchors: %v", rel, err)
	}
	lines := strings.Split(string(raw), "\n")
	cache[rel] = lines
	return lines
}

// TestRegisterAnchorsResolve is the guard: every anchor in the deviation
// register names a line that exists, carries something, and supports the claim
// written beside it.
//
// A failure names the anchor, the deviations.yaml line that wrote it, the token
// that is missing, and WHAT IS AT THAT LINE INSTEAD — with the real line where
// the register's own subject can be found. A guard that reports "7 failures"
// sends nobody anywhere, and being sent to the wrong line is the entire defect
// this closes.
func TestRegisterAnchorsResolve(t *testing.T) {
	claims := registerClaims(t)
	index := goFileIndex(t)
	lineCache := map[string][]string{}
	declCache := map[string]map[string]int{}

	swept, offTree, missingLine, empty, unsupported := 0, 0, 0, 0, 0
	for _, claim := range claims {
		swept++
		target := resolveAnchoredFile(index, claim.named)
		if target == "" {
			// Names no single file in this tree — a basename several packages
			// share, or a file outside it. Outside what this guard can decide,
			// and counted so the sweep cannot shrink unnoticed.
			offTree++
			continue
		}

		lines := fileLines(t, lineCache, target)
		if claim.line < 1 || claim.line > len(lines) {
			missingLine++
			t.Errorf("deviations.yaml:%d anchors %s:%d, and %s has %d lines.\n"+
				"    Register text: %s\n"+
				"    Remedy: re-resolve the anchor against HEAD, or drop the line number and name the "+
				"identifier — a name survives every edit above it.",
				claim.registerLine, claim.named, claim.line, target, len(lines), claim.text)
			continue
		}

		at := lines[claim.line-1]
		if bracesOnly.MatchString(at) {
			empty++
			t.Errorf("deviations.yaml:%d anchors %s:%d, and that line is %s — it carries no claim at all.\n"+
				"    Register text: %s\n"+
				"    Remedy: re-resolve the anchor to the line that does what the register says happens here.",
				claim.registerLine, claim.named, claim.line, quoteOrEmpty(strings.TrimSpace(at)), claim.text)
			continue
		}

		for _, support := range claim.support {
			if support.kind == "declaration" {
				declared := declarationLines(t, declCache, target)
				line, isDeclared := declared[support.token]
				if !isDeclared {
					// The register is not naming a declaration of this file —
					// an enclosing hook, a mode, an English word. Outside what
					// this rule can decide.
					continue
				}
				if abs(line-claim.line) <= supportWindow {
					continue
				}
				unsupported++
				t.Errorf("deviations.yaml:%d anchors %s:%d for %s, and %s declares %s at :%d.\n"+
					"    At %s:%d instead: %s\n"+
					"    Register text: %s\n"+
					"    Remedy: re-resolve the anchor to :%d, or drop the line number — %s alone survives "+
					"every edit above it.",
					claim.registerLine, claim.named, claim.line, support.token, target, support.token, line,
					target, claim.line, quoteOrEmpty(strings.TrimSpace(at)), claim.text, line, support.token)
				continue
			}

			present, atText := nearAnchor(lines, claim.line, support.token)
			if present {
				continue
			}
			unsupported++
			where := "nowhere in the file"
			if found := firstLineWith(lines, support.token); found > 0 {
				where = fmt.Sprintf("at :%d", found)
			}
			t.Errorf("deviations.yaml:%d anchors %s:%d for the %s %q, which is not at that line nor "+
				"within %d of it (%s).\n"+
				"    At %s:%d instead: %s\n"+
				"    Register text: %s\n"+
				"    Remedy: re-resolve the anchor to the line that carries %q, or correct the claim.",
				claim.registerLine, claim.named, claim.line, support.kind, support.token, supportWindow,
				where, target, claim.line, quoteOrEmpty(atText), claim.text, support.token)
		}
	}

	t.Logf("swept %d anchors in the deviation register (%d name no single file in this tree); "+
		"%d name a line that does not exist, %d name a line carrying nothing, %d name a line that does "+
		"not support the claim written beside it",
		swept, offTree, missingLine, empty, unsupported)

	if swept == 0 {
		t.Fatal("0 anchors swept: a guard that reads nothing cannot fail, so a zero count is a broken " +
			"guard rather than a clean register")
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// firstLineWith is the "so where IS it?" half of a failure message.
func firstLineWith(lines []string, token string) int {
	for index, text := range lines {
		if strings.Contains(text, token) {
			return index + 1
		}
	}
	return 0
}

func quoteOrEmpty(text string) string {
	if text == "" {
		return "a blank line"
	}
	return strconv.Quote(text)
}

// TestRegisterAnchorGuardCanFail is the hostile case for the checker itself.
// Written against a register that is already stale, a test asking only "does
// the sweep go red?" cannot tell a checker that finds real drift from one that
// rejects everything — including the corrected anchors this sub-task writes.
// Both directions are asserted, on the checker's own primitives.
func TestRegisterAnchorGuardCanFail(t *testing.T) {
	lines := []string{
		"package main",           // 1
		"",                       // 2
		"// R6.3: the citation",  // 3
		"",                       // 4
		"",                       // 5
		"",                       // 6
		"",                       // 7
		"func presentReport() {", // 8
		"}",                      // 9
	}

	if present, _ := nearAnchor(lines, 3, "R6.3"); !present {
		t.Error("a citation that IS at the anchored line was reported missing: the checker rejects every " +
			"anchor, so a corrected register could never turn it green")
	}
	if present, _ := nearAnchor(lines, 8, "presentReport"); !present {
		t.Error("a token that IS at the anchored line was reported missing")
	}
	if present, _ := nearAnchor(lines, 1, "presentReport"); present {
		t.Errorf("an anchor %d lines from its subject was accepted: the checker approves everything, and "+
			"drift is exactly what it exists to see", 7)
	}
	if _, atText := nearAnchor(lines, 9, "presentReport"); atText != "}" {
		t.Errorf("the failure message must say what is AT the stale line; it reported %q where the line "+
			"holds a closing brace", atText)
	}
	if !bracesOnly.MatchString("\t}") || bracesOnly.MatchString("\tosExit(2)") {
		t.Error("the carries-nothing rule does not separate a closing brace from a statement")
	}

	// Support extraction is the other way a guard silently approves
	// everything: one that binds no token to an anchor checks nothing.
	for _, probe := range []struct {
		name, text, file string
		want, unwanted   string
	}{
		{
			name: "appositive names a declaration",
			text: "A third producer, presentCheckReport (overlay_autoupdate_check.go:736), carries R3.9",
			file: "overlay_autoupdate_check.go",
			want: "declaration=presentCheckReport citation=R3.9",
		},
		{
			name:     "a list of call sites is not a declaration claim",
			text:     "Fullscreen (text.go:93, text.go:130, inline.go:62).",
			file:     "text.go",
			unwanted: "declaration=Fullscreen",
		},
		{
			name:     "a parenthesised note is not a declaration claim",
			text:     "cmd/bentoo/root.go:73 (PersistentPreRunE, Flag only)",
			file:     "cmd/bentoo/root.go",
			unwanted: "declaration=PersistentPreRunE",
		},
		{
			name: "a naming verb binds a second path",
			text: "N-13 width_debt_test.go:161 lists render/style.go in the subject list;",
			file: "width_debt_test.go",
			want: "path=render/style.go",
		},
		{
			name:     "a negated path is not a claim about the line",
			text:     "ManifestUpdate lives in rename.go:104, not in manifest.go, and",
			file:     "rename.go",
			unwanted: "path=manifest.go",
		},
	} {
		at := registerAnchor.FindStringSubmatchIndex(probe.text)
		if at == nil {
			t.Fatalf("%s: the premise is wrong — the anchor pattern does not match the register's own "+
				"spelling %q", probe.name, probe.text)
		}
		var kinds []string
		for _, support := range supportFor(probe.text, probe.file, at) {
			kinds = append(kinds, fmt.Sprintf("%s=%s", support.kind, support.token))
		}
		sort.Strings(kinds)
		got := strings.Join(kinds, " ")
		for _, want := range strings.Fields(probe.want) {
			if !strings.Contains(got, want) {
				t.Errorf("%s: extracted %q, which does not bind %s — a sweep that binds no token to an "+
					"anchor checks nothing", probe.name, got, want)
			}
		}
		if probe.unwanted != "" && strings.Contains(got, probe.unwanted) {
			t.Errorf("%s: extracted %q, which binds %s — the register did not claim that, and a guard "+
				"that invents claims fails on a correct register", probe.name, got, probe.unwanted)
		}
	}
}
