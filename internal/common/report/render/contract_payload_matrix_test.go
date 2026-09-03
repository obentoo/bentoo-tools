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
//
// # The fourth payload, and the count that was never asserted (sub-task 16.3)
//
// Everything above was written and measured over THREE payloads, and the file
// said "three" while the CLI shipped four. `overlay validate` has emitted
// report.KindOverlayValidate since sub-task 8.1; everyKind never gained a run
// for it, and nothing counted the list, so the renderer contract S046-R7.4 rests on
// was checked over three kinds of four for four audits — passing each time,
// exactly as convincingly as it would have with all four, because a guard that
// sweeps a hand-written list cannot tell a short list from a complete one.
//
// 16.3 adds the run (validateDeclaration, below) and adds the thing that makes
// the addition stick: a CENSUS at the top of TestEveryPayloadEveryRenderer,
// against payloadKindCount and against the Kind constants the report package
// declares. The list is no longer trusted.
//
// Measured on 2026-09-02, against HEAD 0fd2f2e. Five mutations, each applied
// ALONE, run, and reverted immediately — every revert verified by md5sum
// against the pre-mutation file and, for the two files outside this sub-task's
// diff, by `git status --porcelain` showing them unmodified.
//
// M1 — the fourth run deleted from everyKind, restoring the three-of-four state
// this sub-task found:
//
//	--- FAIL: TestEveryPayloadEveryRenderer (0.00s)
//	    contract_payload_matrix_test.go: everyKind built 3 runs, want 4 — one
//	    per payload this CLI ships.
//	            A payload missing here is a payload the renderer contract is not
//	            checked over, and the sweep goes green either way (S046-R7.4, S046-R8.3).
//
// M2 — validateDeclaration.Sections made to answer one EMPTY section, the same
// mutation 10.6 applied to SnapshotRun:
//
//	--- FAIL: TestEveryPayloadEveryRenderer/overlay.validate (0.00s)
//	    contract_payload_matrix_test.go: plain rendered overlay.validate as nothing
//	    contract_payload_matrix_test.go: inline rendered overlay.validate as nothing
//	    --- PASS: autoupdate.check, overlay.manifest, snapshot.run
//
// The same two renderers of five, for the same reason 10.6 gives, and the three
// PASSes are what make the failure name the payload rather than the renderers.
// This is the mutation that proves the fourth entry is LIVE: it reaches all
// five renderers, and two of them can report on it.
//
// M3 — the fourth run's Kind changed to a duplicate of snapshot.run, so the
// COUNT still came to four:
//
//	--- FAIL: TestEveryPayloadEveryRenderer (0.00s)
//	    contract_payload_matrix_test.go: everyKind builds 2 runs of kind
//	    "snapshot.run" — a duplicate hides a payload that is missing, because the
//	    count still comes to 4
//
// M3 is why the census compares the SET and not only the length. A run
// duplicated and a run missing keep the total at four between them, and that is
// arithmetic no len() can see.
//
// M4 — a fifth constant, `KindProbeRun Kind = "probe.run"`, added to
// internal/common/report/run.go and deliberately not given a run here:
//
//	--- FAIL: TestEveryPayloadEveryRenderer (0.00s)
//	    contract_payload_matrix_test.go: the report package declares 5 Kind
//	    constants ([autoupdate.check overlay.manifest snapshot.run
//	    overlay.validate probe.run]), and payloadKindCount says 4 — the number
//	    here is stale, not the code
//
// M4 is the one that answers "who counts the counter". It is the failure the
// three-of-four hole should have produced when 8.1 added the kind and did not,
// because until 16.3 nothing here read run.go at all.
//
// M5 — `"validate.Report"` removed from payloadTypes (contract_test.go), the
// subject list this file's second guard sweeps:
//
//	--- FAIL: TestNoRendererSwitchesOnAPayloadType (0.00s)
//	    contract_payload_matrix_test.go: payloadTypes holds 4 entries
//	    ([report.Report report.AutoupdateCheck report.ManifestRun
//	    report.SnapshotRun]), want 5 — one per payload this CLI ships, plus
//	    report.Report for the pre-D1 name of the first.
//	--- PASS: TestEveryPayloadEveryRenderer (all four kinds)
//
// That PASS is the argument for censusing BOTH lists rather than one. The
// matrix and the subject list guard different halves of S046-R7.4 — that every
// payload survives every renderer, and that no renderer knows a payload — so a
// short subject list is invisible to a complete matrix, and was.
//
// Every transcript above is recorded WITHOUT this file's own line numbers, and
// that is deliberate rather than sloppy. A number a failing run prints is a
// coordinate in the file as it was WHEN IT RAN, and this header is inside the
// file the numbers point into: writing one down changes the length of the thing
// it is a coordinate for, and re-running to refresh it changes it again. The
// number was wrong before the paragraph recording it was finished, twice,
// which is how this note came to exist.
//
// It is not a hypothetical failure either. The 10.6 transcript higher up this
// header still says `contract_payload_matrix_test.go:182`, and line 182 has not
// been that assertion since the day it was written. That record is left exactly
// as 10.6 wrote it — it is that sub-task's evidence, not this one's to edit —
// and it is cited here as the measurement that decided this form.
//
// text.go's numbers ARE kept, because they are coordinates in a different file
// and they name the probe the mutation inserted, which exists only in the
// mutated file by definition. The register at
// .draft/red-evidence.yaml keeps every number verbatim, for the opposite
// reason: it is a separate file, so its copy does not move what it describes.

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/obentoo/bentoolkit/internal/common/report"
)

