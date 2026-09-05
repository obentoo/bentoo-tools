package main

// The seam between `overlay compare` and the report it now ends in.
//
// Story 047, sub-task 3.1 — S047-R1.1, S047-R1.3.
//
// internal/common/report must not import internal/overlay — `var
// forbiddenImports` in internal/common/report/boundary_test.go lists it, and
// `func TestPackageImportsNoPresentation` fails the suite for a single such
// import — and internal/overlay must not know what a report is. So the
// conversion from one to the other happens here and nowhere else (S047-D1).
//
// It is the same seam story 044 established at
// `func buildReport` in overlay_autoupdate_report.go and story 046 replicated
// at `func buildManifestReport` in overlay_manifest_report.go, and
// "replicated" is the word: every decision below has a counterpart in that
// second file, made for a reason that has not changed just because the domain
// did.
//
// # This command is the widest translation of the five
//
// A manifest target crosses as three fields. A compared package crosses as
// seven, four of which are ENUMS at the producer and words here. Two of the
// four carry a String() of their own in internal/overlay/compare.go — `type
// CompareStatus`, `type Verdict` — while `type Verification`, `type
// Authorship` and `type Reading` deliberately do not. That asymmetry is a
// decision rather than an omission, and the two functions that spell those
// words below say why at the point they spell them.

import (
	"fmt"
	"strings"

	"github.com/obentoo/bentoolkit/internal/common/config"
	"github.com/obentoo/bentoolkit/internal/common/logger"
	"github.com/obentoo/bentoolkit/internal/common/report"
	"github.com/obentoo/bentoolkit/internal/common/report/render"
	"github.com/obentoo/bentoolkit/internal/overlay"
)

// compareEnvelope puts one comparison's facts inside the envelope every
// exported document carries: the schema version, the kind of run that produced
// it, the reader's label for it, and how much of what it set out to establish
// it never reached (S047-R1.1).
//
// # It is the ONE place this command builds a report.Run
//
// The two values it stamps unconditionally are fixed per COMMAND rather than
// per run, and that is what makes `.kind == "overlay.compare"` a filter rather
// than a guess: the kind is derived from nothing this run establishes — not the
// repository, not how many packages matched, not whether anything was found
// redundant. A kind that varied with what a run found would answer one string
// for an empty run and another for a full one, so a consumer filtering on it
// would receive some of this command's documents and silently miss the rest.
//
// The schema is stamped from report.SchemaVersion for the reason the four
// envelopes before it give: a 2 typed at each producer is the same defect as a
// column width typed into a format string — several places to find on the day
// it changes, and nothing that fails when one is missed.
//
// # Complete and NotEvaluated are ARGUMENTS, never read off the payload
//
// Deriving them here — comparing the lists against the tally, or asking whether
// any row carries a failed reading — is a different rule wearing the same
// answer, and it is wrong in both directions. A run narrowed by
// `--only-redundant` would come out "incomplete" because most of its rows are
// missing, when it evaluated every one of them and the operator merely chose
// not to look (S047-R5.3); and an interrupted run whose last package happened
// to complete would come out complete. Completeness is established by the run
// that did the work, and it travels from there to here.
//
// The doc on `CompareReport.Interrupted` in internal/overlay/compare.go states
// the same rule from the producer's side, in the same words the manifest
// envelope uses: Complete is !Interrupted, and NEVER NotEvaluated == 0.
func compareEnvelope(payload report.CompareRun, complete bool, notEvaluated int) report.Run {
	return report.Run{
		Schema: report.SchemaVersion,
		Kind:   report.KindOverlayCompare,
		// A fixed label, exactly like the kind: it names the command that
		// produced the document in a reader's own words and is derived from
		// nothing the run established. report.Run.Title is documented as a label
		// rather than as something to match on, and Kind is what a consumer
		// discriminates with.
		Title:        "Overlay comparison",
		Complete:     complete,
		NotEvaluated: notEvaluated,
		Payload:      payload,
	}
}

