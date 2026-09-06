package render

// Authored for story 046, sub-task 2.2 — R2.1.
//
// Written from the contract: design.md's Components block gives Inline the same
// section input as Plain — "Plain(w, []Section, Options), Inline, Fullscreen,
// Markdown". R8.2 fixes what the comparison between two modes means: the same
// report, compared on STRIPPED content, so a difference between modes is a
// difference in content and not in decoration.
//
// Red on arrival: Inline takes report.Report today, so it cannot be handed
// sections.
//
// captureStdout and trimTrailing come from inline_test.go in this package —
// Inline writes to stdout by design (it redraws in place inside the normal
// scrollback), so there is no writer to hand it.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/obentoo/bentoolkit/internal/common/report"
)

// inlineSectionFixture is deliberately its own fixture rather than a shared
// one: this file has to be materializable on its own, and a fixture reached
// across files would make the two sub-tasks land together or not at all.
func inlineSectionFixture() []report.Section {
	return []report.Section{
		{
			Title: "Version Check Results",
			Lead:  []string{"6 packages scanned, 4 behind"},
			Rows: report.Table{
				Headers: []string{"package", "current", "candidate"},
				Rows: []report.Row{
					{Cells: []string{"app-misc/jq", "1.7.1", "1.8.0"}},
					{Cells: []string{"sys-apps/portage", "3.0.66", "3.0.67"}, Detail: "depth resolved to none: depth=none in the registry"},
				},
			},
			Notes: []string{"2 packages found up to date are not listed"},
		},
	}
}

// strippedContent is the comparison R8.2 names: escape sequences removed,
// trailing blanks off each line, so what remains is what the report SAYS.
//
// Nothing else is normalized. Leading indentation stays, because indentation is
// how a detail line is told from the row it belongs to — it is content, not
// decoration.
func strippedContent(s string) string {
	lines := strings.Split(ansi.Strip(s), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func inlineOutput(t *testing.T, blocks []report.Section, opts Options) string {
	t.Helper()

	return captureStdout(t, func() error { return Inline(blocks, opts) })
}

// TestInlineAndPlainAgreeOnContent is the wrongly-SPLIT half of R8.2: two modes
// rendering one report must not be found to differ merely because one of them
// paints.
func TestInlineAndPlainAgreeOnContent(t *testing.T) {
	opts := Options{Width: 100}
	blocks := inlineSectionFixture()

	var plainBuf bytes.Buffer
	if err := Plain(&plainBuf, blocks, opts); err != nil {
		t.Fatalf("Plain returned an error: %v", err)
	}

	plain := strippedContent(plainBuf.String())
	inline := strippedContent(trimTrailing(inlineOutput(t, blocks, opts)))

	if plain == inline {
		return
	}

	plainLines, inlineLines := strings.Split(plain, "\n"), strings.Split(inline, "\n")
	for i := 0; i < len(plainLines) || i < len(inlineLines); i++ {
		var p, in string
		if i < len(plainLines) {
			p = plainLines[i]
		}
		if i < len(inlineLines) {
			in = inlineLines[i]
		}
		if p != in {
			t.Fatalf("inline and plain diverge at line %d of %d/%d (R2.1, R8.2)\n  plain:  %q\n  inline: %q",
				i+1, len(plainLines), len(inlineLines), p, in)
		}
	}
}

// TestInlineIsPaintedAndPlainIsNot is the other direction of the same claim.
// Without it, an Inline that had quietly become Plain would satisfy the
// comparison above perfectly — the two modes would agree on content because
// they had become one mode.
func TestInlineIsPaintedAndPlainIsNot(t *testing.T) {
	opts := Options{Width: 100}
	blocks := inlineSectionFixture()

	inline := inlineOutput(t, blocks, opts)
	if !strings.ContainsRune(inline, 0x1b) {
		t.Errorf("inline output carries no escape sequence at all — it is plain wearing inline's name:\n%q", inline)
	}

	var plainBuf bytes.Buffer
	if err := Plain(&plainBuf, blocks, opts); err != nil {
		t.Fatalf("Plain returned an error: %v", err)
	}
	if bytes.IndexByte(plainBuf.Bytes(), 0x1b) >= 0 {
		t.Errorf("plain output carries an escape sequence (R2.2):\n%q", plainBuf.String())
	}
}

// TestStrippedComparisonStillSeesContent is the wrongly-COLLAPSE half, and it
// is the one this comparison is easiest to lose. A normalization aggressive
// enough to erase a differing VALUE would make every mode agree with every
// other for the rest of the story, and each of those agreements would be
// reported as evidence.
//
// So: two reports differing in exactly one cell must NOT compare equal after
// stripping.
func TestStrippedComparisonStillSeesContent(t *testing.T) {
	opts := Options{Width: 100}

	first := inlineSectionFixture()
	second := inlineSectionFixture()
	second[0].Rows.Rows[0].Cells[2] = "1.9.0"

	var a, b bytes.Buffer
	if err := Plain(&a, first, opts); err != nil {
		t.Fatalf("Plain returned an error: %v", err)
	}
	if err := Plain(&b, second, opts); err != nil {
		t.Fatalf("Plain returned an error: %v", err)
	}

	if strippedContent(a.String()) == strippedContent(b.String()) {
		t.Fatal("two reports differing in a candidate version compared EQUAL after stripping — the comparison the three modes are judged by cannot see content (R8.2)")
	}
}
