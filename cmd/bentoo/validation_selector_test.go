package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
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

// readValidationSelectorSource returns tasks.md, or skips naming why. Every
// axis in this file reads the same bytes through this one function: two copies
// of the skip would let one axis run on a tree where the other declared a
// no-op, and the counts they publish would then be about different files.
func readValidationSelectorSource(t *testing.T) string {
	t.Helper()

	raw, err := os.ReadFile(validationSelectorTasks)
	if err != nil {
		t.Skipf("validation-selector guard skipped: %s could not be read (%v). "+
			".epic/ is not committed (Constraint 6), so on a fresh clone this guard is a declared "+
			"no-op rather than a silent pass.", validationSelectorTasks, err)
	}
	return string(raw)
}

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

	raw := readValidationSelectorSource(t)

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

	for _, line := range strings.Split(raw, "\n") {
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

// ---------------------------------------------------------------------------
// Sub-task 16.8 — the PATH axis.
//
// Everything above reads a `-run` pattern. A `-run` pattern is not the only
// thing a Validation command carries: it also carries PATHS, and a path is a
// claim about WHERE the command is run from. Nothing above opens one.
//
// The class, counted honestly, because the honest count is the finding. The
// seventh validation reported SIX `Validation:` fields naming a story-relative
// `.draft/` path. The true number of EXECUTABLE ones is ONE — sub-task 12.2's
// `grep -c 'sub-task 11.3' .draft/deviations.yaml`, which errors "no such file"
// when run verbatim from the project root, where every other command in that
// file is run. The remaining occurrences sat in `Files:` and `ToDo:` fields,
// which are references rather than commands: they mislead a reader and fail no
// runner. One command and a class of references is a different thing from six
// broken commands, and saying so is the point rather than a footnote.
//
// The reference half turned out to be WIDER than the plan's own census, which
// named five sub-tasks (10.2, 11.2, 11.4, 15.5, 15.6). Swept rather than read,
// it was SEVENTEEN references over twelve sub-tasks (10.2, 11.2, 11.4, 12.2,
// 15.3, 15.4, 15.5, 15.6, 16.7, 16.8, 17.2, 17.3) plus the file's preamble —
// measured by TestValidationSelectorDraftReferencesAreRootRelative below, which
// is why that axis exists at all instead of a promise that the sites were fixed
// by eye. Three of the seventeen were QUOTATIONS of the defect rather than
// references to the register, and they were rewritten as prose rather than
// exempted: an exception for "the guard's own subject" is how a guard rots.
//
// # WHAT COUNTS AS A PATH HERE — the limit, stated before the green run is read
//
// A path claim is an OPERAND OF A COMMAND: a whitespace-delimited word inside a
// backticked span on a `- Validation:` line whose first word (after any leading
// VAR=value) is a shell verb this guard knows. Four kinds of operand are
// excluded, each for a reason that survives being written down:
//
//   - QUOTED (`'sub-task 11.3'`, `'TestWidthDebt|TestSubject'`) — a pattern
//     operand. A quoted path is therefore NOT checked; none exists in this file.
//   - FLAG-SHAPED (`--export=/tmp/v.md`) — a value the command WRITES, not one
//     it reads. A path inside a flag is not checked.
//   - ABSOLUTE (`/tmp`) — a system or output location, not this repository's.
//   - WITHOUT A `/` (`version`, `./...`'s siblings, `CHANGELOG.md`,
//     `validate-story.sh`) — telling a subcommand from a bare filename needs
//     each command's grammar, which this guard does not model. The defect class
//     is a DIRECTORY-relative path, which always carries a separator, so the
//     exclusion costs nothing it was written to catch.
//
// And a Validation field may name a file WITHOUT running it — "`render/text.go`
// still declares one width". That is a name, not a command, and this axis does
// not read it. It is the same reference-versus-command line the count above
// draws, held consistently: `render/` is clear prose about a package, and a
// guard that demanded it be spelled internal/common/report/render/ would be
// rewriting English rather than fixing a command. TestValidationSelectorPathCanFail
// pins that exclusion, so the limit is a test rather than a paragraph.
//
// Two operands need more than a stat, and both are handled rather than skipped:
// a Go package wildcard (`./...`, `./internal/common/report/...`) is checked at
// the directory it roots, and a revision-qualified operand
// (`git show c8e347e:internal/common/report/model.go`) is resolved by GIT,
// because it is correct even for a file HEAD no longer has. Where git cannot
// answer — no repository, a shallow clone without that commit — the operand is
// counted `unverified` and the count is printed, rather than failing a reader's
// tree for a tool's absence or passing it in silence.
//
// This axis reads sub-tasks marked `[~]` (superseded) as well as `[ x]`. The
// two axes above accept only `[ x]` and are therefore blind to 6.1, which
// red_evidence_selector_test.go records; a path in a superseded sub-task's
// command is still a path a reader follows.
//
// # R8.3 — THE RED, WHICH THIS GUARD WAS GREEN FOR ON THE DAY IT WAS WRITTEN
//
// The v9 delta corrected 12.2 before this file was extended, so the path axis
// had nothing to fail on at authoring time. The one-token revert was therefore
// applied to tasks.md, the guard run, and the file restored from a copy taken
// first — md5 42f6a4015fabcbfeae1fd59f407264da before and after, verified by
// md5sum rather than by eye:
//
//	validation_selector_test.go:616: sub-task 12.2: `grep -c 'sub-task 11.3' .draft/deviations.yaml`
//	    names .draft/deviations.yaml, which does not resolve from the repository root (looked at
//	    ../../.draft/deviations.yaml).
//	validation_selector_test.go:624: swept 112 path operands in Validation commands across 80 sub-tasks;
//	    1 unresolvable, 0 unverified (revision-qualified, git could not answer)
//	--- FAIL: TestValidationSelectorPathsResolveFromTheRepositoryRoot (0.00s)
//	validation_selector_test.go:677: swept 16 .draft/ references in tasks.md; 1 story-relative, 0 absent
//	--- FAIL: TestValidationSelectorDraftReferencesAreRootRelative (0.00s)
//
// Both new axes fired on one edit and the four other tests stayed green, which
// is the point of running them together: the two `-run` axes are unaffected by
// a path, and the two fixture-driven controls are the classifiers' own, so a
// run where THEY failed would mean this file was broken rather than tasks.md.
//
// The reference axis did not need a mutation. It was RED at HEAD over the
// seventeen real references above; that natural transcript is recorded in
// .epic/stories/046-report-across-the-cli/.draft/red-evidence.yaml under 16.8.

const storyDraftPrefix = ".epic/stories/046-report-across-the-cli/.draft/"

var (
	// The sub-task head, with `[~]` accepted. subTaskHead above takes only
	// `[ x]`; a superseded sub-task's command is still run by whoever reads it.
	validationPathHead = regexp.MustCompile(`^\s*- \[[ x~]\] (\d+\.\d+) - `)

	// The first word that makes a backticked span a COMMAND rather than a name.
	// Every verb this story's Validation fields actually use, measured from the
	// file rather than imagined: an unknown verb makes the span a name, which is
	// the safe direction — it under-reads instead of inventing a path.
	validationVerbs = map[string]bool{
		"awk": true, "bentoo": true, "cat": true, "find": true, "git": true,
		"go": true, "gofmt": true, "grep": true, "jq": true, "rg": true, "sed": true,
	}

	// One operand, with a quoted one kept WHOLE so a pattern containing spaces
	// is never split into words that look like paths.
	shellOperand = regexp.MustCompile(`'[^']*'|"[^"]*"|\S+`)

	// A leading VAR=value, which precedes the verb rather than being it:
	// `BENTOO_UI=bogus bentoo version`.
	envAssignment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

	// `c8e347e:internal/common/report/model.go` — a path inside a commit.
	revisionQualified = regexp.MustCompile(`^([0-9a-f]{7,40}|HEAD[~^]*):(.+)$`)

	// A trailing `:12` or `:141-142`, which anchors a line rather than naming a
	// different file. Stripped before the path is resolved.
	operandLineAnchor = regexp.MustCompile(`:\d+(-\d+)?$`)

	// Every `.draft/` path tasks.md writes, WITH whatever prefix it carries, so
	// that the correct and the story-relative forms are both swept and the
	// published count does not fall to zero once the file is clean.
	draftReference = regexp.MustCompile(`[A-Za-z0-9_./-]*\.draft/[A-Za-z0-9_./-]+`)
)

// validationPath is one path operand of one Validation command, with enough
// context that a failure names all three things needed to act.
type validationPath struct {
	number  string // the sub-task
	command string // the span it came from, as written
	operand string // the operand, as written
}

// commandPathOperands returns the path operands of one backticked span, or nil
// when the span is not a command at all. The exclusions are the ones the header
// states, in the same order, so the two can be read against each other.
func commandPathOperands(span string) []string {
	words := shellOperand.FindAllString(span, -1)

	verb := 0
	for verb < len(words) && envAssignment.MatchString(words[verb]) {
		verb++
	}
	if verb >= len(words) || !validationVerbs[words[verb]] {
		return nil
	}

	var out []string
	for _, word := range words[verb+1:] {
		switch {
		case strings.HasPrefix(word, "'"), strings.HasPrefix(word, `"`):
			continue // a pattern operand
		case strings.HasPrefix(word, "-"):
			continue // a flag, and its value is written rather than read
		case strings.HasPrefix(word, "/"):
			continue // absolute: a system or output location
		case !strings.Contains(word, "/"):
			continue // a subcommand, a package name, or a bare filename
		}
		out = append(out, word)
	}
	return out
}

// parseValidationPaths turns tasks.md into one record per path operand. It
// takes the source as a string so that the classifier can be driven from a
// fixture — a guard whose only input is the tree it guards cannot be shown to
// fail.
func parseValidationPaths(raw string) []validationPath {
	var out []validationPath

	current := ""
	for _, line := range strings.Split(raw, "\n") {
		if head := validationPathHead.FindStringSubmatch(line); head != nil {
			current = head[1]
			continue
		}
		if current == "" {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- Validation:") {
			continue
		}
		for _, span := range backtickSpan.FindAllStringSubmatch(trimmed, -1) {
			for _, operand := range commandPathOperands(span[1]) {
				out = append(out, validationPath{number: current, command: span[1], operand: operand})
			}
		}
	}
	return out
}

// The three answers a path operand can have. `unverified` is not a pass: it is
// printed and counted, because a guard that cannot check something must say so
// rather than let a green run imply it did.
const (
	pathResolved   = "resolved"
	pathMissing    = "missing"
	pathUnverified = "unverified"
)

// resolveValidationPath answers one operand, and returns the location it
// actually looked at so a failure can be reproduced by hand.
func resolveValidationPath(operand string) (string, string) {
	token := operandLineAnchor.ReplaceAllString(operand, "")

	if revision := revisionQualified.FindStringSubmatch(token); revision != nil {
		return resolveGitRevisionPath(revision[1], revision[2])
	}

	// A Go package wildcard names a tree, not a file: `./...` is checked at the
	// root it starts from, `./internal/common/report/...` at that directory.
	token = strings.TrimSuffix(token, "...")
	if token == "" {
		token = "."
	}

	where := filepath.Join(repoRoot, token)
	if _, err := os.Stat(where); err != nil {
		return pathMissing, where
	}
	return pathResolved, where
}

// resolveGitRevisionPath asks git, because `git show <rev>:<path>` is correct
// for a file the worktree no longer has and wrong to check with a stat. When
// the COMMIT itself does not resolve — no repository, a shallow clone — the
// operand is unverified rather than missing: the distinction is the difference
// between "this path is wrong" and "this clone cannot tell".
func resolveGitRevisionPath(revision, path string) (string, string) {
	specification := revision + ":" + path

	if err := exec.Command("git", "-C", repoRoot, "cat-file", "-e", revision+"^{commit}").Run(); err != nil {
		return pathUnverified, specification
	}
	if err := exec.Command("git", "-C", repoRoot, "cat-file", "-e", specification).Run(); err != nil {
		return pathMissing, specification
	}
	return pathResolved, specification
}

// validationPathFault is the sentence a maintainer meets. It names the sub-task,
// the command and the path, and it says what to do — including the one case
// where the right answer is to change this guard rather than the path.
func validationPathFault(claim validationPath, where string) string {
	return fmt.Sprintf(
		"sub-task %s: `%s` names %s, which does not resolve from the repository root (looked at %s).\n"+
			"    Every command in tasks.md is run from the project root, so a path written relative to\n"+
			"    anything else — the story directory, the package directory — errors \"no such file\" when\n"+
			"    the command is run verbatim. The command then exits non-zero for a reason that has nothing\n"+
			"    to do with what the sub-task claims to prove, and the reader abandons it.\n"+
			"    Remedy: write the path from the project root. If the file is one this sub-task's own work\n"+
			"    CREATES, the fix is a pending bucket in this guard, not a shorter path — no command in this\n"+
			"    story names such a file today, which is why there is none.",
		claim.number, claim.command, claim.operand, where)
}

// TestValidationSelectorPathsResolveFromTheRepositoryRoot is the path axis:
// every path a Validation command names must be openable from where that
// command is run.
func TestValidationSelectorPathsResolveFromTheRepositoryRoot(t *testing.T) {
	claims := parseValidationPaths(readValidationSelectorSource(t))

	subTasks := map[string]bool{}
	swept, unresolvable, unverified := 0, 0, 0
	for _, claim := range claims {
		swept++
		subTasks[claim.number] = true

		verdict, where := resolveValidationPath(claim.operand)
		switch verdict {
		case pathMissing:
			unresolvable++
			t.Errorf("%s", validationPathFault(claim, where))
		case pathUnverified:
			unverified++
			t.Logf("    unverified  sub-task %s: %s is revision-qualified and git cannot answer in this clone",
				claim.number, claim.operand)
		}
	}

	t.Logf("swept %d path operands in Validation commands across %d sub-tasks; %d unresolvable, "+
		"%d unverified (revision-qualified, git could not answer)", swept, len(subTasks), unresolvable, unverified)

	// A sweep that read nothing finds no violation, and that looks exactly like
	// success — the vacuity guard the two axes above already carry.
	if swept == 0 {
		t.Fatal("the sweep checked 0 path operands: a guard that opens nothing cannot fail, so a zero " +
			"count is a broken guard rather than a tasks.md whose commands name no paths")
	}
}

// TestValidationSelectorDraftReferencesAreRootRelative is the REFERENCE half of
// the same class, and it is a narrower rule on purpose. A `Files:` or `ToDo:`
// field is prose, and demanding that every slash-bearing word in it resolve
// would fail "`render/`" and "the `testdata/` fixtures" — clear English about a
// package. So exactly one construction is checked, the one that actually
// misled: a `.draft/` path, which is the register this story writes to on every
// sub-task, written relative to the STORY directory rather than the root.
//
// The rule is universal within tasks.md — every field and the preamble alike —
// because an exception for "a quotation of the defect" is how a guard rots. A
// bare `.draft/` with nothing after it is the directory's NAME, used as
// shorthand ("the first guard to read .draft/"), not a path anyone opens, and
// is not swept.
func TestValidationSelectorDraftReferencesAreRootRelative(t *testing.T) {
	raw := readValidationSelectorSource(t)

	swept, storyRelative, absent := 0, 0, 0
	subTask := "the preamble"
	for number, line := range strings.Split(raw, "\n") {
		if head := validationPathHead.FindStringSubmatch(line); head != nil {
			subTask = "sub-task " + head[1]
		}
		for _, reference := range draftReference.FindAllString(line, -1) {
			swept++
			if !strings.HasPrefix(reference, storyDraftPrefix) {
				storyRelative++
				t.Errorf("%s (tasks.md:%d) names %q, which resolves from the STORY directory and not from "+
					"the project root, where every command in this file is run.\n"+
					"    Remedy: write it %s%s. A field is a reference rather than a command, so this fails no "+
					"runner — it sends a reader to a path that is not there, which is worse for being silent.",
					subTask, number+1, reference, storyDraftPrefix, strings.TrimPrefix(reference, ".draft/"))
				continue
			}
			if _, err := os.Stat(filepath.Join(repoRoot, reference)); err != nil {
				absent++
				t.Errorf("%s (tasks.md:%d) names %q, which is written from the project root and is not there "+
					"(%v). A correctly-spelled path to a file that does not exist reads exactly like a correct "+
					"one.", subTask, number+1, reference, err)
			}
		}
	}

	t.Logf("swept %d .draft/ references in tasks.md; %d story-relative, %d root-relative but absent",
		swept, storyRelative, absent)
	if swept == 0 {
		t.Fatal("0 .draft/ references read: this story's fields name its register constantly, so a zero " +
			"count is a broken sweep rather than a clean file")
	}
}

// TestValidationSelectorPathCanFail is the hostile case for the path
// classifier, and it carries the exclusions as ASSERTIONS rather than as
// sentences in the header. A classifier that read every word as a path would
// fail this file forever and be deleted; one that read none would report a
// clean tasks.md forever and be worthless. Both directions are here.
func TestValidationSelectorPathCanFail(t *testing.T) {
	fixture := strings.Join([]string{
		"  - [x] 12.2 - the defect, as it stood before the v9 delta",
		"    - Validation: `go build ./...` clean; `grep -c 'sub-task 11.3' .draft/deviations.yaml` is non-zero",
		"  - [~] 9.1 - a superseded sub-task, whose commands a reader still runs",
		"    - Validation: `go test ./cmd/bentoo/ -run TestX -v` passes; " +
			"`bentoo overlay validate --export=/tmp/v.md`; `grep -rn 'x' cmd/bentoo/root.go`",
		"  - [x] 9.2 - a Validation field that NAMES a file without running it",
		"    - Validation: `render/text.go` still declares one width; `internal/common/tui` is untouched",
		"",
	}, "\n")

	claims := parseValidationPaths(fixture)

	var got []string
	for _, claim := range claims {
		got = append(got, claim.number+" "+claim.operand)
	}
	want := []string{
		"12.2 ./...",
		"12.2 .draft/deviations.yaml",
		"9.1 ./cmd/bentoo/",
		"9.1 cmd/bentoo/root.go",
	}
	if strings.Join(got, " | ") != strings.Join(want, " | ") {
		t.Fatalf("parsed operands\n    got  %v\n    want %v\n"+
			"    A quoted pattern, a --flag=/tmp/value, an absolute path, a word with no separator and a\n"+
			"    field that names a file without running it are all excluded by name in this file's header;\n"+
			"    a disagreement here means the header and the code have parted.", got, want)
	}

	// A superseded sub-task must be visible. The two axes above accept only
	// `[ x]` and are blind to 6.1; repeating that blindness here would hide a
	// real path behind a state character.
	if len(claims) == 0 || claims[2].number != "9.1" {
		t.Errorf("the `[~]` sub-task's operands were not read; a superseded sub-task's command is still a " +
			"command someone runs")
	}

	broken := claims[1]
	verdict, where := resolveValidationPath(broken.operand)
	if verdict != pathMissing {
		t.Errorf("the story-relative %q was classified %q, want %q — the classifier accepts the one path "+
			"this axis exists to reject", broken.operand, verdict, pathMissing)
	}
	message := validationPathFault(broken, where)
	for _, want := range []string{"12.2", ".draft/deviations.yaml", "project root"} {
		if !strings.Contains(message, want) {
			t.Errorf("the failure message does not name %q; a message that stops at \"bad path\" gets deleted "+
				"rather than answered.\n%s", want, message)
		}
	}

	for _, resolvable := range []string{"./...", "./cmd/bentoo/", "cmd/bentoo/root.go", "./internal/common/report/..."} {
		if verdict, where := resolveValidationPath(resolvable); verdict != pathResolved {
			t.Errorf("%q was classified %q (%s), want %q — a classifier that refuses a good path is refused "+
				"back by whoever has to run it", resolvable, verdict, where, pathResolved)
		}
	}

	// The reference half's classifier, on the two forms and the shorthand.
	references := draftReference.FindAllString(
		"names `.draft/deviations.yaml` and `"+storyDraftPrefix+"red-evidence.yaml`, and reads `.draft/` itself", -1)
	if len(references) != 2 {
		t.Fatalf("the .draft/ sweep found %v, want exactly the two PATHS — a bare `.draft/` is the "+
			"directory's name used as shorthand, not a path a reader opens", references)
	}
	if strings.HasPrefix(references[0], storyDraftPrefix) {
		t.Errorf("the story-relative %q was read as root-relative; the prefix test is inverted and every "+
			"field in the file would pass", references[0])
	}
	if !strings.HasPrefix(references[1], storyDraftPrefix) {
		t.Errorf("the root-relative %q was read as story-relative; a guard that fails the correct form "+
			"teaches the next reader to write the broken one", references[1])
	}
}
