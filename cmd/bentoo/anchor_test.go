package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Sub-task 15.5 — S046-R6.2, S046-R8.3: a citation this story wrote resolves at
// HEAD, and KEEPS resolving.
//
// A `file.go:NN` anchor is accurate for exactly one commit: any edit above line
// NN moves what it points at, and nothing tells the comment. An anchor naming
// the IDENTIFIER survives every edit above it, so the guard has two rules:
//
//   - a line-number anchor whose file resolves in this repository is a
//     violation of FORM — it cannot keep resolving, whether or not it happens
//     to be accurate today;
//   - a `file.go` + `func X` / `type X` citation must RESOLVE: the file exists
//     and declares X.
//
// The sweep is scoped to the non-test .go files story 046 touched
// (`git diff --name-only c8e347e..HEAD`), because "a citation THIS STORY wrote"
// is the claim being kept. Anchors elsewhere in the tree predate the story and
// are a different piece of work. Where the base commit does not resolve — a
// shallow clone — the guard SKIPS with a named reason, the shape
// width_debt_subject_test.go already uses.
//
// Every name carries the TestAnchor prefix.

var (
	lineAnchor  = regexp.MustCompile(`([A-Za-z0-9_/.-]+\.go):(\d+)`)
	identAnchor = regexp.MustCompile(`([A-Za-z0-9_/.-]+\.go)\b`)

	// The backtick is REQUIRED, and that is what keeps the guard off English
	// prose: "the type of run" is not a citation, and an earlier draft of this
	// rule read it as one. A citation is written `func reportModeOrPlain`, in
	// the code-quoting form the package's comments already use.
	//
	// `var` and `const` are here for the same reason `func` and `type` are, and
	// were added while 15.5 was rewriting the anchors: three of the eleven
	// pointed at a package-level seam variable rather than a function
	// (sweepPlannerFn, snapshotRunner), and with the keyword set narrower than
	// declaredNames — which already collects every ValueSpec — those three would
	// have been rewritten into a form the guard reads and does not check.
	// Widening the keyword set only ever adds citations to the check; the sweep
	// over the story's 63 non-test files found no pre-existing `var`/`const`
	// citation, so it strengthens the rule without reinterpreting anything
	// already written.
	identDecl = regexp.MustCompile("`(?:func|type|var|const)\\s+([A-Za-z_][A-Za-z0-9_]*)")
)

// anchorSubjects lists the non-test .go files story 046 touched.
func anchorSubjects(t *testing.T) []string {
	t.Helper()

	if err := exec.Command("git", "-C", repoRoot, "cat-file", "-e", storyBaseCommit+"^{commit}").Run(); err != nil {
		t.Skipf("anchor guard skipped: the story base commit %s does not resolve here (%v), so the set of "+
			"files this story wrote cannot be derived. A declared no-op rather than a silent pass.",
			storyBaseCommit, err)
	}
	out, err := exec.Command("git", "-C", repoRoot, "diff", "--name-only", storyBaseCommit+"..HEAD").Output()
	if err != nil {
		t.Skipf("anchor guard skipped: `git diff --name-only %s..HEAD` could not be read (%v)", storyBaseCommit, err)
	}

	var subjects []string
	for _, line := range strings.Split(string(out), "\n") {
		rel := strings.TrimSpace(line)
		if rel == "" || !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			continue
		}
		if _, statErr := os.Stat(filepath.Join(repoRoot, rel)); statErr != nil {
			continue // deleted or renamed away; nothing to sweep
		}
		subjects = append(subjects, rel)
	}
	return subjects
}

// goFileIndex maps every .go file in the tree by its path suffix, so an anchor
// may name a bare basename or a fuller path and resolve either way.
func goFileIndex(t *testing.T) map[string][]string {
	t.Helper()

	index := map[string][]string{}
	err := filepath.WalkDir(repoRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if name := entry.Name(); name == ".git" || name == ".epic" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			return relErr
		}
		index[filepath.Base(rel)] = append(index[filepath.Base(rel)], rel)
		return nil
	})
	if err != nil {
		t.Fatalf("indexing the tree: %v", err)
	}
	return index
}

// resolveAnchoredFile returns the single tree file an anchor names, or "" when
// it names none (a stdlib file, say) or more than one.
func resolveAnchoredFile(index map[string][]string, named string) string {
	candidates := index[filepath.Base(named)]
	if len(candidates) == 0 {
		return ""
	}
	if strings.Contains(named, "/") {
		var narrowed []string
		for _, candidate := range candidates {
			if strings.HasSuffix(candidate, named) {
				narrowed = append(narrowed, candidate)
			}
		}
		candidates = narrowed
	}
	if len(candidates) != 1 {
		return ""
	}
	return candidates[0]
}

// declaredNames returns the top-level names a file declares.
func declaredNames(t *testing.T, path string) map[string]bool {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(repoRoot, path), nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}

	names := map[string]bool{}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			names[d.Name.Name] = true
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					names[s.Name.Name] = true
				case *ast.ValueSpec:
					for _, ident := range s.Names {
						names[ident.Name] = true
					}
				}
			}
		}
	}
	return names
}