// payloadKindCount is how many payloads this CLI ships, and therefore how many
// runs everyKind must build: FOUR — autoupdate.check, overlay.manifest,
// snapshot.run and overlay.validate.
//
// The number is written down because it is the one this file got wrong.
// everyKind held THREE of the four from sub-task 8.1 — the change that started
// emitting `overlay.validate` — until 16.3, and nothing anywhere counted it: the
// literal below was read as "one run per payload" and believed. A guard that
// sweeps a hand-written list passes exactly as convincingly when the list is
// short, which is how three-of-four survived four audits.
//
// So the list is counted twice over, against two things that can disagree with
// it: against this constant, and against the report.Kind constants run.go
// actually declares (declaredKinds, below). The constant alone would be merely
// self-consistent; the derivation alone would not say what the number IS, and a
// reviewer would have nothing to check the diff against.
//
// Story 047 adds the fifth kind. When it does, this constant, everyKind and
// payloadTypes change together in one reviewable diff — which is the point of
// stating the number rather than trusting the literal.
//
// # Why everyKind and payloadTypes stay two lists, censused against one number
//
// They are two hand-maintained lists of the same four payloads, so the obvious
// tidy is to derive one from the other. It cannot be done, and the reason is
// what each list is FOR.
//
// everyKind holds RUNS — a kind, an envelope, and a fully populated payload
// VALUE, because a fixture carrying counts and no rows renders a sentence and
// no table, and a matrix over it would check that five renderers agree about
// prose while never reaching what they differ on (sub-task 5.5's finding, cited
// in everyKind itself). Those values cannot be generated from a type name:
// nothing knows a ManifestRun fixture needs four targets with one failure, and
// a generated zero value is exactly the vacuous fixture 5.5 ruled out.
//
// payloadTypes holds SELECTORS a renderer must not spell — text, matched
// against source — and two of its five entries have no run here at all, by
// design. report.Report is a name no type carries any more, kept so the pre-D1
// spelling cannot come back; validate.Report is a producer type that is not a
// payload, only the half of one a renderer could reach. A derivation would have
// to invent both, and inventing entries is what makes a guard hollow.
//
// So they stay separate, and are tied at the one place they can honestly meet:
// the COUNT. everyKind is censused against this constant directly, payloadTypes
// against this constant plus one for report.Report, and the constant itself
// against the kinds the report package declares. That is what "four names,
// three kinds, and they are not the same four" cost — closed without pretending
// either list can be computed from the other.
//
// The duplication that WAS removable has been removed:
// TestNoRendererSwitchesOnAPayloadType kept a second literal of payloadTypes
// and now derives its set from it through forbiddenPayloadSelectors. That is
// the drift that actually happened, rather than the one that merely looks
// untidy.
const payloadKindCount = 4

