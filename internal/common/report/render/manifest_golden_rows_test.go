package render

// Story 046, sub-task 5.5 — R8.1, R8.2, and the sub-task's own objective:
// "the manifest report's rendering is a reviewable diff from here on".
//
// # Why this file exists beside manifest_golden_test.go
//
// The pre-authored goldens pin a payload carrying COUNTS AND NO ROWS
// (`report.ManifestRun{Ok: 3, Failed: 1}`), which is the degenerate case: the
// section renders two sentences and no table. That case is worth pinning and it
// is pinned there — a report that is asked for counts it holds and answers with
// prose about having none is a real defect, and it was one, found by generating
// that golden and reading it.
//
// What it cannot pin is the table, and the table is most of what a manifest
// report renders: the column sizing (R6.1), the STATE wording, a failure's
// detail line, and the note that says how many successes were held back and why
// (R2.3, S044-R2.5). The fixture below is a populated run, so the diff a future
// change produces is a diff over the thing an operator actually reads.
//
// The pre-authored file is untouched. This one is ADDED beside it, which is the
// same reason boundary_fields_test.go sits beside boundary_test.go: one rule per
// file, and a passing guard is never replaced to add another.
//
// Regenerate with:
//
//	go test ./internal/common/report/render/ -update
//
// and READ the diff before committing it.

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/obentoo/bentoolkit/internal/common/report"
)

// populatedManifestRun is a finished run over four targets: three regenerated,
// one failed with a diagnostic.
//
// The atoms differ in length on purpose — `dev-lang/go` against
// `app-shells/fish` — so the golden pins the measured column width rather than
// agreeing with a typed one by accident (R6.1). The failure carries both an
// Error and an Output because they render on different lines and a golden that
// held only one of them would not notice the other disappearing.
func populatedManifestRun() report.Run {
	return report.Run{
		Schema:   report.SchemaVersion,
		Kind:     report.KindOverlayManifest,
		Title:    "overlay manifest",
		Complete: true,
		Payload: report.ManifestRun{
			Ok:     3,
			Failed: 1,
			Targets: []report.ManifestTarget{
				{Package: "app-misc/jq", Success: true},
				{Package: "dev-lang/go", Success: true},
				{Package: "app-shells/fish", Success: true},
				{
					Package: "sys-apps/portage",
					Success: false,
					Error:   "pkgdev exited 1",
					Output:  "!!! failed to fetch portage-3.0.66.tar.bz2",
				},
			},
		},
	}
}

// TestManifestRowsGoldenPlain pins the default terminal rendering: the failure
// listed, the three successes counted rather than listed, and the note saying
// so.
func TestManifestRowsGoldenPlain(t *testing.T) {
	var buf bytes.Buffer
	if err := Plain(&buf, populatedManifestRun().Sections(report.SectionOptions{}), Options{Width: 100}); err != nil {
		t.Fatalf("Plain returned an error: %v", err)
	}
	golden(t, "TestManifestRowsGoldenPlain", buf.Bytes())
}

// TestManifestRowsGoldenPlainShowAll pins the same run with --all: every target
// listed, and the counts unchanged.
//
// The counts being identical across the two goldens is the point, not a
// coincidence to be tidied away. R8.3 asks that a number never move because a
// listing did, and two files a reviewer can diff against each other is the
// cheapest way to notice the day it does.
func TestManifestRowsGoldenPlainShowAll(t *testing.T) {
	var buf bytes.Buffer
	if err := Plain(&buf, populatedManifestRun().Sections(report.SectionOptions{ShowAll: true}), Options{Width: 100}); err != nil {
		t.Fatalf("Plain returned an error: %v", err)
	}
	golden(t, "TestManifestRowsGoldenPlainShowAll", buf.Bytes())
}

// TestManifestRowsGoldenMarkdown pins the export rendering of a populated run.
func TestManifestRowsGoldenMarkdown(t *testing.T) {
	var buf bytes.Buffer
	if err := Markdown(&buf, populatedManifestRun().Sections(report.SectionOptions{ShowAll: true})); err != nil {
		t.Fatalf("Markdown returned an error: %v", err)
	}
	golden(t, "TestManifestRowsGoldenMarkdown", buf.Bytes())
}

// TestManifestRowsThreeModesAgree is R8.2 over a run that has a table, which is
// where three independently written renderers actually get the chance to
// disagree — story 044 acquired three table styles, and an empty section would
// not have caught it.
//
// It compares CONTENT: escapes stripped, the fullscreen frame removed. The
// modes stay free to look different and are not free to say different things.
func TestManifestRowsThreeModesAgree(t *testing.T) {
	blocks := populatedManifestRun().Sections(report.SectionOptions{ShowAll: true})
	opts := Options{Width: 100}

	var plainBuf bytes.Buffer
	if err := Plain(&plainBuf, blocks, opts); err != nil {
		t.Fatalf("Plain returned an error: %v", err)
	}
	plain := strings.TrimRight(ansi.Strip(plainBuf.String()), "\n")

	inline := strings.TrimRight(trimTrailing(ansi.Strip(captureStdout(t, func() error {
		return Inline(blocks, opts)
	}))), "\n")

	model := newModel(blocks, opts)
	sized, _ := model.Update(tea.WindowSizeMsg{Width: 104, Height: 200})
	fullscreen := strings.TrimRight(unframe(ansi.Strip(sized.View())), "\n")

	// A positive control. If the fixture rendered nothing, every comparison
	// below would pass over three renderers that agree on emptiness, which is
	// exactly the vacuous green this file was added to replace.
	if !strings.Contains(plain, "sys-apps/portage") {
		t.Fatalf("the plain render does not name the failing target — the fixture is not exercising the table, so the agreement below measures nothing:\n%s", plain)
	}

	for _, mode := range []struct {
		name string
		got  string
	}{
		{"inline", inline},
		{"fullscreen", fullscreen},
	} {
		if mode.got == plain {
			continue
		}
		plainLines, gotLines := strings.Split(plain, "\n"), strings.Split(mode.got, "\n")
		for i := 0; i < len(plainLines) || i < len(gotLines); i++ {
			var p, g string
			if i < len(plainLines) {
				p = plainLines[i]
			}
			if i < len(gotLines) {
				g = gotLines[i]
			}
			if p != g {
				t.Errorf("%s and plain diverge at line %d of %d/%d over a populated manifest report (R8.2)\n  plain: %q\n  %s: %q",
					mode.name, i+1, len(plainLines), len(gotLines), p, mode.name, g)
				break
			}
		}
	}
}
