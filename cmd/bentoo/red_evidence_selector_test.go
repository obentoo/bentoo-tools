package main

// Authored for story 046, sub-task 16.7 — S046-R6.2, S046-R8.3.
//
// A sub-task's `Tests:` field is read as a statement about where that
// sub-task's Red was taken. Nothing has ever checked it against the Red
// register, so the field can name one file while the failure that justified the
// sub-task was recorded against another, and every command in the repository
// reports a clean tree.
//
// Sub-task 15.3 closed the other axis of the same claim. Its
// TestValidationSelector* cross-checks a `Tests:` field against the TREE — the
// file exists and declares a name the `Validation:` pattern selects — and it
// never opens .draft/red-evidence.yaml. Field-versus-tree and field-versus-
// register are two different statements, and only the first had a guard.
//
// # What the register says that tasks.md does not
//
// Thirteen sub-tasks at the time of writing name a file the register did not
// record their Red against. Nine of them authored their guard BESIDE the file
// the field names rather than over it — the register holds
// internal/common/report/boundary_fields_test.go where 1.2's field says
// boundary_test.go, render/text_sections_test.go where 2.1's says text_test.go,
// and so on — and that decision is defensible: 1.2's shipped guard argues in
// its own header that replacing boundary_test.go "would delete a passing guard
// to add one". The substance is intact. The FIELD is stale, and the argument
// for it lives in a Go comment no command reads.
//
// Three more (3.3, 5.4, 14.2) carry a non-`None` field that names no file at
// all — "Covered by Task 3.1 and 3.2" — and have no register entry. A
// delegation written as prose cannot be told apart from a delegation that was
// never made, which makes the Red count behind those sub-tasks unfalsifiable.
// One more (17.1) names two files and states in prose that no test was authored
// because the sweeps it repairs ARE the test.
//
// # The declaration token, and why a bare sibling is not enough
//
// The rule this guard enforces is: the register must hold an entry whose
// target_path is the file the field names, OR the field must DECLARE where the
// Red actually lives. A sibling accepted silently is barely a rule at all —
// cmd/bentoo alone holds dozens of test files, and "some file in the same
// directory has an entry" would pass almost any field. So a sibling counts only
// when the field says so, in a token this guard can recognise:
//
//	Red: internal/common/report/boundary_fields_test.go   (a path)
//	Red: Task 13.4                                        (another sub-task)
//
// The path form must be a target_path of an entry belonging to THIS sub-task,
// and must sit in the same package as a file the field names — a token that may
// point anywhere reintroduces the silent-sibling hole one level up. The task
// form must name a sub-task the register actually has an entry for, which is
// what turns "Covered by Task 3.1" from a sentence into a check.
//
// The token is written WITHOUT backticks, and this guard fails a backticked
// one. That is not style: readSubTaskValidations in validation_selector_test.go
// stats every backticked span on a `- Tests:` line that ends in _test.go, and
// the Red path frequently never landed in the tree under that name. A
// backticked token would therefore make 15.3's guard count the sub-task
// `unresolved` and drop it — the exact trap that file's own header records,
// where four quoted filenames took its sweep from 44 pairs to 40 in silence.
//
// # WHICH HALF THIS DOES NOT COVER — read this before trusting a green run
//
// The question answered is "does this field name the file the Red was taken
// against, or say where it is instead?". A field naming TWO files of which one
// has an entry passes on that one; the second is not required to have its own.
// Sub-task 16.5's field names citation_test.go and citation_debt_test.go and
// the register holds only the latter, and this guard accepts it. Requiring an
// entry per named file would be a different and stricter rule; it would also
// fail sub-tasks that widen an existing guard, where one Red run covers both
// files. Saying which of the two rules is in force is the point of this
// paragraph — a reader who assumes the strict one will read a green run as
// proof it never gave.
//
// Nor does it judge whether the Red was a GOOD one. `failed: true` and a
// transcript are the register's business; this guard only asks that the entry
// exists and that tasks.md points at it.
//
// A sub-task still marked `[ ]` has not run, so it has no Red yet and is
// counted as pending rather than judged. A sub-task marked `[~]` — superseded —
// HAS run and does have an entry (6.1 is the live example), so it is swept.
// Note that 15.3's own head regexp accepts only `[ x]` and therefore cannot see
// 6.1 at all; that sub-task is invisible to one axis and in violation on the
// other, which is why the state characters are spelled out below.
//
// The guard reads .epic/, which Constraint 6 keeps out of the repository, so on
// a fresh clone it SKIPS with a named reason rather than passing silently — a
// declared no-op in CI and a real gate locally, the shape 11.3, 15.3 and 17.2
// all use.
//
// Every name carries the TestRedEvidenceSelector prefix.

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const (
	redEvidenceTasksPath    = "../../.epic/stories/046-report-across-the-cli/tasks.md"
	redEvidenceRegisterPath = "../../.epic/stories/046-report-across-the-cli/.draft/red-evidence.yaml"
)