// buildCompareReport turns what a comparison run returned into the run it
// reports, whole, before anything is printed (S047-R1.1, S047-R1.3).
//
// # It is a TRANSLATION and nothing else
//
// No package is filtered, no order is changed and no outcome is decided here.
// The order is the order `func CompareWithProvider` in
// internal/overlay/compare.go sorted the results into — category, then package
// — and it is a fact about the run rather than a presentation choice; sorting
// here would be this file deciding something the producer already decided.
//
// # It is called AFTER the view is narrowed
//
// `func filterCompareResults` in overlay_compare.go replaces
// `CompareReport.Results` in place, with only the rows `--only-redundant` and
// `--only-patched` asked to see, and the call site runs this function after it.
// The payload's LISTS therefore follow what the operator asked to see, while
// every count stays the producer's: each one below is read from a field
// `func CompareWithProvider` in internal/overlay/compare.go maintained, and
// never from len(Results). So a narrowed document states the whole overlay's
// tally beside the rows it shows, and still reports Complete — a filter is a
// choice the operator made, not a shortfall the run suffered (S047-R5.3).
//
// Building later loses nothing a reader had: `ComparePkg.Reason` is mapped by
// atom out of `CompareReport.Findings`, and `func EstablishFindings` in
// internal/overlay/compare.go fixes that slice onto the report before the
// filter runs, so every surviving row still finds the finding that explains it.
//
// # The counters are taken from the producer, never subtracted from each other
//
// `CompareRun.Scanned`, `InBoth` and `OnlyLocal` each come from a counter
// `CompareWithProvider` maintained. A subtraction taken here would invent an
// answer the run never gave and print it beside two the run did — the rule
// CompareRun's own field docs state at length — and the three would part
// company the first time a status is added.
//
// # A nil report is reported as an empty run, not as a crash
//
// The signature takes a pointer for the reason `func buildManifestReport` does:
// the caller is holding one, `CompareWithProvider` returns one, and copying a
// struct that carries five slices per result to pass it by value would be a
// copy nothing needs. A report is the thing an operator gets INSTEAD of a
// crash, so building one must not be the moment the crash arrives.
func buildCompareReport(rep *overlay.CompareReport, repository string) report.Run {
	payload := report.CompareRun{
		Repository:  repository,
		Redundant:   []report.ComparePkg{},
		NeedsRebase: []report.ComparePkg{},
		Keep:        []report.ComparePkg{},
		Unknown:     []report.ComparePkg{},
		KeepGroups:  []report.KeepGroup{},
	}
	if rep == nil {
		// No run to ask, so no gap to state: the empty report of a run that
		// established nothing, rather than one that claims to have been cut
		// short. Every slice is already non-nil above, so this document and a
		// full one differ in their rows and not in the shape of their keys.
		return compareEnvelope(payload, true, 0)
	}

	payload.Scanned = rep.TotalPackages
	// InBoth is the three statuses that REQUIRE a remote version to have been
	// read — outdated, newer, up-to-date — summed from the producer's own
	// counters. It is deliberately not TotalPackages minus NotInRemoteCount: a
	// package whose lookup errored is one the run does not know the remote
	// carries, and a subtraction would file it under "in both" on no evidence.
	payload.InBoth = rep.OutdatedCount + rep.NewerCount + rep.UpToDateCount
	payload.OnlyLocal = rep.NotInRemoteCount
	payload.Verdicts = report.VerdictTally{
		Keep:        rep.VerdictKeepCount,
		Redundant:   rep.VerdictRedundantCount,
		NeedsRebase: rep.VerdictNeedsRebaseCount,
		Unknown:     rep.VerdictUnknownCount,
	}

	reasons := compareReasons(rep.Findings)
	for _, result := range rep.Results {
		pkg := comparePkgFacts(result, reasons)
		switch result.Verdict {
		case overlay.VerdictRedundant:
			payload.Redundant = append(payload.Redundant, pkg)
		case overlay.VerdictNeedsRebase:
			payload.NeedsRebase = append(payload.NeedsRebase, pkg)
		case overlay.VerdictKeep:
			payload.Keep = append(payload.Keep, pkg)
		default:
			// VerdictUnknown, and any verdict added later without a case here.
			// The fail-safe direction is the one that recommends nothing: a
			// package this file has no list for must not silently join the list
			// that recommends deleting it.
			payload.Unknown = append(payload.Unknown, pkg)
		}

		// Unread counts every comparison nobody read, which is every reading
		// state except ReadingDone: a review nobody requested, one the content
		// check made impossible, and one that was attempted and failed all leave
		// an operator with a recommendation and no evidence behind it
		// (S047-R3.1).
		if result.Reading != overlay.ReadingDone {
			payload.Unread++
		}
	}

	// The call sub-task 2.3 left owing. GroupKeep is a package-level function
	// rather than a method that fills this field, so the assignment is written
	// where the payload is built and is visible here: a method would make a
	// correctly built payload depend on somebody having remembered to call it,
	// and an empty KeepGroups would then mean either "no version pair repeats"
	// or "nobody ran the rule" (S047-D2).
	//
	// It runs on the FULL Keep list, after it is built, because a group is a
	// fact the run established and travels as a field — a grouping performed
	// while rows are drawn would exist on a terminal and nowhere in the exported
	// file (S047-R2.1).
	payload.KeepGroups = report.GroupKeep(payload.Keep)

	// Complete and NotEvaluated, as S047-R5.1 and S047-R5.2 phrase them. A run
	// is complete when it was not cut short AND left nothing it set out to
	// establish unestablished; NotEvaluated is how many of those there were.
	//
	// The two are one answer read twice, which is why they are computed
	// together: a document stating Complete true beside a NotEvaluated of three
	// would give a machine reader two contradictory answers to one question.
	// Interruption stays a recorded FACT rather than a count — a run cut short
	// after its last package has a gap of zero and was still cut short — so it
	// is OR-ed in and never inferred (S047-R5.1).
	notEvaluated := compareNotEvaluated(rep)
	return compareEnvelope(payload, !rep.Interrupted && notEvaluated == 0, notEvaluated)
}