// validateDeclaration stands in for cmd/bentoo's `validatePayload`, the fourth
// payload — and it is a STAND-IN because the real one cannot be reached from
// here.
//
// # What it stands for, and why the substitution is forced
//
// `validatePayload` embeds validate.Report, and internal/common/report may not
// import internal/autoupdate (boundary_test.go's forbiddenImports), so the type
// that puts a validation run in the payload position sits at the adapter, in
// `package main` under cmd/bentoo. No Go file can import main. The route the
// other three entries take — build a literal of the payload type — is therefore
// closed for this one, and no amount of rearranging opens it while the payload
// lives where the boundary requires.
//
// It follows the precedent the other three already set at a smaller scale
// rather than inventing one: none of them calls its producer either, because
// TestRenderImportsNoProducer forbids importing one, so each is a literal
// standing in for what a producer builds. This is that same substitution one
// step further out — the payload type as well as the producer.
//
// # The SHAPE it contributes, which is why it earns an entry
//
// One block, a title, a lead, notes, and NO TABLE. The other three payloads all
// carry rows, so until this entry the matrix never asked what five renderers do
// with a section that is prose alone. That is not a hypothetical shape: it is
// what validatePayload returns, and the version of it that returned NOTHING
// left `overlay validate --export=report.md` writing a file of zero bytes until
// sub-task 10.1 — a failure two of these five renderers are the only ones able
// to report, exactly as the SnapshotRun mutation above found.
//
// # What it therefore does NOT prove
//
// It does not prove that validatePayload's own Sections produces this shape,
// and it cannot: what is asserted below is a fact about the RENDERERS, over a
// payload of this shape. The derivation is pinned where the type is —
// cmd/bentoo/overlay_validate_payload_test.go asserts what Sections returns
// over four validation runs, and overlay_validate_export_test.go drives it
// through a real `overlay validate --export`. If validatePayload stopped
// matching the shape here, THOSE two fail and this one does not.
//
// The prose is deliberately written as placeholder text rather than copied from
// the real block: a near-copy would be a second declaration of sentences that
// live in cmd/bentoo, wrong the first time somebody rewords one, and silent on
// both sides when it happened. Nothing below asserts the words — only that five
// renderers produce something recognisable from a block that has them.
//
// Story 047 ends the substitution by moving this payload into the report
// package with a model of its own, at which point this type is deleted and a
// report.ValidateRun literal takes its place beside the other three.
type validateDeclaration struct{}

// Sections returns the one block of the shape described above. The parameter is
// unnamed for the same reason the real payload's is: a declaration with no unit
// to list has nothing to shorten or expand.
func (validateDeclaration) Sections(report.SectionOptions) []report.Section {
	return []report.Section{{
		Title: "Overlay Validation",
		Lead:  []string{"stands in for the sentence naming what this document does not carry"},
		Notes: []string{
			"stands in for the note stating WHY that content is absent",
			"stands in for the note routing the reader to the document that does hold the run",
		},
	}}
}

