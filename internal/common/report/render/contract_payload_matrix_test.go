package render

// Authored for story 046, sub-task 9.2 — R7.4.
//
// Written from the contract: sub-task 9.2's objective — "The producer-to-
// consumer contract is verified across all three payloads and all renderers" —
// and story.md's Success Metric that a new kind costs no renderer edit. 6.3
// checks that no renderer NAMES a payload; this checks the other direction,
// that every payload actually goes through every renderer. A contract nothing
// exercised is a contract nobody has tested.
//
// Red on arrival: none of the three payloads exists.
//
// # Evidence that it still fails once they do (R8.3)
//
// The line above is a COMPILE failure, and a compile failure is not the rule
// this file states — it says only that the fixtures below name types task 1 had
// not written yet. It also stops being checkable the moment the package builds:
// from then on the matrix passes, and a green test proves nothing about what it
// would catch. R8.3 asks for the evidence a green guard cannot supply, and the
// story artifacts holding it (.draft/red-evidence.yaml) are not committed —
// `git ls-files .epic/` returns nothing — so a pointer to that file is a
// pointer to nothing for anyone who cloned this. citation_test.go makes that
// argument in full. The evidence is written down here instead, and it breaks
// the RULE rather than the build: a payload that does not survive a renderer.
//
// Measured on 2026-08-29, against sub-task 10.6. One mutation, applied on its
// own, run, and reverted immediately with `git checkout --`, the restore
// verified by `git status --porcelain` reporting nothing and by a second run of
// this test going green.
//
// Mutation: SnapshotRun.Sections in snapshot_run.go was made to answer with one
// EMPTY section. The payload still describes itself as a block, so it clears
// the len(blocks) check below and reaches all five renderers; it just puts no
// title, no lead, no row and no note inside the block —
//
//	func (r SnapshotRun) Sections(opts SectionOptions) []Section {
//	        return []Section{{}}
//	}
//
// Observed:
//
//	--- FAIL: TestEveryPayloadEveryRenderer (0.00s)
//	    --- FAIL: TestEveryPayloadEveryRenderer/snapshot.run (0.00s)
//	        contract_payload_matrix_test.go:182: plain rendered snapshot.run as nothing
//	        contract_payload_matrix_test.go:182: inline rendered snapshot.run as nothing
//
// Two renderers out of five, and the three that stayed silent are the argument
// for running all five. Markdown still wrote a "## " for the empty title, the
// export still carried the envelope wrapped around the empty payload and named
// its kind, and the fullscreen model still drew its border and its key hint:
// each of those three produces output of its OWN, so each would have answered
// "yes, the payload arrived" about a payload that had arrived as nothing. Only
// plain and inline print nothing but what the section holds, which is what
// makes them the two that can answer the question at all — and a matrix built
// on the export alone, the obvious economy, would have caught none of it.
//
// The other two payloads passed in the same run, which is the second half of
// what makes the failure usable: it names the payload that broke rather than
// condemning the renderers.
//
// It is authored beside contract_test.go rather than into it: 6.3's guard and
// this matrix fail for different reasons, and one file failing for two reasons
// is one file whose failure has to be read twice.

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/obentoo/bentoolkit/internal/common/report"
)

// everyKind is one run per payload this story ships. A fourth command adds a
// line here and changes nothing else — which is the claim being checked.
func everyKind() []report.Run {
	return []report.Run{
		{
			Schema: report.SchemaVersion, Kind: report.KindAutoupdateCheck, Title: "autoupdate check",
			Complete: true,
			Payload: report.AutoupdateCheck{
				Scanned: []report.PackageResult{{Package: "app-misc/jq", Type: "source", CurrentVersion: "1.7.1", CandidateVersion: "1.8.0", HasUpdate: true}},
				Plan:    []report.PlanEntry{{Package: "app-misc/jq", CurrentVersion: "1.7.1", CandidateVersion: "1.8.0", Class: "minor", Depth: "compile", Reason: "minor bump earns compile"}},
				Results: []report.ValidationRow{{Package: "app-misc/jq", CandidateVersion: "1.8.0", Outcome: report.Proved, Depth: "compile", Reason: "every deciding gate passed"}},
				Tally:   report.Tally{Proved: 1},
			},
		},
		{
			Schema: report.SchemaVersion, Kind: report.KindOverlayManifest, Title: "overlay manifest",
			Complete: true,
			Payload: report.ManifestRun{
				Ok: 3, Failed: 1,
				Targets: []report.ManifestTarget{
					{Package: "app-misc/jq", Success: true},
					{Package: "dev-lang/go", Success: true},
					{Package: "app-shells/fish", Success: true},
					{Package: "sys-apps/portage", Success: false, Error: "pkgdev exited 1"},
				},
			},
		},
		{
			Schema: report.SchemaVersion, Kind: report.KindSnapshotRun, Title: "snapshot run",
			Complete: true,
			// The construction differs from the ER sketch this file was written
			// against — Subvolumes is a slice and Steps holds the steps rather
			// than a count. Sub-task 6.2 records why: a run covers every
			// subvolume engine.subvolumes names, so one string could only be a
			// join or a lie, and a report that says "3" without saying WHICH
			// three does not satisfy R1.6. Call site only; no assertion moved.
			//
			// Both payloads above are given real rows for the same reason 5.5
			// established: a fixture carrying counts and no rows renders a
			// sentence and no table, so a matrix over it would check that five
			// renderers agree about prose and never reach the thing they
			// actually differ on.
			Payload: report.SnapshotRun{
				Subvolumes: []string{"/home"},
				Ok:         2, Failed: 1,
				Steps: []report.SnapshotStep{
					{Subvolume: "/home", Step: "create", Success: true},
					{Subvolume: "/home", Step: "prune", Success: true},
					{Subvolume: "/home", Step: "ship", Target: "offsite", Success: false, Error: "rclone exited 1"},
				},
			},
		},
	}
}