var (
	// The sub-task head, with its state character CAPTURED: `x` done, `~`
	// superseded, ` ` not yet run. The three are treated differently below and
	// a parser that collapsed them would drop 6.1 or fail every unrun sub-task.
	redEvidenceHead = regexp.MustCompile(`^\s*- \[([ x~])\] (\d+\.\d+) - `)

	redEvidenceSpan = regexp.MustCompile("`([^`]+)`")

	// The declaration token. The optional backtick is captured rather than
	// ignored so that a backticked token can be REPORTED rather than silently
	// accepted — see the header on 15.3's stat trap.
	redEvidenceToken = regexp.MustCompile("Red: (`?)(Task \\d+\\.\\d+|[A-Za-z0-9_./-]+_test\\.go)")

	// The register's own two lines. Anchored at their exact indentation so that
	// a transcript quoted inside a `reason: |` block — which is indented deeper
	// — cannot be read as a second entry.
	redRegisterTask   = regexp.MustCompile(`^  - task: "(\d+\.\d+)"`)
	redRegisterTarget = regexp.MustCompile(`^    target_path: "([^"]+)"`)
)

// verdicts. A delegated field is ACCEPTED and REPORTED: the log names both the
// file the field carries and the file the Red was taken against, so that a
// reader can judge the delegation rather than discovering it was assumed.
const (
	redVerdictExact     = "exact"
	redVerdictDelegated = "delegated"
	redVerdictMismatch  = "mismatch"
)

// testsClaim is one sub-task's `Tests:` field, already extracted: the files it
// names in backticks, the delegations it declares, and whether it is a `None`.
type testsClaim struct {
	number      string
	state       string // "x", "~" or " "
	body        string
	files       []string
	delegations []redDelegation
	none        bool
}

type redDelegation struct {
	task       string // "13.4", or "" for the path form
	path       string // "internal/common/report/boundary_fields_test.go", or ""
	backticked bool
}

// parseTestsClaims turns tasks.md into one claim per sub-task that has a
// `Tests:` field. Sub-tasks with no such field — a Commit sub-task — are not
// claims and are not returned.
func parseTestsClaims(raw string) []testsClaim {
	var (
		out     []testsClaim
		current *testsClaim
	)
	flush := func() {
		if current != nil && current.body != "" {
			out = append(out, *current)
		}
		current = nil
	}

	for _, line := range strings.Split(raw, "\n") {
		if head := redEvidenceHead.FindStringSubmatch(line); head != nil {
			flush()
			current = &testsClaim{number: head[2], state: head[1]}
			continue
		}
		if current == nil {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- Tests:") || current.body != "" {
			continue
		}

		body := strings.TrimSpace(strings.TrimPrefix(trimmed, "- Tests:"))
		current.body = body
		current.none = strings.HasPrefix(body, "None")

		for _, span := range redEvidenceSpan.FindAllStringSubmatch(body, -1) {
			if strings.HasSuffix(span[1], "_test.go") {
				current.files = append(current.files, span[1])
			}
		}
		for _, hit := range redEvidenceToken.FindAllStringSubmatch(body, -1) {
			delegation := redDelegation{backticked: hit[1] == "`"}
			if strings.HasPrefix(hit[2], "Task ") {
				delegation.task = strings.TrimPrefix(hit[2], "Task ")
			} else {
				delegation.path = hit[2]
			}
			current.delegations = append(current.delegations, delegation)
		}
	}
	flush()

	return out
}