// compareNotEvaluated counts the facts this run set out to establish and did
// not (S047-R5.2).
//
// # Two populations, and they cannot overlap
//
// A package goes unestablished in two different ways, and the run records the
// two in different places.
//
// The first is a package the run never REACHED. The doc on
// `CompareReport.Interrupted` states that gap as TotalPackages minus
// ComparedPackages exactly: ComparedPackages++ runs once per result arriving at
// the collector, above the switch that splits results by status and outside the
// include filter, so a package missing from that number is a package whose
// worker never ran. The second is a package the run did reach and could not
// finish: a review that was killed, a version pair the content check refused, a
// lookup that errored (S047-R4.3).
//
// The two cannot double-count one package, because a package in the first
// population has no ROW. Results is appended under the same lock that
// increments ComparedPackages, so a worker that never ran left nothing for the
// loop below to look at. They are therefore added, not reconciled.
//
// The subtraction is clamped at zero. A CompareReport assembled by hand — in a
// test, or by a caller building one — may carry results without carrying the
// tally, and a negative gap would subtract from the second population and
// report a holed run as whole.
//
// # The second population is counted per PACKAGE, and that is what removes the
// double count
//
// `CompareReport.ErrorCount` is a field, so reading it looks like the better
// source than a walk. It is not, because it is not disjoint from the reading
// states: the review pass in internal/overlay/review.go writes
// ReadingNotComparable onto EVERY result whose Verified is NotVerified, and a
// package whose lookup errored was never content-verified, so it carries that
// state as well. ErrorCount plus a tally of the reading states would report one
// failed package as two unestablished facts, and the number a machine reads
// would exceed the number of packages that produced it.
//
// So the loop asks each row a single question and increments at most once: this
// package's outcome was not established, however many ways it went wrong.
//
// # ReadingNotRequested is excluded, and that exclusion IS S047-R5.3
//
// `--no-review` leaves every row at that zero value. Counting it would report a
// run the operator deliberately narrowed as a run with a hole in it, and a
// choice is not a shortfall. This is why the set here is NOT the set
// `CompareRun.Unread` counts: Unread is every state but ReadingDone, because it
// answers "did anybody read this difference", and "nobody was asked to" is a
// true answer to that question and a false one to this one.
//
// # A narrowed view can only lower this number, never raise it
//
// The call site filters Results before this runs, so a row `--only-redundant`
// removed takes its shortfall with it. That direction is the safe one:
// dropping rows can only move a run TOWARD complete, so a filter can never be
// the reason a run reports itself cut short (S047-R5.3). The cost is that a
// narrowed run can understate a shortfall among rows it was not asked to show;
// recovering it would take a counter maintained by the producer beside
// ErrorCount, which no field carries today.
func compareNotEvaluated(rep *overlay.CompareReport) int {
	// Population one: dispatched never, so established nothing.
	unreached := rep.TotalPackages - rep.ComparedPackages
	if unreached < 0 {
		unreached = 0
	}

	// Population two: reached, and still short of an answer.
	unestablished := 0
	for _, result := range rep.Results {
		switch {
		case result.Status == overlay.StatusError:
			// The lookup itself did not come back, so nothing downstream of it
			// was established either.
			unestablished++
		case result.Reading == overlay.ReadingFailed,
			result.Reading == overlay.ReadingNotComparable:
			// A review that was attempted and did not return, and a pair of
			// versions the content check refused. Both leave a recommendation
			// standing on evidence nobody has (S047-R4.3).
			unestablished++
		}
	}

	return unreached + unestablished
}