// declaredKinds is every report.Kind constant the report package declares, read
// from that package's SOURCE rather than listed again here.
//
// This is what turns payloadKindCount from a number somebody typed into a
// number something measured. A fifth kind added to run.go and forgotten in
// everyKind makes the matrix fail, naming the kind that has no run — which is
// the failure the three-of-four hole went four audits without producing,
// because nothing was comparing the list to anything.
//
// It parses `..`, the report package's own directory one level up from this
// one. A Go test's working directory is its package directory, which is the
// same route cmd/bentoo's docs_test.go, width_debt_test.go and
// register_anchor_test.go take out to the repository root. Reading the source
// is the only route available: constants have no runtime representation, so
// neither reflection nor an import can enumerate them.
//
// A sweep that found nothing fails loudly instead of agreeing with an empty
// matrix — the same vacuity rule renderSourceFiles states for its own sweep.
func declaredKinds(t *testing.T) []report.Kind {
	t.Helper()

	const reportPackage = ".."

	entries, err := os.ReadDir(reportPackage)
	if err != nil {
		t.Fatalf("reading the report package directory %s: %v", reportPackage, err)
	}

	fset := token.NewFileSet()
	var kinds []report.Kind

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		path := filepath.Join(reportPackage, name)
		file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}

		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				// The type is stated on each spec — `KindX Kind = "…"` — so a
				// const block holding kinds beside anything else is read
				// correctly rather than swept in whole.
				declaredType, ok := value.Type.(*ast.Ident)
				if !ok || declaredType.Name != "Kind" {
					continue
				}
				for _, expr := range value.Values {
					lit, ok := expr.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					unquoted, err := strconv.Unquote(lit.Value)
					if err != nil {
						t.Fatalf("%s: reading the kind literal %s: %v", path, lit.Value, err)
					}
					kinds = append(kinds, report.Kind(unquoted))
				}
			}
		}
	}

	if len(kinds) == 0 {
		t.Fatalf("found no report.Kind constant under %s — the comparison below would agree with any matrix at all, including an empty one", reportPackage)
	}
	return kinds
}

// everyKind is one run per payload this story ships. A fifth command adds a
// line here and changes nothing else — which is the claim being checked.
//
// "One run per payload" is now ASSERTED by its caller rather than stated here
// and believed: see payloadKindCount, and the census at the top of
// TestEveryPayloadEveryRenderer.
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
		{
			Schema: report.SchemaVersion, Kind: report.KindOverlayValidate, Title: "overlay validation",
			// Complete: true, like the three above. The envelope's gap is not
			// what this matrix measures, and validate.Run has none anyway: a
			// cancelled sweep still lists every target it never reached
			// (cmd/bentoo/overlay_validate_report.go says why).
			Complete: true,
			// The fourth payload, added by sub-task 16.3 — absent from here
			// since 8.1 started emitting its kind, which meant the renderer
			// contract S046-R7.4 rests on was checked over three kinds of four.
			//
			// It is a STAND-IN, and validateDeclaration says at length what it
			// stands for and what it therefore does not prove. The short
			// version: the real type is `validatePayload` in `package main`,
			// and no Go file can import main.
			Payload: validateDeclaration{},
		},
	}
}

