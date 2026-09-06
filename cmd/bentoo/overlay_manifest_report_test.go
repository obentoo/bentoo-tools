package main

// Authored for story 046, sub-task 5.2 — R1.1, R1.5, R4.4.
//
// Written from the contract: R1.1 — "WHEN `overlay manifest` finishes, THE
// SYSTEM SHALL render a report of that run in the mode the run resolved" —
// R1.5, which asks for the ok and failed counts as values every renderer and
// the export read, and R4.4: "WHEN a command not previously exportable is
// exported, THE SYSTEM SHALL produce a document distinguishable from every
// other command's BY ITS NAMED KIND ALONE."
//
// Red on arrival: the harness does not exist, `overlay manifest` ends in a
// sentence its library formatted, and nothing about it is exportable.
//
// A dry run is what these drive, so the test needs neither pkgdev nor the
// network. If the implementation reports only on a real run, put a pkgdev stub
// on PATH — what the assertions are about is the REPORT, not the regeneration.
//
// seedCheckOverlay comes from report_fixture_test.go (materialized with 4.3);
// writeExitTestEbuild is this package's own.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/obentoo/bentoolkit/internal/common/report"
)

// exportedDocumentAt reads a written export back as a JSON object.
func exportedDocumentAt(t *testing.T, path string) map[string]any {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the run wrote no export at %s: %v", path, err)
	}

	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		t.Fatalf("the export at %s is not valid JSON: %v\n%s", path, err, body)
	}
	return doc
}

// documentKeys is the document's top-level shape, sorted.
func documentKeys(doc map[string]any) []string {
	keys := make([]string, 0, len(doc))
	for key := range doc {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// manifestRunExport runs a manifest over a seeded overlay and returns what the
// terminal saw and where the export landed.
func manifestRunExport(t *testing.T, extra ...string) (stdout, exportPath string, code int) {
	t.Helper()

	c := newTestCLI(t)
	for _, pkg := range []string{"app-misc/jq", "dev-lang/go"} {
		writeExitTestEbuild(t, c.Overlay(), pkg, "1.0.0")
	}

	exportPath = filepath.Join(t.TempDir(), "manifest.json")
	args := append([]string{"overlay", "manifest", "--dry-run", "--ui=plain", "--export=" + exportPath}, extra...)

	stdout, stderr, code := c.Run(args...)
	if strings.Contains(stderr, "unknown flag") {
		t.Fatalf("a flag was rejected: %s", stderr)
	}
	return stdout, exportPath, code
}

// TestManifestReportRendersInTheResolvedMode pins R1.1. The run must END in a
// report — not in a sentence, and not in nothing.
func TestManifestReportRendersInTheResolvedMode(t *testing.T) {
	stdout, _, code := manifestRunExport(t)

	if code != 0 {
		t.Errorf("`overlay manifest --dry-run` exited %d", code)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Fatal("the run printed nothing at all (R1.1)")
	}
	if strings.ContainsRune(stdout, 0x1b) {
		t.Errorf("--ui=plain produced escape sequences (R2.2):\n%q", stdout)
	}
	for _, pkg := range []string{"app-misc/jq", "dev-lang/go"} {
		if !strings.Contains(stdout, pkg) {
			t.Errorf("the report does not name %s — a report of a run that says nothing about its units is not a report of that run\n%s", pkg, stdout)
		}
	}
}

// TestManifestReportStatesItsCounts pins R1.5 at the surface the operator
// reads. The counts are stated on the terminal AND carried in the export,
// because "as values every renderer and the export read from the report" is
// what separates a number from a sentence somebody formatted.
func TestManifestReportStatesItsCounts(t *testing.T) {
	stdout, exportPath, _ := manifestRunExport(t)

	doc := exportedDocumentAt(t, exportPath)
	payload, ok := doc["payload"].(map[string]any)
	if !ok {
		t.Fatalf(`the export has no payload object: %v`, documentKeys(doc))
	}

	for _, key := range []string{"ok", "failed"} {
		if _, present := payload[key]; !present {
			t.Errorf("the exported payload has no %q count (R1.5) — keys: %v", key, documentKeys(payload))
		}
	}
	if strings.TrimSpace(stdout) == "" {
		t.Error("the terminal saw no report to state the counts in")
	}
}

// TestManifestReportEnvelopeNamesTheManifestKind pins R4.1 for the new
// producer: the document says what it is at its root.
func TestManifestReportEnvelopeNamesTheManifestKind(t *testing.T) {
	_, exportPath, _ := manifestRunExport(t)

	doc := exportedDocumentAt(t, exportPath)

	if got, _ := doc["kind"].(string); got != string(report.KindOverlayManifest) {
		t.Errorf(`the export's kind is %q, want %q`, got, report.KindOverlayManifest)
	}
	if got, ok := doc["schema"].(float64); !ok || int(got) != 2 {
		t.Errorf(`the export's schema is %v, want 2`, doc["schema"])
	}
}

// TestManifestExportIsToldApartByItsKindAlone is R4.4, stated as the
// requirement states it — and it is the assertion that needs two documents.
//
// Both halves are here on purpose:
//
//   - The kinds must DIFFER, or a consumer holding one of these documents
//     cannot tell which command wrote it.
//   - Everything else about their shape must MATCH. That is what makes "by its
//     named kind alone" true rather than merely convenient: if the two
//     documents also differed in their top-level keys, a consumer could
//     distinguish them by sniffing structure, and the discriminator would go
//     untested until the day a third kind happened to share a shape.
func TestManifestExportIsToldApartByItsKindAlone(t *testing.T) {
	_, manifestPath, _ := manifestRunExport(t)
	manifestDoc := exportedDocumentAt(t, manifestPath)

	checkCLI := newTestCLI(t)
	seedCheckOverlay(t, checkCLI, "1.0.0", "2.0.0", "app-misc/jq")
	checkPath := filepath.Join(t.TempDir(), "check.json")
	if _, stderr, _ := checkCLI.Run("overlay", "autoupdate", "--check", "--force", "--ui=plain", "--export="+checkPath); strings.Contains(stderr, "unknown flag") {
		t.Fatalf("a flag was rejected on the check path: %s", stderr)
	}
	checkDoc := exportedDocumentAt(t, checkPath)

	manifestKind, _ := manifestDoc["kind"].(string)
	checkKind, _ := checkDoc["kind"].(string)

	if manifestKind == checkKind {
		t.Fatalf("both commands exported kind %q — the one field that tells two documents apart says the same thing about both (R4.4)", manifestKind)
	}
	if manifestKind != string(report.KindOverlayManifest) || checkKind != string(report.KindAutoupdateCheck) {
		t.Errorf("kinds are %q and %q, want %q and %q", manifestKind, checkKind, report.KindOverlayManifest, report.KindAutoupdateCheck)
	}

	if got, want := strings.Join(documentKeys(manifestDoc), ","), strings.Join(documentKeys(checkDoc), ","); got != want {
		t.Errorf("the two documents do not share a shape — a consumer could tell them apart WITHOUT the kind, so the discriminator is not what is carrying the promise (R4.4)\n  manifest: %s\n  check:    %s", got, want)
	}
}
