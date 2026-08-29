package main

// Authored for story 046, sub-task 4.5 — R3.4, R3.5.
//
// Written from the contract: R3.4 — "WHEN --export=<path> is given, THE SYSTEM
// SHALL write the COMPLETE report to that path — every unit, every reason in
// full, nothing shortened — whatever --all and --ui asked of the terminal" —
// and R3.5, which says an unwritable path costs neither the terminal render nor
// the exit status. Unchanged Behavior 4 says both are story 044's behaviour,
// inherited rather than redesigned.
//
// The check path is what these run over, because at task 4 it is the only
// producer there is. That is also why it is the right one: it is the reference
// implementation Unchanged Behavior 1 forbids moving, so a difference found
// here is a difference this story introduced.
//
// Red on arrival: --export is autoupdate's own flag today and the harness does
// not exist.
//
// seedCheckOverlay comes from report_fixture_test.go, materialized with 4.3.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// exportedCheck runs a check that exports to path and returns what the terminal
// got, plus the exit status.
func exportedCheck(t *testing.T, c *testCLI, path string, extra ...string) (stdout string, code int) {
	t.Helper()

	args := append([]string{"overlay", "autoupdate", "--check", "--force", "--ui=plain"}, extra...)
	if path != "" {
		args = append(args, "--export="+path)
	}

	stdout, stderr, code := c.Run(args...)
	if strings.Contains(stderr, "unknown flag") {
		t.Fatalf("a flag was rejected: %s", stderr)
	}
	return stdout, code
}

// TestExportCarriesEveryUnitWhateverTheTerminalWasTold pins R3.4. The screen
// may count the packages it does not list; the file may not, because the file
// is kept precisely for when the terminal is gone.
//
// The positive control is the load-bearing half: if --all changed nothing on
// screen, the two exports would be identical for a reason that has nothing to
// do with the requirement, and the comparison would pass over a broken export
// for the rest of the story.
func TestExportCarriesEveryUnitWhateverTheTerminalWasTold(t *testing.T) {
	pkgs := []string{"app-misc/jq", "dev-lang/go", "app-shells/fish"}

	withoutAll := newTestCLI(t)
	seedCheckOverlay(t, withoutAll, "1.0.0", "2.0.0", pkgs...)
	quietPath := filepath.Join(t.TempDir(), "quiet.md")
	quietTerminal, _ := exportedCheck(t, withoutAll, quietPath)

	withAll := newTestCLI(t)
	seedCheckOverlay(t, withAll, "1.0.0", "2.0.0", pkgs...)
	loudPath := filepath.Join(t.TempDir(), "loud.md")
	loudTerminal, _ := exportedCheck(t, withAll, loudPath, "--all")

	if quietTerminal == loudTerminal {
		t.Fatal("--all changed nothing on the terminal — the comparison below would pass whatever the export did, so the fixture is not exercising the flag (R3.4)")
	}

	quiet, err := os.ReadFile(quietPath)
	if err != nil {
		t.Fatalf("reading %s: %v", quietPath, err)
	}
	loud, err := os.ReadFile(loudPath)
	if err != nil {
		t.Fatalf("reading %s: %v", loudPath, err)
	}

	if string(quiet) != string(loud) {
		t.Errorf("the export differs with and without --all — the file mirrored what the terminal was told, and a record missing exactly what the screen dropped answers no question later (R3.4)\n--- without --all ---\n%s\n--- with --all ---\n%s", quiet, loud)
	}

	for _, pkg := range pkgs {
		if !strings.Contains(string(quiet), pkg) {
			t.Errorf("the export does not name %s — every unit means every unit (R3.4)\n%s", pkg, quiet)
		}
	}
}

// TestExportFormatFollowsTheExtension pins the rule story 044 established: .md
// is Markdown, .json is JSON, anything else is plain text. It is asserted on
// what the file CONTAINS, not on the name it was given.
func TestExportFormatFollowsTheExtension(t *testing.T) {
	cases := map[string]func(t *testing.T, body string){
		"report.json": func(t *testing.T, body string) {
			var doc map[string]any
			if err := json.Unmarshal([]byte(body), &doc); err != nil {
				t.Fatalf("the .json export is not valid JSON: %v\n%s", err, body)
			}
			if _, ok := doc["kind"]; !ok {
				t.Errorf(`the .json export has no "kind" at its root (R4.1):\n%s`, body)
			}
		},
		"report.md": func(t *testing.T, body string) {
			if !strings.Contains(body, "|") || !strings.Contains(body, "#") {
				t.Errorf("the .md export carries no Markdown table or heading:\n%s", body)
			}
		},
		"report.txt": func(t *testing.T, body string) {
			if strings.HasPrefix(strings.TrimSpace(body), "{") {
				t.Errorf("the .txt export is a JSON document:\n%s", body)
			}
			if strings.ContainsRune(body, 0x1b) {
				t.Errorf("the .txt export carries an escape sequence — a file is not a terminal:\n%q", body)
			}
		},
	}

	for name, check := range cases {
		t.Run(name, func(t *testing.T) {
			c := newTestCLI(t)
			seedCheckOverlay(t, c, "1.0.0", "2.0.0", "app-misc/jq")

			path := filepath.Join(t.TempDir(), name)
			if _, code := exportedCheck(t, c, path); code != 0 {
				t.Logf("the run exited %d; the export is still expected to exist", code)
			}

			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading %s: %v", path, err)
			}
			check(t, string(body))
		})
	}
}

// TestExportFailureCostsNeitherTheRenderNorTheStatus pins R3.5. An export is an
// ADDITIONAL copy; losing it must not cost the operator the run they already
// paid for, and must not change what the exit status meant.
//
// The status is compared against the same run without --export rather than
// against a literal, because the check's own status depends on what it found —
// pinning a number here would make the test a claim about the fixture.
func TestExportFailureCostsNeitherTheRenderNorTheStatus(t *testing.T) {
	readOnly := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(readOnly, 0o500); err != nil {
		t.Fatalf("creating the read-only directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(readOnly, 0o700) })

	baselineCLI := newTestCLI(t)
	seedCheckOverlay(t, baselineCLI, "1.0.0", "2.0.0", "app-misc/jq")
	baselineOut, baselineCode := exportedCheck(t, baselineCLI, "")

	failingCLI := newTestCLI(t)
	seedCheckOverlay(t, failingCLI, "1.0.0", "2.0.0", "app-misc/jq")
	unwritable := filepath.Join(readOnly, "report.json")
	failedOut, failedCode := exportedCheck(t, failingCLI, unwritable)

	if failedCode != baselineCode {
		t.Errorf("an unwritable export changed the exit status: %d with --export, %d without (R3.5)", failedCode, baselineCode)
	}
	if strings.TrimSpace(failedOut) == "" {
		t.Error("an unwritable export cost the operator the terminal render as well as the file (R3.5)")
	}
	if strings.TrimSpace(failedOut) != strings.TrimSpace(baselineOut) {
		t.Errorf("the terminal render changed because an export failed (R3.5)\n--- with a failing --export ---\n%s\n--- without --export ---\n%s", failedOut, baselineOut)
	}
	if _, err := os.Stat(unwritable); err == nil {
		t.Errorf("%s was written after all — the fixture did not make the path unwritable, so nothing above was tested", unwritable)
	}
}
