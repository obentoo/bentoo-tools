package render

// Authored for story 046, sub-task 6.4 — R8.1, R8.2.
//
// Written from the contract: the same two requirements 5.5 answers for the
// manifest report, over the payload design.md D6 chose for DISTANCE — a run
// with no package vocabulary at all. If the envelope only fits package-shaped
// runs, this file is where that is discovered.
//
// The fixture names only the two fields the Data Models block declares for
// SnapshotRun: the subvolume and the steps.
//
// Red on arrival: report.SnapshotRun does not exist.
//
// # The fixture's CALL SITE was adjusted when the payload landed, and why
//
// It was written as `report.SnapshotRun{Subvolume: "/home", Steps: 3}` — a
// string and a count, from design.md's ER sketch. Sub-task 6.2 built the field
// as `Subvolumes []string` because a run covers every subvolume
// engine.subvolumes names, so a single string could only be a join or a lie;
// the JSON key stayed `subvolume`, which is the question R1.6 asks. `Steps`
// became the steps themselves, because a report that says "3" and cannot say
// WHICH three does not satisfy "the outcome of each step it ran".
//
// Only the construction moved. No assertion was touched: what this file pins is
// what the three renderers produce, and it pins it over a run that HAS steps —
// the alternative, a fixture with a count and no rows, renders a sentence and no
// table and would pin nothing about the thing an operator reads. Sub-task 5.5
// hit exactly that and is where the lesson comes from.
//
// The deviation is recorded in .draft/deviations.yaml with
// test_surface_adjusted: true.
//
// Generate the goldens once the payload does:
//
//	go test ./internal/common/report/render/ -run TestSnapshotGolden -update

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/obentoo/bentoolkit/internal/common/report"
)

func snapshotRun() report.Run {
	return report.Run{
		Schema:   report.SchemaVersion,
		Kind:     report.KindSnapshotRun,
		Title:    "snapshot run",
		Complete: true,
		Payload: report.SnapshotRun{
			Subvolumes: []string{"/home"},
			Ok:         2,
			Failed:     1,
			Steps: []report.SnapshotStep{
				{Subvolume: "/home", Step: "create", Success: true},
				{Subvolume: "/home", Step: "prune", Success: true},
				{
					Subvolume: "/home",
					Step:      "ship",
					Target:    "offsite",
					Success:   false,
					Error:     "rclone exited 1: directory not found",
				},
			},
		},
	}
}

// TestSnapshotGoldenPlain pins the plain rendering of a snapshot report.
func TestSnapshotGoldenPlain(t *testing.T) {
	var buf bytes.Buffer
	if err := Plain(&buf, snapshotRun().Sections(report.SectionOptions{}), Options{Width: 100}); err != nil {
		t.Fatalf("Plain returned an error: %v", err)
	}
	golden(t, "TestSnapshotGoldenPlain", buf.Bytes())
}

// TestSnapshotGoldenMarkdown pins the export rendering.
func TestSnapshotGoldenMarkdown(t *testing.T) {
	var buf bytes.Buffer
	if err := Markdown(&buf, snapshotRun().Sections(report.SectionOptions{ShowAll: true})); err != nil {
		t.Fatalf("Markdown returned an error: %v", err)
	}
	golden(t, "TestSnapshotGoldenMarkdown", buf.Bytes())
}

// TestSnapshotGoldenThreeModesAgree is R8.2 over the second proving consumer,
// and it is the one that tests the CONTRACT rather than the renderer: a
// subvolume run shares no vocabulary with a package run, so a mode that had
// quietly learned something about packages diverges here.
func TestSnapshotGoldenThreeModesAgree(t *testing.T) {
	blocks := snapshotRun().Sections(report.SectionOptions{})
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
				t.Errorf("%s and plain diverge at line %d of %d/%d over the snapshot report (R8.2)\n  plain: %q\n  %s: %q",
					mode.name, i+1, len(plainLines), len(gotLines), p, mode.name, g)
				break
			}
		}
	}

	if !strings.Contains(plain, "/home") {
		t.Errorf("the snapshot report never names the subvolume it operated on (R1.6)\n--- plain ---\n%s", plain)
	}
}
