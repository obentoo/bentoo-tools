package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Sub-task 15.3 — S046-R6.2, S046-R8.3: a Validation field executed literally
// proves what its sub-task says it proves.
//
// Go's -run is an UNANCHORED SUBSTRING regexp over the test's name, so a
// pattern that names no test in the file its own Tests: field points at still
// exits 0 — having run a neighbour's tests, or none. That is the defect class
// this guard closes. It has recurred three times because every check for it so
// far has been a person reading.
//
// The guard reads .epic/, which Constraint 6 keeps out of the repository, so on
// a fresh clone it SKIPS with a named reason rather than passing silently — a
// declared no-op in CI and a real gate locally, the same shape 11.3's
// cross-check uses when no base commit resolves.
//
// # WHICH HALF THIS DOES NOT COVER — read this before trusting a green run
//
// The rule is "at least ONE test of the named file is selected", which is what a
// NAME-level check can decide. It cannot decide "the RIGHT ones". Sub-task 11.3
// is the live example and is precisely why this paragraph exists: its
// `-run TestWidthDebt` selected four tests — two in the named
// width_debt_subject_test.go and two in a neighbour — and not one of the three
// 11.3 itself authored. This guard passed it. A human reading fixed it at 15.3.
//
// So a green run here means no sub-task's pattern is EMPTY against its own file.
// It does not mean every sub-task's gate would go red if that sub-task's work
// were reverted. Deciding that needs the tests mapped to the change they cover,
// which nothing in the plan records — and claiming this guard settles it would
// be the same over-claim the whole of Task 15 exists to remove.
//
// # A TRAP FOR WHOEVER EDITS tasks.md NEXT
//
// readSubTaskValidations takes EVERY backticked span on a `- Tests:` line whose
// text ends in _test.go as a file it must stat. A superseded filename quoted as
// code in an explanatory note is therefore read as a second named file, missing
// from the tree — and the sub-task is counted `unresolved` and SKIPPED rather
// than checked. That happened while 15.3 was being written: four corrective
// notes quoting the old filename took the swept count from 44 pairs to 40 and
// silently dropped four sub-tasks. The published counts are what caught it,
// which is the argument for publishing them. Name a superseded file WITHOUT
// backticks, or put the note on the Validation: line, where only spans
// containing "go test" are read.
//
// Every name carries the TestValidationSelector prefix.

const validationSelectorTasks = "../../.epic/stories/046-report-across-the-cli/tasks.md"

// subTaskValidation is one sub-task's two relevant fields, already extracted:
// the test files its Tests: field names, the -run patterns its Validation:
// field carries, and the package directories those commands name.
type subTaskValidation struct {
	number   string
	files    []string
	patterns []string
	dirs     []string
}

var (
	subTaskHead     = regexp.MustCompile(`^\s*- \[[ x]\] (\d+\.\d+) - `)
	backtickSpan    = regexp.MustCompile("`([^`]+)`")
	runPattern      = regexp.MustCompile(`-run\s+('[^']*'|"[^"]*"|[^\s'"]+)`)
	goTestDirectory = regexp.MustCompile(`\./[^\s'"` + "`" + `]+/`)
)