// comparePkgFacts is one compared package as the model spells it.
//
// # The atom is joined here, once
//
// Category and Package are separate on the producer's struct because the
// filesystem is; they are one string on the model's because every reader prints
// them as one. Joining at this boundary is what stops each renderer from
// picking its own separator, and it is the same crossing
// `func manifestTargetFacts` makes.
//
// # Status crosses through the producer's own String(), and Reading does not
//
// `type CompareStatus` in internal/overlay/compare.go has a String() and it is
// used: "up-to-date", "outdated", "newer" are the words that command has
// printed since before this report existed, and re-spelling them here would be
// a second vocabulary free to drift from the one the library publishes.
// `type Reading` and `type Verification` have no String() BY DESIGN — a display
// word is report vocabulary, not library vocabulary — so the words for them are
// written below, in this package, where the payload that consumes them is
// documented (S047-D1).
//
// # Authorship does not cross at all
//
// It is a fact about WHO WROTE a difference, and the producer has already put
// it into the sentence this row carries as its Reason: `func compareFindings`
// spells the proved case as "proved ours — our ebuild references <file>, which
// ::gentoo does not ship". A second field here would be a second answer to one
// question, and the one that drifted would contradict the sentence beside it.
func comparePkgFacts(result overlay.CompareResult, reasons map[string]string) report.ComparePkg {
	atom := result.Category + "/" + result.Package
	return report.ComparePkg{
		Package: atom,
		Local:   result.LocalVersion,
		Remote:  result.RemoteVersion,
		Status:  result.Status.String(),
		Reading: compareReadingWord(result.Reading),
		Diff:    compareDiffCell(result),
		Reason:  compareOneLine(reasons[atom]),
	}
}

// compareReadingWord is the report's word for whether anybody read this
// difference (S047-R3.1).
//
// # The words are written HERE because they are report vocabulary
//
// `type Reading` carries no String(), and that is deliberate: the four states
// are a fact the library establishes, while "not requested" is a word a reader
// meets in a document. A String() on the producer would publish this package's
// display choice as part of internal/overlay's API, where the next renderer
// would be free to use it and the one after free to change it.
//
// The four strings are `const readingNotRequested` and its three neighbours in
// internal/common/report/compare_run.go, which counts them to build the
// redundant section's lead. They are spelled in two packages and cannot be
// shared — the constants are unexported, and exporting them would publish a
// vocabulary a consumer would then be entitled to match on. A typo in either
// half shows up as a count of zero rather than as a failure, so
// `func TestBuildCompareReport` asserts the four words literally.
//
// # There is no blank cell
//
// A Reading value with no word — reachable only if a fifth state is added
// without a case here — reads as "not requested", which is the fail-safe
// direction the redundant section already applies to an unmapped word: it can
// cost a report a removal recommendation it could have made, and never earn one
// over evidence nobody has.
func compareReadingWord(reading overlay.Reading) string {
	switch reading {
	case overlay.ReadingNotComparable:
		return "not comparable"
	case overlay.ReadingFailed:
		return "failed"
	case overlay.ReadingDone:
		return "read"
	case overlay.ReadingNotRequested:
		return "not requested"
	default:
		return "not requested"
	}
}

