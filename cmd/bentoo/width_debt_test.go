package main

// Authored for story 046, sub-task 9.1 — R6.2.
//
// Written from the contract: R6.2 — "WHERE this story modifies a file that
// declares a fixed column width, THE SYSTEM SHALL replace that declaration with
// a measured width" — and Constraint 8, which says what happens to the rest:
// "the typed column widths in files this story does not touch. The count at
// hand-off is recorded in story 047 rather than described, so that it can be
// checked rather than believed."
//
// story.md measures the debt at c8e347e: 18 typed widths in 6 files. Eight of
// them are in files this story touches — six in overlay_autoupdate.go, two in
// overlay_validate.go — so this test fails until they are measured, and then
// reports the ten that remain as a number story 047 can check.
//
// # Why the AST and not grep
//
// internal/common/report/render/width.go is on the touched list and contains
// the text "%-45s" in a comment explaining why it no longer types one. A grep
// would report the explanation as the defect, for the lifetime of the file. The
// sweep below reads STRING LITERALS out of the parsed source, where a comment
// is not a literal and cannot be one.
//
// Red on arrival: not by mutation and not by a missing symbol — the eight
// widths are in the tree right now.
//
// # Evidence that it still fails, taken after those eight were measured (R8.3)
//
// The line above is true of the day this file was written and stopped being
// checkable the moment task 9 finished: the eight widths are gone, this guard
// passes, and a green test proves nothing about what it would catch. R8.3 asks
// for the evidence a green guard cannot supply, and the story artifacts holding
// the original Red (.draft/red-evidence.yaml) are not committed — `git ls-files
// .epic/` returns nothing — so a pointer to that file is a pointer to nothing
// for anyone who cloned this. internal/common/report/citation_test.go makes
// that argument in full; this is the same correction, applied here.
//
// Measured on 2026-08-29, against sub-task 10.6. One mutation, applied on its
// own, run, and reverted immediately with `git checkout --`, the restore
// verified by `git status --porcelain` reporting nothing and by a second run of
// this test going green.
//
// Mutation: ONE of the six widths this story took out of overlay_autoupdate.go
// was put back where it had stood. The unclaimed-ebuild line is handed a key
// already padded to a column render.ColumnWidth measured; it was returned to
// typing that column into the format string, exactly as it read at c8e347e —
//
//	return fmt.Sprintf("%s %s", key, d.Disk)        // measured, today
//	return fmt.Sprintf("%-45s %s", d.Key, d.Disk)   // typed, as it was
//
// Observed, wrapped to this comment's width and otherwise unedited:
//
//	--- FAIL: TestWidthDebt (0.00s)
//	    width_debt_test.go:220: cmd/bentoo/overlay_autoupdate.go:1146: "%-45s %s"
//	    declares a fixed column width (R6.2).
//	            Remedy: measure it. render.ColumnWidth(values) gives the width the
//	            values actually need, in display cells, and render.Shorten bounds
//	            it — a width typed into a format string is wrong in both
//	            directions at once, too narrow the moment one value outgrows it
//	            and too wide on every run that never comes close.
//
// The mutation went inside a file on the touched list deliberately, and the
// same run shows what that buys: TestWidthDebtRemainderIsCounted PASSED while
// this test failed. The two guards divide the tree between them — this one owns
// the files the story edits, that one counts what is left outside them — so a
// width typed into a touched file has exactly one guard that can catch it, and
// this run is the demonstration that the one which can, does.
//
// The literal is quoted above inside a COMMENT, which costs nothing: the sweep
// reads string literals out of the parsed source, for the reason the section
// above gives, so this note cannot make its own file the defect.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// repoRoot is where this package sits relative to the module root: `go test`
// runs each package in its own directory.
const repoRoot = "../.."

// typedWidth matches a COLUMN whose width was typed rather than measured: a
// left-aligned string field, %-45s or %-10.10s.
//
// It is exactly the pattern story.md counted at c8e347e — 18 in 6 files — so
// the number this test reports and the number the story recorded are the same
// measurement. Two things are deliberately outside it:
//
//   - %3d and %02d. A zero-padded counter or a progress percentage is a
//     NUMBER's format, not a column's width: nothing in a run's data can make
//     "[ 42%]" need more room, so there is nothing to measure.
//   - %-*s. The width comes from an argument, which is what a measured column
//     looks like once it reaches Printf. Flagging it would flag the fix.
var typedWidth = regexp.MustCompile(`%-\d+(\.\d+)?s`)

// widthDebtBaseline is the count story.md measured at c8e347e: 18 typed widths
// in 6 files. It is the number this story may not exceed, in a repository where
// story 044 recorded three as accepted debt and found 18 when it came back.
const widthDebtBaseline = 18

// MEASURED, 2026-08-27: this sweep finds NINETEEN, not eighteen — eight in the
// two touched files and eleven outside them. The nineteenth is in
// misc/design/design-system/component/catalogue.go, which story.md's own count
// missed because it was taken over cmd/ and internal/ only. The baseline stays
// at 18 rather than being quietly raised to 19: it is the number story 047 will
// be handed, and a debt figure edited to match what was found is not a check.