// TestEveryPayloadEveryRenderer drives the whole matrix: three payloads by five
// renderers, each of which must produce output the payload can be recognised
// in.
//
// "Produced output" is asserted as non-empty AND as naming the run — a renderer
// that answered "" for every payload would otherwise pass fifteen times.
func TestEveryPayloadEveryRenderer(t *testing.T) {
	for _, run := range everyKind() {
		t.Run(string(run.Kind), func(t *testing.T) {
			blocks := run.Sections(report.SectionOptions{})
			if len(blocks) == 0 {
				t.Fatalf("%s produced no section at all — a payload that describes itself as nothing renders as nothing in every mode", run.Kind)
			}

			opts := Options{Width: 100}

			var plain bytes.Buffer
			if err := Plain(&plain, blocks, opts); err != nil {
				t.Errorf("Plain: %v", err)
			}

			var markdown bytes.Buffer
			if err := Markdown(&markdown, blocks); err != nil {
				t.Errorf("Markdown: %v", err)
			}

			var jsonDoc bytes.Buffer
			if err := JSON(&jsonDoc, run); err != nil {
				t.Errorf("JSON: %v", err)
			}

			inline := captureStdout(t, func() error { return Inline(blocks, opts) })

			model := newModel(blocks, opts)
			sized, _ := model.Update(tea.WindowSizeMsg{Width: 104, Height: 200})
			fullscreen := sized.View()

			for _, mode := range []struct {
				name string
				out  string
			}{
				{"plain", plain.String()},
				{"markdown", markdown.String()},
				{"json", jsonDoc.String()},
				{"inline", inline},
				{"fullscreen", fullscreen},
			} {
				if strings.TrimSpace(ansi.Strip(mode.out)) == "" {
					t.Errorf("%s rendered %s as nothing", mode.name, run.Kind)
				}
			}

			// The export is the one renderer that sees the envelope, so it is
			// the one that must name the kind.
			if !strings.Contains(jsonDoc.String(), string(run.Kind)) {
				t.Errorf("the exported document does not name the kind %q:\n%s", run.Kind, jsonDoc.String())
			}
		})
	}
}

// TestNoRendererSwitchesOnAPayloadType is the failure mode a matrix alone would
// miss. Fifteen passing combinations are perfectly compatible with a renderer
// that gets there by asking which payload it holds — and that renderer is
// edited by every command added after it, which is precisely the cost R7.4
// exists to prevent.
//
// It inspects CASE CLAUSES rather than banning type switches: fullscreen.go
// switches on tea.Msg for its key handling, which is a renderer reading its own
// input and not a renderer reading a domain.
func TestNoRendererSwitchesOnAPayloadType(t *testing.T) {
	forbidden := map[string]bool{
		"Report": true, "AutoupdateCheck": true, "ManifestRun": true, "SnapshotRun": true, "Payload": true,
	}

	fset := token.NewFileSet()

	for _, path := range renderSourceFiles(t) {
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}

		ast.Inspect(file, func(node ast.Node) bool {
			clause, ok := node.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, expr := range clause.List {
				selector, ok := expr.(*ast.SelectorExpr)
				if !ok {
					// A pointer case: *report.ManifestRun.
					if star, isStar := expr.(*ast.StarExpr); isStar {
						selector, ok = star.X.(*ast.SelectorExpr)
					}
					if !ok {
						continue
					}
				}
				pkg, isIdent := selector.X.(*ast.Ident)
				if !isIdent || pkg.Name != "report" || !forbidden[selector.Sel.Name] {
					continue
				}
				position := fset.Position(expr.Pos())
				t.Errorf("%s:%d switches on report.%s — a renderer deciding by payload type is edited by every command added after it (R7.4).\n"+
					"    Remedy: ask the payload for its sections; it is the only thing a renderer needs from a domain.",
					path, position.Line, selector.Sel.Name)
			}
			return true
		})
	}
}