// compareDiffCell is what the content check found, drawn from the closed
// vocabulary S047-R3.2 fixes: "+N/-M", "identical", "not compared",
// "unreadable".
//
// The vocabulary is closed so that "no difference was found" can never be read
// as "no comparison was made". Those two are the same blank cell in the output
// this payload replaces, and they are opposite answers: one says the overlay's
// copy is redundant, the other says nobody has established anything about it.
//
// # "unreadable" has no producer today, and that is a MEASUREMENT not an omission
//
// `func verifyAgainstLocalContent` in internal/overlay/compare.go returns the
// same zero contentCheck — NotVerified, no magnitude — for all four of its
// exits: the package not resolving on both sides, the two versions differing,
// our ebuild failing to read, and upstream's failing to read. The cause is
// collapsed at the producer, and `type Reading` cannot recover it either:
// ReadingNotComparable is written on Verified == NotVerified whatever the
// cause, and ReadingFailed is about a REVIEW that did not come back rather than
// about a file that would not open.
//
// So this function emits three of the four words. Printing "unreadable" for any
// state it can actually observe would state a cause the run never established,
// which is precisely what a closed vocabulary exists to prevent. The word stays
// in the payload's documented vocabulary for the producer that later splits
// NotVerified into its causes; nothing here or downstream changes when it does.
//
// # The magnitude is stated only where a comparison found one
//
// DiffAdded and DiffRemoved are meaningful ONLY at VerifiedDiffers and are zero
// otherwise, so they are read inside that arm and nowhere else. A "+0/-0" on a
// check that never ran would be indistinguishable from a byte-identical pair,
// which is the conflation the three other words exist to remove.
func compareDiffCell(result overlay.CompareResult) string {
	switch result.Verified {
	case overlay.VerifiedDiffers:
		return fmt.Sprintf("+%d/-%d", result.DiffAdded, result.DiffRemoved)
	case overlay.VerifiedIdentical:
		return "identical"
	case overlay.NotVerified:
		return "not compared"
	default:
		// A Verification value with no word — reachable only if a fourth state
		// is added without a case here. "not compared" is the fail-safe answer:
		// it withholds a recommendation rather than claiming evidence.
		return "not compared"
	}
}

// compareReasons indexes the findings by atom, keeping the FIRST finding each
// package has of its own.
//
// # FindingCompared is skipped, and skipping it is the whole rule
//
// `func compareFindings` in internal/overlay/compare.go emits one
// FindingCompared per package — "::gentoo ships X and the overlay carries Y" —
// and then a second, louder entry for the packages that also carry a
// divergence, a stale declaration or a registry declaration. The row-level
// entry restates Status, which this row already carries as its own cell, so
// carrying it as a Reason would print the STATE column again under every line.
//
// It would also, measurably, empty the report of its groups: `func GroupKeep`
// refuses to absorb a package whose Reason is non-empty, because grouping
// compresses repetition and a finding is not repetition (S047-R2.2). With every
// package carrying a restated status, no version pair would ever collect two
// members and S047-R2.1 would produce nothing on any run.
//
// # FIRST, in the order the findings were established
//
// `func EstablishFindings` appends the run-level baseline finding, then the
// comparison's, then the baseline review's per-package ones. So the first
// non-FindingCompared entry for an atom is the comparison's own exception where
// there is one, and a baseline finding only where there is not — which is the
// priority a reader wants: what is wrong with this package outranks what a
// later pass measured about it.
//
// The test is on the KIND rather than on a list of the kinds that qualify. A
// list would be a hand-maintained registry of the sort story 047's own design
// counts four of, wrong the first time a producer adds a finding and forgets
// it; the complement is one entry and cannot go stale.
//
// A finding with an empty Atom is run-scoped — FindingBaselineSkipped is the
// one — and is skipped rather than filed under an empty key: no package is
// named by the empty string, so an entry there could only ever be read by
// accident.
func compareReasons(findings []overlay.Finding) map[string]string {
	reasons := make(map[string]string, len(findings))
	for _, finding := range findings {
		if finding.Kind == overlay.FindingCompared || finding.Atom == "" {
			continue
		}
		if _, seen := reasons[finding.Atom]; seen {
			continue
		}
		reasons[finding.Atom] = finding.Detail
	}
	return reasons
}

