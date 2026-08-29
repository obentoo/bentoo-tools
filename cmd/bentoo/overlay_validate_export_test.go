package main

// Authored for story 046, sub-task 8.1 — R4.3.
//
// Written from the contract: R4.3 — "WHEN `overlay validate --json` is used,
// THE SYSTEM SHALL produce a document of THE SAME SHAPE --export produces for a
// .json path" — and design.md D8, which makes --json an alias for --export at
// stdout rather than a second format with a schema of its own.
//
// Story.md's Success Metrics state the point in one line: export formats that
// cannot read each other, 2 → 1.
//
// # What this story does and does not move
//
// Out of Scope is explicit: "overlay validate's report content. This story
// moves the meaning of its --json flag and RESERVES ITS KIND; the report itself
// migrates in 047." So nothing here asserts what the payload contains — only
// that the document is the envelope, that it names the validate kind, and that
// a failed write costs no exit status.
//
// The name assumed: report.KindOverlayValidate, the kind being reserved.
//
// Red on arrival: --json writes validate.Report.Normalized() at the document
// root, in the schema no other command shares.
//
// stubValidateRunner comes from overlay_validate_test.go and mixedReport from
// overlay_validate_render_test.go — both this package's own. The runner is a
// package variable rather than a flag, so it survives the harness rebuilding
// the command tree for each run.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/obentoo/bentoolkit/internal/common/report"
)

// validateDocumentKeys is one document's top-level shape, sorted.
func validateDocumentKeys(t *testing.T, source string, body []byte) []string {
	t.Helper()

	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("the %s document is not valid JSON: %v\n%s", source, err, body)
	}

	keys := make([]string, 0, len(doc))
	for key := range doc {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// TestValidateExportJSONFlagProducesTheEnvelope pins the half a consumer meets
// first: the document `--json` writes to stdout is the envelope, root and all.
func TestValidateExportJSONFlagProducesTheEnvelope(t *testing.T) {
	stubValidateRunner(t, mixedReport())
	c := newTestCLI(t)

	stdout, stderr, _ := c.Run("overlay", "validate", "--json")
	if strings.Contains(stderr, "unknown flag") {
		t.Fatalf("--json was rejected: %s", stderr)
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("`overlay validate --json` did not write one JSON document: %v\n%s", err, stdout)
	}

	if got, ok := doc["schema"].(float64); !ok || int(got) != 2 {
		t.Errorf(`root["schema"] = %v, want 2 — --json still writes the schema no other command shares (D8)`, doc["schema"])
	}
	if got, _ := doc["kind"].(string); got != string(report.KindOverlayValidate) {
		t.Errorf(`root["kind"] = %q, want %q`, got, report.KindOverlayValidate)
	}
	if _, ok := doc["payload"]; !ok {
		t.Errorf(`the document has no "payload" — the validate report is still at the root, so a consumer cannot read it the way it reads every other export (R4.3)`)
	}
}

// TestValidateExportMatchesTheJSONFlagShape is R4.3 itself. Two ways of asking
// for the same document must not produce two documents.
//
// The comparison is on SHAPE, not on bytes: the two runs are two runs, and a
// timestamp or an ordering that differed between them would make a byte
// comparison fail for a reason that has nothing to do with the requirement.
func TestValidateExportMatchesTheJSONFlagShape(t *testing.T) {
	stubValidateRunner(t, mixedReport())

	fromFlag := newTestCLI(t)
	stdout, _, flagCode := fromFlag.Run("overlay", "validate", "--json")

	stubValidateRunner(t, mixedReport())
	toFile := newTestCLI(t)
	path := filepath.Join(t.TempDir(), "validate.json")
	_, _, exportCode := toFile.Run("overlay", "validate", "--export="+path)

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("`--export=%s` wrote nothing: %v", path, err)
	}

	flagKeys := validateDocumentKeys(t, "--json", []byte(stdout))
	fileKeys := validateDocumentKeys(t, "--export", body)

	if got, want := strings.Join(flagKeys, ","), strings.Join(fileKeys, ","); got != want {
		t.Errorf("--json and --export produce different documents (R4.3)\n  --json:   %s\n  --export: %s", got, want)
	}
	if flagCode != exportCode {
		t.Errorf("the two ways of exporting the same run exited differently: --json %d, --export %d", flagCode, exportCode)
	}
}

// TestValidateExportWriteFailurePreservesTheExitStatus pins R3.5 on this path
// too. An export is an additional copy; failing to write it must not change
// what the run DECIDED, because the exit status is what a CI job branches on.
func TestValidateExportWriteFailurePreservesTheExitStatus(t *testing.T) {
	readOnly := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(readOnly, 0o500); err != nil {
		t.Fatalf("creating the read-only directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0o700) })

	stubValidateRunner(t, mixedReport())
	plain := newTestCLI(t)
	plainOut, _, plainCode := plain.Run("overlay", "validate")

	stubValidateRunner(t, mixedReport())
	failing := newTestCLI(t)
	unwritable := filepath.Join(readOnly, "validate.json")
	failedOut, failedStderr, failedCode := failing.Run("overlay", "validate", "--export="+unwritable)

	if failedCode != plainCode {
		t.Errorf("an unwritable export changed the exit status: %d with --export, %d without (R3.5)", failedCode, plainCode)
	}
	if strings.TrimSpace(failedOut) == "" && strings.TrimSpace(plainOut) != "" {
		t.Error("an unwritable export cost the operator the terminal render as well as the file (R3.5)")
	}
	if strings.TrimSpace(failedStderr) == "" {
		t.Error("the failed write was not reported at all — a copy that silently did not happen is worse than one that failed loudly (R3.5)")
	}
	if _, err := os.Stat(unwritable); err == nil {
		t.Errorf("%s was written after all — the fixture did not make the path unwritable, so nothing above was tested", unwritable)
	}
}
