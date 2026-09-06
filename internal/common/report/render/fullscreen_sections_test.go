package render

// Authored for story 046, sub-task 2.3 — R2.1, R2.4.
//
// Written from the contract: design.md's Components block gives Fullscreen the
// same section input as Plain and Inline, and story 044's D7 fixes what happens
// on the way out — the report is left in the terminal's scrollback.
//
// Red on arrival: Fullscreen and newModel take report.Report today.
//
// # Why this file does not import teatest
//
// It cannot. github.com/charmbracelet/x/exp/teatest imports x/exp/golden, which
// registers a package-level `-update` flag, and text_test.go in THIS package
// already registers one of its own. Linking both into the render test binary
// panics before any test runs:
//
//	render.test flag redefined: update
//	panic: render.test flag redefined: update
//
// (Measured, 2026-08-27, by adding a teatest import to this package.) That is
// why story 044's fullscreen tests drive the model through the runProgram seam
// instead, and why the frame below is compared with this package's own golden
// helper. teatest stays usable in internal/common/tui, which registers no
// update flag of its own.
//
// captureRun, stubRunProgram, golden and unframe come from this package's
// existing tests.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/obentoo/bentoolkit/internal/common/report"
)

// fullscreenSectionFixture is a report with more blocks than a short terminal
// can show, so the omission path (R2.3) has something to omit. Its first cell
// values are distinctive, because the dump assertion looks for content only
// this report has.
func fullscreenSectionFixture() []report.Section {
	rows := make([]report.Row, 0, 12)
	for _, atom := range []string{
		"app-misc/jq", "dev-lang/go", "sys-apps/portage", "app-editors/zed",
		"media-libs/libjxl", "dev-libs/icu", "app-shells/fish", "net-misc/curl",
		"dev-vcs/git", "sys-fs/btrfs-progs", "app-arch/zstd", "dev-util/pkgdev",
	} {
		rows = append(rows, report.Row{Cells: []string{atom, "proved"}})
	}

	return []report.Section{
		{
			Title: "Version Check Results",
			Lead:  []string{"12 packages scanned"},
			Rows:  report.Table{Headers: []string{"package", "outcome"}, Rows: rows},
		},
		{
			Title: "Validation Plan",
			Lead:  []string{"1 pending update"},
			Rows: report.Table{
				Headers: []string{"package", "depth"},
				Rows:    []report.Row{{Cells: []string{"app-misc/jq", "compile"}, Detail: "minor bump earns compile"}},
			},
		},
		{
			Title: "Validation Summary",
			Lead:  []string{"1 package evaluated: 1 proved, 0 errored, 0 inconclusive, 0 skipped"},
		},
	}
}

// TestFullscreenSectionsDumpAfterQuit pins R2.4 for the section-fed renderer.
// The alternate screen takes the report away when it closes; the dump is what
// an operator still has afterwards, and it must be the WHOLE report rather than
// the part that happened to be on screen.
func TestFullscreenSectionsDumpAfterQuit(t *testing.T) {
	blocks := fullscreenSectionFixture()

	out := captureRun(t, func() {
		defer stubRunProgram(func(m tea.Model, _ ...tea.ProgramOption) (tea.Model, error) {
			quit, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
			if cmd == nil {
				t.Error("pressing q produced no command — nothing asked the program to quit")
			}
			return quit, nil
		})()

		if err := Fullscreen(blocks, Options{Width: 100}); err != nil {
			t.Errorf("Fullscreen returned an error on the quit path: %v", err)
		}
	})

	dumped := ansi.Strip(out)
	for _, block := range blocks {
		if !strings.Contains(dumped, block.Title) {
			t.Errorf("the scrollback is missing the %q section after quit (R2.4)\n--- stdout ---\n%s", block.Title, dumped)
		}
	}
	// The LAST row of the longest table: a dump that carried only what fitted
	// on the screen would stop before it.
	if !strings.Contains(dumped, "dev-util/pkgdev") {
		t.Errorf("the scrollback holds only part of the report — the dump is the whole report, not the visible part (R2.4)\n--- stdout ---\n%s", dumped)
	}
}

// TestFullscreenStatesWhatItCouldNotShow pins R2.3 in the one mode that has to
// drop content: a viewport too short for the report omits blocks, and an
// operator reading three sections on a screen that had eight has no way to know
// unless the renderer says so.
func TestFullscreenStatesWhatItCouldNotShow(t *testing.T) {
	model := newModel(fullscreenSectionFixture(), Options{Width: 100})
	// A terminal far too short for twelve rows plus three headings.
	sized, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 10})

	view := ansi.Strip(sized.View())
	if !mentionsOmission(view) {
		t.Errorf("a viewport too short for the report omitted content silently (R2.3)\n--- view ---\n%s", view)
	}
}

// TestFullscreenSectionsGoldenFrame pins the frame itself, so a change to what
// the fullscreen mode draws is a reviewable diff rather than a discovery
// (R8.1). Regenerate with:
//
//	go test ./internal/common/report/render/ -run TestFullscreenSectionsGoldenFrame -update
//
// and READ the diff — a golden accepted without being read freezes whatever the
// renderer happened to produce, defects included.
func TestFullscreenSectionsGoldenFrame(t *testing.T) {
	model := newModel(fullscreenSectionFixture(), Options{Width: 100})
	sized, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})

	golden(t, "TestFullscreenSectionsGoldenFrame", []byte(ansi.Strip(sized.View())))
}

// TestFullscreenAgreesWithPlainOnContent is R8.2 for the third mode: unframed
// and stripped, the fullscreen view says exactly what plain says. The frame and
// the key-binding line are removed because a border is the clearest case of
// presentation there is; what must survive unframing is every character of the
// report.
func TestFullscreenAgreesWithPlainOnContent(t *testing.T) {
	blocks := fullscreenSectionFixture()
	opts := Options{Width: 100}

	var buf strings.Builder
	if err := Plain(&buf, blocks, opts); err != nil {
		t.Fatalf("Plain returned an error: %v", err)
	}
	plain := strings.TrimRight(ansi.Strip(buf.String()), "\n")

	model := newModel(blocks, opts)
	// Tall enough that nothing is paginated away: pagination is a property of
	// the screen, and the test above is the one that covers it.
	sized, _ := model.Update(tea.WindowSizeMsg{Width: 104, Height: 200})
	full := strings.TrimRight(unframe(ansi.Strip(sized.View())), "\n")

	if full == plain {
		return
	}

	plainLines, fullLines := strings.Split(plain, "\n"), strings.Split(full, "\n")
	for i := 0; i < len(plainLines) || i < len(fullLines); i++ {
		var p, f string
		if i < len(plainLines) {
			p = plainLines[i]
		}
		if i < len(fullLines) {
			f = fullLines[i]
		}
		if p != f {
			t.Fatalf("fullscreen and plain diverge at line %d of %d/%d (R2.1, R8.2)\n  plain:      %q\n  fullscreen: %q",
				i+1, len(plainLines), len(fullLines), p, f)
		}
	}
}