// parseRedRegister indexes .draft/red-evidence.yaml by sub-task number: which
// paths in the real tree that sub-task's Red was recorded against.
func parseRedRegister(raw string) map[string][]string {
	index := make(map[string][]string)

	current := ""
	for _, line := range strings.Split(raw, "\n") {
		if task := redRegisterTask.FindStringSubmatch(line); task != nil {
			current = task[1]
			continue
		}
		if current == "" {
			continue
		}
		if target := redRegisterTarget.FindStringSubmatch(line); target != nil {
			index[current] = append(index[current], target[1])
		}
	}
	return index
}

func containsPath(paths []string, want string) bool {
	for _, got := range paths {
		if got == want {
			return true
		}
	}
	return false
}

// sameGoPackage is the sibling test, and it is a directory comparison on
// STRINGS rather than a stat. It has to be: the file a Red was taken against is
// frequently a file that never landed in the tree under that name, because the
// sub-task shipped its assertions into an existing neighbour instead.
func sameGoPackage(a, b string) bool {
	return path.Dir(a) == path.Dir(b)
}

// classifyRedEvidence answers, for one sub-task, which of the three things is
// true: the field names the file the Red was taken against (exact); the field
// declares where the Red lives instead, and that declaration resolves in the
// register (delegated); or neither (mismatch).
//
// The detail string is the sentence a maintainer meets, and every mismatch
// branch names all three things needed to act: the sub-task, the file the field
// carries, and the file the register holds.
func classifyRedEvidence(claim testsClaim, index map[string][]string) (string, string) {
	own := index[claim.number]

	// A backticked token is checked BEFORE anything else, and the order is the
	// whole of its value. redEvidenceSpan reads a backticked _test.go span as a
	// file the FIELD NAMES, so `Red: `x_test.go`` also satisfies the exact match
	// below and this axis goes green — while readSubTaskValidations in
	// validation_selector_test.go stats that same span, fails to find it (a Red
	// path frequently never landed in the tree under that name), and drops the
	// sub-task as unresolved. Green here, one pair quietly gone there: the two
	// axes would trade rather than agree. Measured while authoring this file —
	// the case was classified "exact" until this branch moved up.
	for _, delegation := range claim.delegations {
		if delegation.backticked {
			return redVerdictMismatch, fmt.Sprintf(
				"sub-task %s declares its Red in backticks (`Red: %s%s`). Write the token WITHOUT them: "+
					"readSubTaskValidations in validation_selector_test.go stats every backticked span on this "+
					"line ending in _test.go, and a Red path that never landed in the tree makes 15.3's guard "+
					"count this sub-task unresolved and drop it — the sweep shrinks and nothing fails.",
				claim.number, delegation.task, delegation.path)
		}
	}

	for _, file := range claim.files {
		if containsPath(own, file) {
			return redVerdictExact, fmt.Sprintf("%s — the register's entry for %s was taken against it", file, claim.number)
		}
	}

	for _, delegation := range claim.delegations {
		if delegation.task != "" {
			if len(index[delegation.task]) == 0 {
				return redVerdictMismatch, fmt.Sprintf(
					"sub-task %s declares `Red: Task %s`, and the register holds no entry for sub-task %s at all. "+
						"A delegation to a Red that does not exist is the prose problem with a token wrapped "+
						"around it: name a sub-task whose Red was recorded, or record one.",
					claim.number, delegation.task, delegation.task)
			}
			return redVerdictDelegated, fmt.Sprintf(
				"field names %s; Red delegated to task %s and recorded against %s",
				redFieldFiles(claim), delegation.task, strings.Join(index[delegation.task], ", "))
		}

		if !containsPath(own, delegation.path) {
			return redVerdictMismatch, fmt.Sprintf(
				"sub-task %s declares `Red: %s`, and the register holds no entry for sub-task %s with that "+
					"target_path (it holds: %s). The path form points at THIS sub-task's own entry — an entry "+
					"belonging to some other sub-task is that other sub-task's evidence, and reading it as this "+
					"one's is how a Red gets counted twice.",
				claim.number, delegation.path, claim.number, redRegisterPaths(own))
		}
		if len(claim.files) > 0 && !redSiblingOfAny(delegation.path, claim.files) {
			return redVerdictMismatch, fmt.Sprintf(
				"sub-task %s declares `Red: %s`, which is not in the same package as the file its field names "+
					"(%s). A declaration that may point anywhere is the silent-sibling hole one level up: the "+
					"Red must have been taken beside the work, or the field names the wrong file.",
				claim.number, delegation.path, redFieldFiles(claim))
		}
		return redVerdictDelegated, fmt.Sprintf(
			"field names %s; Red recorded against the sibling %s, declared",
			redFieldFiles(claim), delegation.path)
	}

	return redVerdictMismatch, redEvidenceFault(claim, own)
}

