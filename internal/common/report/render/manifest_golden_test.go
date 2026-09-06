package render

// Authored for story 046, sub-task 5.5 — R8.1, R8.2.
//
// Written from the contract: story.md R8.1 asks that a renderer's output change
// show up as a reviewable difference against a stored golden, and R8.2 asks
// that the three modes be compared over the SAME report on stripped content, so
// that a difference between modes is a difference in content and not in
// decoration. The shape is the one commit 0d508e2 established for the check.
//
// Red on arrival: report.ManifestRun does not exist, and no renderer takes a
// Run yet.
//
// # The fixture names only the two fields design.md declares
//
// The Data Models block gives ManifestRun exactly Ok and Failed. Anything else
// this payload ends up carrying is the implementer's to choose, and a fixture
// that guessed at it would fail after a CORRECT implementation — so the
// assertions below are about the three modes agreeing and about the goldens,
// never about a field this story has not fixed.
//
// The goldens do not exist yet, by design. Generate them once the payload does:
//
//	go test ./internal/common/report/render/ -run TestManifestGolden -update
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

func manifestRun() report.Run {
	return report.Run{
		Schema:   report.SchemaVersion,
		Kind:     report.KindOverlayManifest,
		Title:    "overlay manifest",
		Complete: true,
		Payload:  report.ManifestRun{Ok: 3, Failed: 1},
	}
}

// TestManifestGoldenPlain pins the plain rendering of a manifest report.
func TestManifestGoldenPlain(t *testing.T) {
	var buf bytes.Buffer
	if err := Plain(&buf, manifestRun().Sections(report.SectionOptions{}), Options{Width: 100}); err != nil {
		t.Fatalf("Plain returned an error: %v", err)
	}
	golden(t, "TestManifestGoldenPlain", buf.Bytes())
}

// TestManifestGoldenMarkdown pins the export rendering, which lists everything
// whatever the terminal was told (R3.4).
func TestManifestGoldenMarkdown(t *testing.T) {
	var buf bytes.Buffer
	if err := Markdown(&buf, manifestRun().Sections(report.SectionOptions{ShowAll: true})); err != nil {
		t.Fatalf("Markdown returned an error: %v", err)
	}
	golden(t, "TestManifestGoldenMarkdown", buf.Bytes())
}

// TestManifestGoldenThreeModesAgree is R8.2 over the first proving consumer.
//
// Three renderers written independently is how story 044 acquired three table
// styles; this is the comparison that would have caught it. It compares
// CONTENT — escapes stripped, the fullscreen frame removed — so the modes stay
// free to look different and are not free to say different things.
func TestManifestGoldenThreeModesAgree(t *testing.T) {
	blocks := manifestRun().Sections(report.SectionOptions{})
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
				t.Errorf("%s and plain diverge at line %d of %d/%d over the manifest report (R8.2)\n  plain: %q\n  %s: %q",
					mode.name, i+1, len(plainLines), len(gotLines), p, mode.name, g)
				break
			}
		}
	}

	// A report that rendered to nothing would satisfy every comparison above.
	if strings.TrimSpace(plain) == "" {
		t.Fatal("the manifest report rendered empty — three modes agreeing on nothing is not agreement")
	}
}