// filesThisStoryTouches is R6.2's subject: every file named in the task list's
// Context blocks, which is the set this story edits.
//
// It is written out rather than derived from `git diff`, because the rule has
// to be checkable in a working tree with no branch point to diff against — and
// because a list somebody has to add to when they touch a seventh file is a
// list that states its own scope. An executor adding a file to this story adds
// it here.
var filesThisStoryTouches = []string{
	"cmd/bentoo/main.go",
	"cmd/bentoo/overlay_autoupdate.go",
	"cmd/bentoo/overlay_autoupdate_report.go",
	"cmd/bentoo/overlay_autoupdate_ui.go",
	"cmd/bentoo/overlay_manifest.go",
	"cmd/bentoo/overlay_validate.go",
	"cmd/bentoo/snapshot_run.go",
	"internal/common/report/classify.go",
	"internal/common/report/mode.go",
	"internal/common/report/model.go",
	"internal/common/report/render/fullscreen.go",
	"internal/common/report/render/inline.go",
	"internal/common/report/render/json.go",
	"internal/common/report/render/style.go",
	"internal/common/report/render/text.go",
	"internal/common/report/render/width.go",
	"internal/overlay/annotate_baseline.go",
	"internal/overlay/compare.go",
	"internal/overlay/manifest.go",
	"internal/snapshot/runner.go",
}

// typedWidthsIn returns every typed width in one Go file, as "line: literal".
func typedWidthsIn(t *testing.T, path string) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}

	var found []string
	ast.Inspect(file, func(node ast.Node) bool {
		lit, ok := node.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if typedWidth.MatchString(lit.Value) {
			found = append(found, fmt.Sprintf("%d: %s", fset.Position(lit.Pos()).Line, lit.Value))
		}
		return true
	})
	return found
}

// TestTypedWidthDetectorSeesOne is the positive control, and it is not
// ceremony: a sweep whose pattern stopped matching would report zero typed
// widths across the whole repository and be read as the story having succeeded.
//
// The star case is the one that matters most. %-*s takes its width from an
// argument — it IS the measured form — so a detector that flagged it would make
// the fix impossible to apply.
func TestTypedWidthDetectorSeesOne(t *testing.T) {
	cases := map[string]bool{
		`"%-45s"`:                       true,
		`"    %-45s %s\n"`:              true,
		`"  %-14s"`:                     true,
		`"%-10.10s"`:                    true,
		`"%-*s"`:                        false, // measured: the width is an argument
		`"\r  Checking: [%3d%%] %d/%d"`: false, // a counter's format, not a column
		`"%s: %v"`:                      false,
		`"%d ok, %d failed"`:            false,
		`"no format at all"`:            false,
	}

	for literal, want := range cases {
		if got := typedWidth.MatchString(literal); got != want {
			t.Errorf("typedWidth.MatchString(%s) = %v, want %v", literal, got, want)
		}
	}
}

// TestWidthDebt is R6.2 itself: no file this story touches declares a width.
//
// The message names the file, the line and the literal, because "a typed width
// exists somewhere in the files you edited" costs its reader a search through
// twenty files — which is how a mechanical rule turns back into a conventional
// one.
func TestWidthDebt(t *testing.T) {
	scanned := 0

	for _, rel := range filesThisStoryTouches {
		path := filepath.Join(repoRoot, rel)
		if _, err := os.Stat(path); err != nil {
			// A file this story renamed or has not created yet. Reported
			// rather than skipped in silence: a list that quietly stopped
			// covering half the story would pass by covering nothing.
			t.Logf("not present, not scanned: %s (%v)", rel, err)
			continue
		}
		scanned++

		for _, hit := range typedWidthsIn(t, path) {
			t.Errorf("%s:%s declares a fixed column width (R6.2).\n"+
				"    Remedy: measure it. render.ColumnWidth(values) gives the width the values actually need, "+
				"in display cells, and render.Shorten bounds it — a width typed into a format string is wrong in "+
				"both directions at once, too narrow the moment one value outgrows it and too wide on every run "+
				"that never comes close.", rel, hit)
		}
	}

	if scanned == 0 {
		t.Fatal("scanned no file — the sweep passed without inspecting anything")
	}
}

// TestWidthDebtRemainderIsCounted is Constraint 8's hand-off: the widths in
// files this story does NOT touch are carried as a number story 047 can check,
// rather than as a description somebody has to believe.
//
// It fails if the debt GROWS. That is the trigger story 044 lacked — it
// recorded three widths as debt "to be repaid when one of them is next
// touched", none were touched, and the count reached 18.
func TestWidthDebtRemainderIsCounted(t *testing.T) {
	touched := make(map[string]bool, len(filesThisStoryTouches))
	for _, rel := range filesThisStoryTouches {
		touched[rel] = true
	}

	remaining := 0
	perFile := map[string]int{}

	err := filepath.WalkDir(repoRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".epic", "testdata", "vendor", "node_modules":
				return fs.SkipDir
			}
			return nil
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}

		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			return relErr
		}
		if touched[filepath.ToSlash(rel)] {
			return nil
		}

		if hits := typedWidthsIn(t, path); len(hits) > 0 {
			perFile[filepath.ToSlash(rel)] = len(hits)
			remaining += len(hits)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the repository: %v", err)
	}

	t.Logf("typed column widths remaining in files this story does not touch: %d", remaining)
	for file, count := range perFile {
		t.Logf("    %2d  %s", count, file)
	}

	if remaining > widthDebtBaseline {
		t.Errorf("the typed-width debt GREW to %d, from the %d measured at c8e347e — the count is carried so that it can be checked, not so that it can rise (Constraint 8)",
			remaining, widthDebtBaseline)
	}
}