// redEvidenceFault is the undeclared case, split by what the register does hold,
// because the two have different remedies.
func redEvidenceFault(claim testsClaim, own []string) string {
	if len(claim.files) == 0 {
		return fmt.Sprintf(
			"sub-task %s: `Tests:` is not None, names no file, and the register holds %s. Its coverage is "+
				"claimed in prose (%q), and prose cannot be told apart from a delegation nobody made — which is "+
				"what makes the Red count behind this sub-task unfalsifiable.\n"+
				"    Remedy: add the token, unbackticked — `Red: Task N.M` naming the sub-task whose Red covers "+
				"this one. The register must hold an entry for that sub-task, so the claim becomes checkable "+
				"rather than merely written.",
			claim.number, redRegisterPaths(own), redShortBody(claim.body))
	}

	sibling := ""
	for _, file := range claim.files {
		for _, target := range own {
			if sameGoPackage(file, target) {
				sibling = target
			}
		}
	}
	if sibling != "" {
		return fmt.Sprintf(
			"sub-task %s: the field names %s and the Red was recorded against %s — a SIBLING in the same "+
				"package, not the named file.\n"+
				"    The substance may be entirely sound (a sub-task that added its assertions to an existing "+
				"neighbour rather than replacing a passing guard), but the field states the other thing and no "+
				"command has ever compared the two.\n"+
				"    Remedy: declare it on the `Tests:` line, without backticks — `Red: %s` — or correct the "+
				"field to name the file the Red was taken against. Do NOT move a shipped test to fit a field.",
			claim.number, redFieldFiles(claim), sibling, sibling)
	}

	return fmt.Sprintf(
		"sub-task %s: the field names %s and the register holds %s for it.\n"+
			"    Remedy: record the Red against a file this field names, or — where the sub-task's evidence is "+
			"another sub-task's run — declare it with `Red: Task N.M`, unbackticked. A `Tests:` field naming a "+
			"file with no Red behind it reads as a test that was written first and was not.",
		claim.number, redFieldFiles(claim), redRegisterPaths(own))
}

func redSiblingOfAny(candidate string, files []string) bool {
	for _, file := range files {
		if sameGoPackage(candidate, file) {
			return true
		}
	}
	return false
}

func redFieldFiles(claim testsClaim) string {
	if len(claim.files) == 0 {
		return "no file"
	}
	return strings.Join(claim.files, " and ")
}

func redRegisterPaths(own []string) string {
	if len(own) == 0 {
		return "no entry"
	}
	return strings.Join(own, ", ")
}

func redShortBody(body string) string {
	if len(body) <= 72 {
		return body
	}
	return body[:72] + "..."
}

// readRedEvidenceInputs returns tasks.md and the register, or skips naming
// which of the two was missing.
func readRedEvidenceInputs(t *testing.T) ([]testsClaim, map[string][]string) {
	t.Helper()

	tasks, err := os.ReadFile(redEvidenceTasksPath)
	if err != nil {
		t.Skipf("red-evidence selector guard skipped: %s could not be read (%v). .epic/ is not committed "+
			"(Constraint 6), so on a fresh clone this guard is a declared no-op rather than a silent pass.",
			redEvidenceTasksPath, err)
	}
	register, err := os.ReadFile(redEvidenceRegisterPath)
	if err != nil {
		t.Skipf("red-evidence selector guard skipped: %s could not be read (%v). The Red register lives under "+
			".epic/, which Constraint 6 keeps out of the repository, so a clone without it declares a no-op "+
			"rather than passing silently.", redEvidenceRegisterPath, err)
	}

	index := parseRedRegister(string(register))
	if len(index) == 0 {
		t.Fatalf("%s parsed to 0 entries: the register is present and this guard read nothing out of it, "+
			"which fails every field rather than checking any", redEvidenceRegisterPath)
	}
	return parseTestsClaims(string(tasks)), index
}