// compareOneLine collapses every run of whitespace in s to a single space.
//
// A Row.Detail may contain no newline: a raw one corrupts both the plain writer
// and the Markdown pipe table. `ComparePkg.Reason`'s own doc says the adapter
// folds it, and this is the adapter. strings.Fields splits on any whitespace
// and discards none of the text between, so the result holds every visible
// character the original did, in order — a fold of the layout rather than a
// loss of content, which is what keeps S047-R6.3's "every explanation in full"
// true of the exported document.
//
// It is spelled here rather than shared with `func foldToOneLine` in
// internal/common/report/manifest_run.go, which is unexported: exporting it
// would publish a helper as API to save one line.
func compareOneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// presentCompareReport puts the finished comparison in front of the operator:
// the terminal first, then the export (S047-R1.2, S047-R1.3).
//
// It is `func presentManifestReport` in overlay_manifest_report.go with nothing
// added and nothing taken away, and that sameness is the requirement rather than
// a coincidence: S047-R1.2 asks that this payload reach the operator through the
// same renderer the other four kinds use, with no renderer edited to accommodate
// it. Three steps — resolve the mode, build the sections once, export the run —
// and not one of them mentions a package, a verdict or a diff. If a fifth
// command ever needs a step this one does not have, that is the signal the
// envelope stopped fitting, and the place to answer it is the envelope.
//
// # It must be called AFTER the live progress region is down
//
// `overlay compare` drives a live region through CompareOptions.ProgressCallback
// while it walks the overlay, and this writes to the same stdout. Rendering
// while that region is still up would draw the report into a frame the UI then
// redraws over, so the caller's finishUI() — the one point in the compare run
// where the terminal has been handed back — is the earliest this function may be
// called. That ordering lives at the CALL SITE because only the call site holds
// the teardown, which is the same division `func presentManifestReport` states
// for its own producer. It is also the display half of S047-R7.1: the
// interactive and building work a run does finishes before any alternate screen
// is opened, and the report is drawn once the screen is closed again.
//
// # The terminal render happens first, and that ordering is S047-R1.3
//
// An export is a convenience; the report on the terminal is the answer. Writing
// the file first would let a bad path — a directory that does not exist, a
// read-only mount — cost the operator the comparison they waited on, which for
// this command is the most expensive report of the five to produce: it reads a
// remote tree and diffs ebuilds to get there. Rendering first makes that loss
// impossible rather than merely unlikely, and `func exportReport` in
// report_export.go returns nothing precisely so a failed export cannot reach
// into this run's exit status either.
//
// # The mode is resolved HERE, not at the top of the run
//
// `func reportModeOrPlain` in overlay_autoupdate_ui.go is a pure function of the
// flags, the configuration and the terminal, so asking at this point gives the
// same answer any other producer gets, and asking late costs nothing. It also
// cannot fail: an unusable BENTOO_UI or ui.mode is refused out loud, the report
// falls back to plain and the exit status does not move. Resolving at the top of
// the run instead would turn a typo in a shell profile into "your comparison did
// not run" — a comparison that already did all of its work and has an answer to
// give.
//
// # ShowAll is answered ONCE, and it is content rather than device
//
// report.SectionOptions says what the report should SAY; render.Options says
// what the device ALLOWS. --all is the first kind, so it is read here where the
// sections are built and nowhere downstream. render.Options is passed empty
// because its only field is Width and a Width of 0 means "ask the device": a
// number typed at a call site would be this command deciding how wide somebody
// else's terminal is. The export does not share that answer — `const
// unshortenedWidth` in overlay_autoupdate_ui.go renders the plain export at a
// budget no report can reach, so every explanation crosses in full whatever the
// screen was rendered at (S047-R6.3).
//
// # A render that fails is reported and does not stop the export
//
// The two are independent answers to the same report. A terminal that went away
// mid-write is no reason to also withhold the file, which may be the only copy
// of the comparison left.
func presentCompareReport(cfg *config.Config, run report.Run) {
	mode := reportModeOrPlain(cfg)
	content := report.SectionOptions{ShowAll: autoupdateAll}
	if err := renderCheckReportIn(mode, run.Sections(content), render.Options{}); err != nil {
		logger.Warn("the report could not be rendered: %v", err)
	}
	exportReport(run)
}