// readSubTaskValidations parses tasks.md into one record per sub-task that has
// BOTH a test file and a -run pattern. Anything else — a Tests: None, a Commit
// sub-task, a Validation with no -run — is not a claim this guard can check.
func readSubTaskValidations(t *testing.T) []subTaskValidation {
	t.Helper()

	raw, err := os.ReadFile(validationSelectorTasks)
	if err != nil {
		t.Skipf("validation-selector guard skipped: %s could not be read (%v). "+
			".epic/ is not committed (Constraint 6), so on a fresh clone this guard is a declared "+
			"no-op rather than a silent pass.", validationSelectorTasks, err)
	}

	var (
		out     []subTaskValidation
		current *subTaskValidation
	)
	flush := func() {
		if current != nil && len(current.files) > 0 && len(current.patterns) > 0 {
			out = append(out, *current)
		}
		current = nil
	}

	for _, line := range strings.Split(string(raw), "\n") {
		if head := subTaskHead.FindStringSubmatch(line); head != nil {
			flush()
			current = &subTaskValidation{number: head[1]}
			continue
		}
		if current == nil {
			continue
		}
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "- Tests:"):
			for _, span := range backtickSpan.FindAllStringSubmatch(trimmed, -1) {
				if strings.HasSuffix(span[1], "_test.go") {
					current.files = append(current.files, span[1])
				}
			}
		case strings.HasPrefix(trimmed, "- Validation:"):
			for _, span := range backtickSpan.FindAllStringSubmatch(trimmed, -1) {
				command := span[1]
				if !strings.Contains(command, "go test") {
					continue
				}
				for _, hit := range runPattern.FindAllStringSubmatch(command, -1) {
					current.patterns = append(current.patterns, strings.Trim(hit[1], `'"`))
				}
				current.dirs = append(current.dirs, goTestDirectory.FindAllString(command, -1)...)
			}
		}
	}
	flush()

	return out
}

// testNamesIn returns the `func TestX` names a file declares, from a go/ast
// parse. Reading the names rather than running the tests is what keeps the
// guard honest: it compares the pattern against the same strings `go test`
// would, without needing the package to build.
func testNamesIn(t *testing.T, path string) []string {
	t.Helper()

	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}

	var names []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		names = append(names, fn.Name.Name)
	}
	return names
}

// filesPresent answers whether every file a sub-task names exists in the tree.
func filesPresent(files []string) bool {
	for _, file := range files {
		if _, err := os.Stat(filepath.Join(repoRoot, file)); err != nil {
			return false
		}
	}
	return true
}

// selectsAny is Go's own -run semantics for a top-level test name: an
// unanchored regexp match, nothing compiled into the package.
func selectsAny(t *testing.T, pattern string, names []string) bool {
	t.Helper()

	for _, name := range names {
		matched, err := regexp.MatchString(pattern, name)
		if err != nil {
			t.Fatalf("pattern %q is not a valid regexp: %v", pattern, err)
		}
		if matched {
			return true
		}
	}
	return false
}

// selectorFault is the sentence a maintainer meets, and it names all three
// things needed to act: which sub-task, which pattern, which file.
func selectorFault(number, pattern, file string, names []string) string {
	return fmt.Sprintf(
		"sub-task %s: `-run %s` selects none of the %d tests in %s (%s).\n"+
			"    Go's -run is an unanchored substring regexp, so this command exits 0 having run a\n"+
			"    neighbour's tests or none at all, and the sub-task's Validation field proves nothing.\n"+
			"    Remedy: rewrite the pattern to select the named file's tests, or correct the Tests:\n"+
			"    field to name the file the tests actually live in — whichever is the true one. Do NOT\n"+
			"    rename a test to fit a pattern.",
		number, pattern, len(names), file, strings.Join(names, ", "))
}

// TestValidationSelectorPatternsSelectTheirNamedFiles is the guard: for every
// sub-task that names both a test file and a -run pattern, at least one of its
// patterns must select at least one test that file declares.
func TestValidationSelectorPatternsSelectTheirNamedFiles(t *testing.T) {
	subTasks := readSubTaskValidations(t)

	swept, unresolved, mismatches := 0, 0, 0
	for _, sub := range subTasks {
		for _, file := range sub.files {
			path := filepath.Join(repoRoot, file)
			if _, err := os.Stat(path); err != nil {
				// The file does not exist yet (a sub-task not yet run) or no
				// longer does. Nothing to compare against; counted, not judged.
				unresolved++
				continue
			}
			swept++
			names := testNamesIn(t, path)
			if len(names) == 0 {
				mismatches++
				t.Errorf("sub-task %s: %s declares no `func Test` at all, so no pattern can select it", sub.number, file)
				continue
			}
			selected := false
			for _, pattern := range sub.patterns {
				if selectsAny(t, pattern, names) {
					selected = true
					break
				}
			}
			if !selected {
				mismatches++
				t.Errorf("%s", selectorFault(sub.number, strings.Join(sub.patterns, " / "), file, names))
			}
		}
	}

	t.Logf("swept %d sub-task/file pairs across %d sub-tasks (%d files not present in the tree); %d mismatches",
		swept, len(subTasks), unresolved, mismatches)

	// A sweep that read nothing finds no violation, and that looks exactly like
	// success. This is the vacuity guard 13.4's measurement asks for.
	if swept == 0 {
		t.Fatal("the sweep checked 0 sub-task/file pairs: a guard that reads nothing cannot fail, " +
			"so a zero count is a broken guard rather than a clean tree")
	}
}