// TestRedEvidenceSelectorFieldsResolveToTheRegister is the rule: every sub-task
// that has run and claims a test names the file its Red was taken against, or
// declares where that Red lives.
func TestRedEvidenceSelectorFieldsResolveToTheRegister(t *testing.T) {
	claims, index := readRedEvidenceInputs(t)

	entries := 0
	for _, paths := range index {
		entries += len(paths)
	}

	swept, exact, mismatched, pending, none := 0, 0, 0, 0, 0
	var delegated []string

	for _, claim := range claims {
		if claim.none {
			none++
			continue
		}
		if claim.state == " " {
			// Not run: it has no Red yet and ought not to. Counted, not judged
			// — a guard that failed until the last sub-task landed is a guard
			// nobody keeps.
			pending++
			continue
		}

		swept++
		verdict, detail := classifyRedEvidence(claim, index)
		switch verdict {
		case redVerdictExact:
			exact++
		case redVerdictDelegated:
			delegated = append(delegated, fmt.Sprintf("%s: %s", claim.number, detail))
		default:
			mismatched++
			t.Errorf("%s", detail)
		}
	}

	sort.Strings(delegated)
	t.Logf("swept %d sub-task Tests fields against %d register entries over %d sub-tasks "+
		"(%d exact, %d delegated, %d mismatched; %d not yet run, %d declare no test)",
		swept, entries, len(index), exact, len(delegated), mismatched, pending, none)
	for _, line := range delegated {
		t.Logf("    delegated  %s", line)
	}

	// A sweep that read nothing finds no violation, and that looks exactly like
	// success — the vacuity guard 13.4's measurement asks for and 15.3 carries.
	if swept == 0 {
		t.Fatal("the sweep checked 0 Tests fields: a guard that reads nothing cannot fail, so a zero count " +
			"is a broken guard rather than a clean tasks.md")
	}
}

// TestRedEvidenceSelectorCanFail is the hostile case for the classifier itself,
// and it is the half that decides whether the sweep above means anything. A
// classifier that accepted any same-package entry would report a clean tasks.md
// forever, because cmd/bentoo alone holds dozens of test files and
// internal/common/report holds a dozen more.
//
// Both converses are here. A field and an entry that name DIFFERENT files must
// not be collapsed into one just because they share a directory; a field and an
// entry that name different files but are joined by an explicit declaration
// must not be split apart. And the third-element case: a declaration must not
// be satisfied by a path that belongs to some OTHER sub-task's entry, which is
// how one Red gets counted for two sub-tasks.
func TestRedEvidenceSelectorCanFail(t *testing.T) {
	index := map[string][]string{
		"1.2":  {"internal/common/report/boundary_fields_test.go"},
		"13.4": {"cmd/bentoo/width_debt_subject_test.go"},
		"10.4": {"internal/common/report/citation_test.go"},
	}

	claim := func(number, body string) testsClaim {
		parsed := parseTestsClaims("  - [x] " + number + " - a sub-task\n    - Tests: " + body + "\n")
		if len(parsed) != 1 {
			t.Fatalf("the fixture %q parsed to %d claims, want 1", body, len(parsed))
		}
		return parsed[0]
	}

	cases := []struct {
		name    string
		number  string
		body    string
		want    string
		mustSay []string
	}{{
		name:   "the named file is the file the Red was taken against",
		number: "1.2",
		body:   "Unit · `internal/common/report/boundary_fields_test.go` — the field is true",
		want:   redVerdictExact,
	}, {
		name:   "an UNDECLARED sibling is refused",
		number: "1.2",
		body:   "Unit · `internal/common/report/boundary_test.go` — the shipped guard, beside the recorded one",
		want:   redVerdictMismatch,
		// The message must name all three, or it gets deleted rather than answered.
		mustSay: []string{"1.2", "internal/common/report/boundary_test.go", "internal/common/report/boundary_fields_test.go"},
	}, {
		name:   "the same sibling, DECLARED, is accepted",
		number: "1.2",
		body:   "Unit · `internal/common/report/boundary_test.go` — the shipped guard. Red: internal/common/report/boundary_fields_test.go",
		want:   redVerdictDelegated,
	}, {
		name:   "a declaration pointing outside the named file's package is refused",
		number: "1.2",
		body:   "Unit · `internal/common/report/boundary_test.go` — Red: cmd/bentoo/width_debt_subject_test.go",
		want:   redVerdictMismatch,
	}, {
		name:   "a declaration satisfied only by a THIRD sub-task's entry is refused",
		number: "1.2",
		body:   "Unit · `internal/common/report/boundary_test.go` — Red: internal/common/report/citation_test.go",
		want:   redVerdictMismatch,
		// 10.4 owns that path. Accepting it here would let one Red stand as
		// evidence for two sub-tasks, which is the collision a pair-wise check
		// never sees.
		mustSay: []string{"1.2", "internal/common/report/citation_test.go"},
	}, {
		name:    "prose coverage with no token is refused",
		number:  "5.4",
		body:    "Covered by Task 2.1 — the measuring behaviour is tested there; this sub-task applies it",
		want:    redVerdictMismatch,
		mustSay: []string{"5.4", "no file", "no entry"},
	}, {
		name:   "the same coverage, declared, is accepted",
		number: "5.4",
		body:   "Covered by Task 13.4 — Red: Task 13.4",
		want:   redVerdictDelegated,
	}, {
		name:    "a delegation to a sub-task the register has no entry for is refused",
		number:  "5.4",
		body:    "Covered by Task 9.9 — Red: Task 9.9",
		want:    redVerdictMismatch,
		mustSay: []string{"5.4", "9.9"},
	}, {
		name:    "a backticked token is refused, because 15.3's guard would stat it",
		number:  "1.2",
		body:    "Unit · `internal/common/report/boundary_test.go` — Red: `internal/common/report/boundary_fields_test.go`",
		want:    redVerdictMismatch,
		mustSay: []string{"backticks"},
	}}

	for _, testCase := range cases {
		got, detail := classifyRedEvidence(claim(testCase.number, testCase.body), index)
		if got != testCase.want {
			t.Errorf("%s: classified %q, want %q\n    %s", testCase.name, got, testCase.want, detail)
			continue
		}
		for _, want := range testCase.mustSay {
			if !strings.Contains(detail, want) {
				t.Errorf("%s: the message does not name %q; a message that stops at \"mismatch\" gets deleted "+
					"rather than answered.\n    %s", testCase.name, want, detail)
			}
		}
	}
}