// eachCommentLine visits every line of every comment in a file.
func eachCommentLine(t *testing.T, rel string, visit func(line int, text string)) {
	t.Helper()

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, filepath.Join(repoRoot, rel), nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing %s: %v", rel, err)
	}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			base := fileSet.Position(comment.Pos()).Line
			for offset, text := range strings.Split(comment.Text, "\n") {
				visit(base+offset, text)
			}
		}
	}
}

// unresolvedIdentifier is the sentence a maintainer meets when an identifier
// citation names something its file does not declare. It names BOTH, because
// which half is wrong is the whole question.
func unresolvedIdentifier(rel string, line int, named, identifier string) string {
	return fmt.Sprintf(
		"%s:%d cites %s for %q, and %s declares no %s.\n"+
			"    Remedy: name the identifier the comment actually means, or the file that declares it.\n"+
			"    An identifier survives every edit above it; a citation that names neither survives nothing.",
		rel, line, named, identifier, named, identifier)
}

// TestAnchorCitationsResolve is the guard: one sweep, two rules, one count.
//
// It reads every comment in the non-test .go files this story touched. A
// `file.go:NN` anchor whose file resolves in this tree is a violation of FORM —
// it is accurate for one commit and cannot keep resolving. A `file.go` cited
// beside a backticked `func X` / `type X` must RESOLVE: the file has to declare
// X, or the fix has traded an anchor that rots for one that was never true.
//
// The two rules share a swept count on purpose. Today the tree carries the
// first form and none of the second, so counting them separately would let the
// half that is not written yet report a clean zero — and a sweep that reads
// nothing looks exactly like a sweep that found nothing wrong.
func TestAnchorCitationsResolve(t *testing.T) {
	subjects := anchorSubjects(t)
	index := goFileIndex(t)
	declared := map[string]map[string]bool{}

	swept, offRepo, stale, unresolved := 0, 0, 0, 0
	for _, rel := range subjects {
		eachCommentLine(t, rel, func(line int, text string) {
			for _, hit := range lineAnchor.FindAllStringSubmatch(text, -1) {
				swept++
				if resolveAnchoredFile(index, hit[1]) == "" {
					// Names no single file in this tree — the stdlib's own
					// flag.go, or a basename several packages share. Outside
					// what this guard can keep true.
					offRepo++
					continue
				}
				stale++
				t.Errorf("%s:%d anchors %s to line %s.\n"+
					"    A line number is accurate for one commit: any edit above it moves what the comment\n"+
					"    points at, and nothing tells the comment. Remedy: name the identifier at that line\n"+
					"    (`%s`, `func …`) — an identifier survives every edit above it. Where a position is\n"+
					"    genuinely needed, quote the line's text so it can be found again.",
					rel, line, hit[1], hit[2], hit[1])
			}

			named := identAnchor.FindStringSubmatch(text)
			identifiers := identDecl.FindAllStringSubmatch(text, -1)
			if named == nil || len(identifiers) == 0 {
				return
			}
			target := resolveAnchoredFile(index, named[1])
			if target == "" {
				return
			}
			if _, seen := declared[target]; !seen {
				declared[target] = declaredNames(t, target)
			}
			for _, identifier := range identifiers {
				swept++
				if !declared[target][identifier[1]] {
					unresolved++
					t.Errorf("%s", unresolvedIdentifier(rel, line, named[1], identifier[1]))
				}
			}
		})
	}

	t.Logf("swept %d anchors across %d non-test files this story touched (%d name no single file in this "+
		"tree); %d line-number anchors, %d unresolved identifiers", swept, len(subjects), offRepo, stale, unresolved)

	if swept == 0 {
		t.Fatal("0 anchors swept: a guard that reads nothing cannot fail, so a zero count is a broken " +
			"guard rather than a clean tree")
	}
}

// TestAnchorCanFail is the hostile case for the classifier itself. Written
// after the tree is clean, a test asking "does a good citation pass?" would
// pass against a classifier that approves everything.
func TestAnchorCanFail(t *testing.T) {
	index := goFileIndex(t)

	target := resolveAnchoredFile(index, "overlay_autoupdate_ui.go")
	if target == "" {
		t.Fatal("the premise is wrong: overlay_autoupdate_ui.go does not resolve, so this test cannot tell " +
			"a broken resolver from a broken citation")
	}
	names := declaredNames(t, target)

	if !names["reportModeOrPlain"] {
		t.Error("an identifier the file does declare was reported missing; the resolver refuses everything")
	}
	if names["thisFunctionDoesNotExist"] {
		t.Error("an identifier the file does not declare was reported present; the resolver approves everything")
	}
	if resolveAnchoredFile(index, "no_such_file_anywhere.go") != "" {
		t.Error("a file that is not in the tree resolved anyway")
	}

	message := unresolvedIdentifier("cmd/bentoo/example.go", 12, "overlay_autoupdate_ui.go", "thisFunctionDoesNotExist")
	for _, want := range []string{"overlay_autoupdate_ui.go", "thisFunctionDoesNotExist"} {
		if !strings.Contains(message, want) {
			t.Errorf("the failure message does not name %q; it must name both halves, since which one is wrong "+
				"is the question.\n%s", want, message)
		}
	}
}