// TestEveryPayloadEveryRenderer drives the whole matrix: four payloads by five
// renderers, each of which must produce output the payload can be recognised
// in.
//
// "Produced output" is asserted as non-empty AND as naming the run — a renderer
// that answered "" for every payload would otherwise pass twenty times.
//
// # The census comes first, and it is what 16.3 added
//
// Twenty passing combinations say nothing about the payload that is not in the
// list. This test ran green over THREE of four from sub-task 8.1 onward, and
// read as though it covered all of them, because the loop trusted the literal
// it was given. So before any rendering happens the list is counted — against
// the number this package states (payloadKindCount) and against the kinds the
// report package declares (declaredKinds) — and the three must agree.
//
// It is a Fatal rather than an Error: a matrix run over the wrong set of
// payloads produces failures about renderers for a fault that is in the list,
// and the second reading is the expensive one.
func TestEveryPayloadEveryRenderer(t *testing.T) {
	runs := everyKind()

	if len(runs) != payloadKindCount {
		t.Fatalf("everyKind built %d runs, want %d — one per payload this CLI ships.\n"+
			"    A payload missing here is a payload the renderer contract is not checked over, and the sweep goes green either way (S046-R7.4, S046-R8.3).\n"+
			"    Remedy: add the missing run, or update payloadKindCount if a payload was genuinely retired.",
			len(runs), payloadKindCount)
	}

	declared := declaredKinds(t)
	if len(declared) != payloadKindCount {
		t.Fatalf("the report package declares %d Kind constants (%v), and payloadKindCount says %d — the number here is stale, not the code",
			len(declared), declared, payloadKindCount)
	}

	// Not just the count: the same four. A run duplicated and a run missing
	// keep the count at four between them, and that is the arithmetic a bare
	// len() cannot see.
	built := make(map[report.Kind]int, len(runs))
	for _, run := range runs {
		built[run.Kind]++
	}
	for _, kind := range declared {
		switch built[kind] {
		case 1:
		case 0:
			t.Fatalf("the report package declares the kind %q and everyKind builds no run for it — that payload goes through no renderer here, and this test passes anyway (S046-R7.4)", kind)
		default:
			t.Fatalf("everyKind builds %d runs of kind %q — a duplicate hides a payload that is missing, because the count still comes to %d", built[kind], kind, payloadKindCount)
		}
		delete(built, kind)
	}
	for kind := range built {
		t.Fatalf("everyKind builds a run of kind %q, which the report package does not declare — an export naming it is a document no consumer can filter for (S046-R4.4)", kind)
	}

	for _, run := range runs {
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
// miss. Twenty passing combinations are perfectly compatible with a renderer
// that gets there by asking which payload it holds — and that renderer is
// edited by every command added after it, which is precisely the cost R7.4
// exists to prevent.
//
// It inspects CASE CLAUSES rather than banning type switches: fullscreen.go
// switches on tea.Msg for its key handling, which is a renderer reading its own
// input and not a renderer reading a domain.
//
// # The subject list is payloadTypes', not one of its own (16.3)
//
// This kept a literal map of four names plus report.Payload, and contract_test.go
// kept a list of the same four, and the two were free to drift — which they had:
// four names beside three kinds, and not the same four. One list read twice
// cannot do that, so the set comes from payloadTypes with this test's own extra
// added.
//
// report.Payload is that extra, and it belongs only here. A renderer NAMING the
// interface is ordinary — it is what a payload is passed as — while a renderer
// SWITCHING on it is a renderer about to ask which one it holds.
func TestNoRendererSwitchesOnAPayloadType(t *testing.T) {
	// The census of the list this sweep depends on, for the reason
	// TestEveryPayloadEveryRenderer counts its own: a subject list that
	// silently lost an entry is a sweep that passes by looking for less. The
	// +1 is report.Report, the pre-D1 name of AutoupdateCheck — the one entry
	// that is a second spelling of a payload rather than a fourth payload.
	if want := payloadKindCount + 1; len(payloadTypes) != want {
		t.Fatalf("payloadTypes holds %d entries (%v), want %d — one per payload this CLI ships, plus report.Report for the pre-D1 name of the first.\n"+
			"    A payload with no entry is a payload no renderer is stopped from naming, and both sweeps over this list pass anyway (S046-R7.4, S046-R8.3).",
			len(payloadTypes), payloadTypes, want)
	}

	forbidden := forbiddenPayloadSelectors("report.Payload")

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
				if !isIdent {
					continue
				}
				// Qualified, for payloadTypes' reason: the fourth payload's
				// reachable half is validate.Report, and a sweep hardcoded to
				// one package could not see a case clause naming it.
				qualified := pkg.Name + "." + selector.Sel.Name
				if !forbidden[qualified] {
					continue
				}
				position := fset.Position(expr.Pos())
				t.Errorf("%s:%d switches on %s — a renderer deciding by payload type is edited by every command added after it (R7.4).\n"+
					"    Remedy: ask the payload for its sections; it is the only thing a renderer needs from a domain.",
					path, position.Line, qualified)
			}
			return true
		})
	}
}