// TestRedEvidenceSelectorReadsEverySubTaskState is the parser's own hostile
// case, and it is not ceremony: a sub-task the parser cannot see is a sub-task
// no rule applies to, and the sweep above reports that as a pass.
//
// 6.1 is the live instance. It is marked `[~]` — superseded — and 15.3's head
// regexp accepts only `[ x]`, so that sub-task is invisible to the entire
// validation-selector axis while carrying a stale field on this one. A
// superseded sub-task HAS run and DOES have a register entry; an unrun `[ ]`
// sub-task has neither and must be reported as pending rather than failed.
func TestRedEvidenceSelectorReadsEverySubTaskState(t *testing.T) {
	fixture := strings.Join([]string{
		"  - [x] 1.1 - done",
		"    - Tests: Unit · `a/one_test.go` — a scenario",
		"  - [~] 6.1 - superseded (superseded-by: 6.2)",
		"    - Tests: Unit · `b/two_test.go` — a scenario",
		"  - [ ] 9.9 - not yet run",
		"    - Tests: Unit · `c/three_test.go` — a scenario",
		"  - [x] 4.9 - a commit sub-task, which claims nothing",
		"    - Commit: \"feat(046): something\"",
		"",
	}, "\n")

	claims := parseTestsClaims(fixture)
	if len(claims) != 3 {
		t.Fatalf("parsed %d claims, want 3 — a Commit sub-task states no test and the other three do", len(claims))
	}

	states := map[string]string{}
	for _, claim := range claims {
		states[claim.number] = claim.state
		if len(claim.files) != 1 {
			t.Errorf("sub-task %s: parsed %d named files, want 1", claim.number, len(claim.files))
		}
	}
	if states["6.1"] != "~" {
		t.Errorf("the superseded sub-task 6.1 parsed with state %q, want \"~\" — a parser that accepts only "+
			"[ x], as 15.3's does, drops it entirely and every rule then passes it vacuously", states["6.1"])
	}
	if states["9.9"] != " " {
		t.Errorf("the unrun sub-task 9.9 parsed with state %q, want a space — it must be distinguishable so "+
			"that it can be counted pending instead of failed for a Red it cannot yet have", states["9.9"])
	}
}