// TestValidationSelectorAlternativesAreLive is the second half of the class. A
// pattern can select its own file and still carry a dead alternative — one that
// names nothing anywhere in the package the command runs — which reads as
// coverage and is not.
func TestValidationSelectorAlternativesAreLive(t *testing.T) {
	subTasks := readSubTaskValidations(t)

	checked, dead, pending := 0, 0, 0
	for _, sub := range subTasks {
		// A sub-task whose test file is not in the tree has not been run yet,
		// so its alternatives name tests that do not exist AND ought not to.
		// Counting those as dead would make the guard red until the last
		// sub-task lands, which is a guard nobody keeps.
		if !filesPresent(sub.files) {
			pending++
			continue
		}

		var names []string
		for _, dir := range sub.dirs {
			entries, err := filepath.Glob(filepath.Join(repoRoot, strings.TrimPrefix(dir, "./"), "*_test.go"))
			if err != nil || len(entries) == 0 {
				continue
			}
			for _, entry := range entries {
				names = append(names, testNamesIn(t, entry)...)
			}
		}
		if len(names) == 0 {
			continue
		}
		for _, pattern := range sub.patterns {
			for _, alternative := range strings.Split(pattern, "|") {
				if alternative == "" {
					continue
				}
				checked++
				if !selectsAny(t, alternative, names) {
					dead++
					t.Errorf("sub-task %s: the alternative %q in `-run %s` selects no test in %s — "+
						"it reads as coverage and adds none",
						sub.number, alternative, pattern, strings.Join(sub.dirs, " "))
				}
			}
		}
	}

	t.Logf("checked %d -run alternatives against the packages their commands name; %d dead "+
		"(%d sub-tasks skipped: their test files are not in the tree yet)", checked, dead, pending)
	if checked == 0 {
		t.Fatal("0 alternatives checked: the sweep read nothing, which cannot fail and is not a pass")
	}
}

// TestValidationSelectorCanFail is the hostile case for the classifier itself,
// written first in intent: a guard whose matcher approved everything would
// report a clean tasks.md forever. A wrong pattern must be rejected and the
// message must name the sub-task, the pattern and the file; a right one must
// be accepted.
func TestValidationSelectorCanFail(t *testing.T) {
	names := []string{"TestManifestSingleVoiceRealRun", "TestManifestSingleVoiceDryRun"}

	if selectsAny(t, "TestSomethingElse", names) {
		t.Error("a pattern naming no test in the file was accepted; the matcher approves everything")
	}
	if !selectsAny(t, "TestManifestSingleVoice", names) {
		t.Error("a pattern that does select the file's tests was rejected; the matcher refuses everything")
	}

	message := selectorFault("9.9", "TestSomethingElse", "cmd/bentoo/example_test.go", names)
	for _, want := range []string{"9.9", "TestSomethingElse", "cmd/bentoo/example_test.go"} {
		if !strings.Contains(message, want) {
			t.Errorf("the failure message does not name %q; a message that stops at \"mismatch\" gets deleted rather than answered.\n%s", want, message)
		}
	}
}
