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
// # unfiltered is the WHOLE run's rows, and it is a parameter for one reason
//
// Being called after the narrowing costs this function the only population it
// cannot reconstruct: the rows the filter removed. Every other count is a field
// the producer maintained and survives untouched, but the two answers taken by
// walking the rows — `CompareRun.Unread` and the shortfall
// `func compareNotEvaluated` counts — would be taken over the narrowed view and
// would then shrink because the operator chose to look at less. A
// `--only-patched` run whose removed rows carried a failed reading would state
// a smaller gap than the same run unfiltered, and a filter would have made a
// holed run look whole — the one direction S047-R5.2 exists to forbid.
//
// So the caller keeps the slice before `func filterCompareResults` in
// overlay_compare.go replaces it, and hands it over here: the LISTS follow
// rep.Results and show what was asked for, the COUNTS follow this slice and
// state what the run did.
//
// It is a required parameter rather than an optional one because omitting it is
// exactly the defect above, and an omission that compiles is an omission that
// ships. A nil says something different and true: this report was never
// narrowed, so its own rows are the whole run.
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
func buildCompareReport(rep *overlay.CompareReport, repository string, unfiltered []overlay.CompareResult) report.Run {
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

	// The whole run's rows, whatever the operator asked to see. A caller that
	// never narrowed anything passes nil, and the report's own rows are then
	// already the whole run.
	whole := unfiltered
	if whole == nil {
		whole = rep.Results
	}

	reasons, _ := comparePackageFindings(rep.Findings)
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
	}

	// Unread counts every comparison nobody read, which is every reading state
	// except ReadingDone: a review nobody requested, one the content check made
	// impossible, and one that was attempted and failed all leave an operator
	// with a recommendation and no evidence behind it (S047-R3.1).
	//
	// It is walked over the WHOLE run rather than over the rows above, because
	// it is a count and it sits in the payload beside counts the producer
	// maintained over every package. One of the two shrinking under
	// `--only-patched` while the others did not would put two populations in one
	// section under one heading (S047-R5.2).
	for _, result := range whole {
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
	notEvaluated := compareNotEvaluated(rep, whole)
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
// # The rows are an ARGUMENT, so the count follows the run and not the view
//
// The call site narrows `CompareReport.Results` before this runs, so reading
// rep.Results here would count only the rows the operator asked to see: a
// `--only-patched` run would lose the shortfall of every row the filter removed
// and report itself closer to complete than it was. A choice about what to LOOK
// AT must not change what the run says it ESTABLISHED (S047-R5.2, S047-R5.3),
// so the population to walk is passed in and the caller passes the whole run's
// rows.
//
// The report is still the first parameter because the other population — the
// packages never dispatched — is a pair of counters on it, and those are not
// narrowed by anything.
func compareNotEvaluated(rep *overlay.CompareReport, results []overlay.CompareResult) int {
	// Population one: dispatched never, so established nothing.
	unreached := rep.TotalPackages - rep.ComparedPackages
	if unreached < 0 {
		unreached = 0
	}

	// Population two: reached, and still short of an answer.
	unestablished := 0
	for _, result := range results {
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

// comparePackageFindings splits the per-package findings in two: the FIRST
// finding each package has of its own, which becomes its row's reason, and
// every further finding, which becomes a note naming that package
// (S047-R6.1).
//
// # One walk, so the two halves cannot disagree
//
// They are returned together rather than by two functions because they are one
// partition: the same three exclusions decide both, and a second walk applying
// them again would be free to drift, in the direction where a finding is either
// stated twice or stated nowhere.
//
// # The second half exists because a ROW can hold one reason
//
// `ComparePkg.Reason` is one line and one record per package, so a package with
// three findings had two of them dropped here — measured on this repo, 11 of the
// 12 finding kinds are per-atom, so that is the normal case rather than an edge
// one. A note has no such budget: notes are wrapped by the renderer rather than
// cut to a column, which is why S047-R6.1 puts an explanation that does not fit
// a cell there.
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
// one — and is skipped by BOTH halves rather than filed under an empty key: no
// package is named by the empty string, so an entry there could only ever be
// read by accident. `func compareRunNotes` is where those are picked up.
func comparePackageFindings(findings []overlay.Finding) (map[string]string, map[string][]string) {
	reasons := make(map[string]string, len(findings))
	extra := make(map[string][]string)
	for _, finding := range findings {
		if finding.Kind == overlay.FindingCompared || finding.Atom == "" {
			continue
		}
		if _, seen := reasons[finding.Atom]; !seen {
			reasons[finding.Atom] = finding.Detail
			continue
		}
		extra[finding.Atom] = append(extra[finding.Atom], finding.Detail)
	}
	return reasons, extra
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
//
// # notes are what the RUN has to say about itself, and they are variadic
//
// `func compareNotes` builds them; a run with nothing extra to say passes
// none, and that is why the parameter is variadic where
// `func buildCompareReport`'s third one is required: passing no note is a true
// and complete answer, while passing no population would silently understate a
// gap. They reach the render and NOT the export — they are attached to the
// sections built here, and `func exportReport` in report_export.go serializes
// the run's payload, which by decision gains no field for them (S047-R1.5).
func presentCompareReport(cfg *config.Config, run report.Run, notes ...compareNote) {
	mode := reportModeOrPlain(cfg)
	content := report.SectionOptions{ShowAll: autoupdateAll}
	sections := appendCompareNotes(run.Sections(content), notes)
	if err := renderCheckReportIn(mode, sections, render.Options{}); err != nil {
		logger.Warn("the report could not be rendered: %v", err)
	}
	exportReport(run)
}

// appendCompareNotes puts every note where it is read: a note about a package
// under the section holding that package's rows, a run-scoped one last.
//
// # A package's section is found by its ROWS, never by its title
//
// `type Section` in internal/common/report/section.go carries no identifier, so
// the only two ways to recognise a block are its Title and what is in it. A
// Title is prose — the model's own doc says a consumer matching on one is
// matching on prose — and it is written in another package, where a reworded
// heading would silently send every note somewhere else. A row's first cells are
// this file's own output: `func comparePkgFacts` writes the atom into
// ComparePkg.Package, and the table carries it verbatim. So the search asks the
// question the placement actually depends on — which block shows this package —
// and it cannot be answered wrongly by an edit to a heading.
//
// # Last is the fallback, and it is the safe direction
//
// A package with no row anywhere — a Keep row not listed without --all, a run
// whose rows are all counted rather than shown — still has something said about
// it, and the note names its package, so the association survives the move. A
// note dropped for want of a home would be the report quietly losing a finding,
// which is the one outcome S047-R6.1 exists to prevent.
//
// # Run-scoped notes go last for the reason they always did
//
// `func (r Run) Sections` in internal/common/report/run.go PREPENDS an
// interruption block when a run is incomplete, so the FIRST block is not a fixed
// position: filing "no ::gentoo tree was reached" under "Run Interrupted" would
// explain one gap with another gap's sentence. The last block is the summary the
// payload always ends with.
//
// Writing into the slice is safe because `func (r CompareRun) Sections` in
// internal/common/report/compare_run.go builds fresh sections on every call, so
// nothing here reaches back into the payload. A run with no sections at all —
// reachable only through a nil payload — gets a section of its own rather than a
// lost note or a panic on an empty slice.
func appendCompareNotes(sections []report.Section, notes []compareNote) []report.Section {
	if len(notes) == 0 {
		return sections
	}
	if len(sections) == 0 {
		texts := make([]string, 0, len(notes))
		for _, note := range notes {
			texts = append(texts, note.text)
		}
		return []report.Section{{Title: "Run Notes", Notes: texts}}
	}

	last := len(sections) - 1
	introduced := make([]bool, len(sections))
	for _, note := range notes {
		target := last
		if note.atom != "" {
			if at := compareSectionOf(sections, note.atom); at >= 0 {
				target = at
			}
			if !introduced[target] {
				sections[target].Notes = append(sections[target].Notes, compareNotesLead)
				introduced[target] = true
			}
		}
		sections[target].Notes = append(sections[target].Notes, note.text)
	}
	return sections
}

// compareSectionOf is the index of the section showing atom, or -1 when no
// section shows it.
//
// It compares against every cell rather than only the first: the package column
// leads every table this payload builds, but a row is a slice and a table that
// later leads with something else would otherwise stop being searchable with no
// test able to see it. No other cell can hold an atom — the rest are versions, a
// status, a reading and a diff — so widening the search cannot match by accident.
func compareSectionOf(sections []report.Section, atom string) int {
	for i, section := range sections {
		for _, row := range section.Rows.Rows {
			for _, cell := range row.Cells {
				if cell == atom {
					return i
				}
			}
		}
	}
	return -1
}

// compareNote is one sentence the report says beside its tables, and the atom
// that decides WHERE it is said.
//
// The empty atom means run-scoped, which is the producer's own rule: a
// `type Finding` in internal/overlay/finding.go with no Atom is a fact about the
// run rather than about a package, and reusing that convention here means one
// idea is spelled one way on both sides of the seam. A note that names a package
// is placed with that package's rows by `func appendCompareNotes`; a run-scoped
// one goes last.
type compareNote struct {
	atom string
	text string
}

// compareNotesLead introduces a section's package notes once, so a reader meets
// a sentence rather than a list of loose strings under a table. It is only ever
// emitted where at least one such note follows it.
const compareNotesLead = "Beside the reason on each row, the run established more about these packages:"

// compareNotes is everything this run says outside its tables: what it has to
// say about ITSELF, then what it has to say about individual packages
// (S047-R1.5, S047-R6.1).
//
// The two are built by two functions and concatenated in that order, because
// that is the order they are read in: the run's own state qualifies everything
// under it, and a package's extra finding qualifies one row.
func compareNotes(rep *overlay.CompareReport, realignRan, judged, noReview bool) []compareNote {
	return append(compareRunNotes(rep, realignRan, judged, noReview), comparePackageNotes(rep)...)
}

// comparePackageNotes is what a package's row could not carry (S047-R6.1).
//
// # It walks the RESULTS, and that is what makes the order deterministic
//
// `func CompareWithProvider` in internal/overlay/compare.go sorts its results by
// category and then package, and `func EstablishFindings` builds the findings by
// walking those same results in that order. Iterating the results here — rather
// than the findings, or the map below — therefore produces the same sentences in
// the same order on every run over one overlay, which is the property S047-R2.4
// states for groups and which a note list needs for exactly the same reason: two
// runs a maintainer diffs must differ only where the overlay did.
//
// # It walks the NARROWED results, on purpose
//
// A note names a package a reader can see, so the population here is the rows
// the operator asked for and not the whole run. A note about a package
// `--only-redundant` removed would name a package with no row to be beside — the
// mirror of the counts, which follow the run precisely because they are NOT
// beside anything.
func comparePackageNotes(rep *overlay.CompareReport) []compareNote {
	if rep == nil {
		return nil
	}

	_, extra := comparePackageFindings(rep.Findings)
	if len(extra) == 0 {
		return nil
	}

	var notes []compareNote
	for _, result := range rep.Results {
		atom := result.Category + "/" + result.Package
		for _, detail := range extra[atom] {
			notes = append(notes, compareNote{
				atom: atom,
				// The package is NAMED in the sentence and not only used to
				// place it: a note is read as prose, several may sit under one
				// table, and a reader must not have to count rows to learn
				// which package a sentence is about.
				text: fmt.Sprintf("%s: %s", atom, compareOneLine(detail)),
			})
		}
	}
	return notes
}

// compareRunNotes is what a comparison run has to say about ITSELF: the two
// run-level facts the terminal output printed beside the table, which no row
// carries and `type CompareRun` in internal/common/report/compare_run.go has no
// field for (S047-R1.5).
//
// # The baseline fact is HARVESTED from the findings, not re-derived
//
// `func baselineRunFindings` in internal/overlay/annotate_baseline.go emits
// FindingBaselineSkipped with `CompareReport.BaselineSkipped` verbatim as its
// Detail — the producer has already turned the field into the sentence that
// names the tree it looked for and the marker it looked for in it. Reading the
// field here instead would be a second spelling of one fact, free to drift from
// the one the library publishes.
//
// It is RUN-scoped: its Atom is empty, and `func comparePackageFindings` skips exactly
// those, so no row carries it and it would be lost with nothing to say so. That
// skip and this harvest are two halves of one decision.
//
// # FindingRealignVerdict is NOT harvested here, and that is measured
//
// `func baselineResultsFindings` in internal/overlay/annotate_baseline.go writes
// it WITH an atom, so a realignment verdict is a fact about one package rather
// than about the run: `func comparePackageFindings` already files it as that package's
// Reason and the row carries it. Repeating it here would print every verdict
// twice, once against its package and once against the whole report.
//
// # The "no verdict at all" notice is derived from the flags, because nothing
// records it
//
// No finding is written when a review reaches no model — both realign counters
// simply stay zero — so the report renders silence, and silence there reads as
// "every divergence was judged and none objected". `func realignAddendum` in
// overlay_compare_realign.go derives the sentence from the same two flags this
// takes, for the reason stated there: the renderer never learns which flags were
// passed.
//
// # The baseline review's COVERAGE is rebuilt here, from the producer's counters
//
// `func formatBaselineSummary` in internal/overlay/annotate_baseline.go writes
// the same sentence today and is reached only from the renderer this story
// retires. The FACTS are read from the report; the SENTENCE is written here,
// because a library that formats a report line is the boundary story 046 closed
// (S047-D1) — internal/overlay establishes what is true, this file decides how a
// reader is told.
//
// The denominator is `CompareReport.ComparedPackages` and it may never be a len:
// Results is the VIEW, narrowed after the review has already run, so a share
// taken over it could print a fraction larger than one — and, worse, a
// believable one. `--only-outdated` is what separates the two, comparing three
// packages and showing two rows, and a report that answered "1 of the 2" there
// would be claiming it examined everything it was able to. It is the rule the
// counts in `func buildCompareReport` already follow, said once more because
// this sentence states a ratio and a ratio has two ways to go wrong (R6.4).
//
// It says what was FOUND rather than asserting anything about the packages the
// review never reached, which is the producer's own wording and the reason it is
// kept: with a status filter in play the review ranges over fewer packages than
// were compared. It renders nothing at zero, which is every run that asked for
// no review.
//
// ::gentoo is spelled here rather than taken from the producer's `baselineRepo`,
// which is unexported: the report's own vocabulary already names that tree in
// the section titles a reader meets above this note, and exporting a constant to
// save a word would publish one package's spelling as another's API.
//
// # What it deliberately does NOT carry
//
// The candidate declarations a `--realign` run proposes are per-package,
// multi-line paste blocks a maintainer copies, and every note is wrapped to the
// device by the renderer. They are printed by the run itself, beside the report,
// and the call site says why.
func compareRunNotes(rep *overlay.CompareReport, realignRan, judged, noReview bool) []compareNote {
	if rep == nil {
		return nil
	}

	var notes []compareNote
	for _, finding := range rep.Findings {
		if finding.Kind == overlay.FindingBaselineSkipped && finding.Atom == "" {
			notes = append(notes, compareNote{text: compareOneLine(finding.Detail)})
		}
	}

	if rep.NoBaselineCount > 0 {
		notes = append(notes, compareNote{text: fmt.Sprintf(
			"%d of the %d packages compared were found to have no ::gentoo counterpart — those are the overlay's own work rather than a divergence from anyone's, and no realignment is proposed for them.",
			rep.NoBaselineCount, rep.ComparedPackages)})
	}

	if realignRan && !judged {
		reason := "no model was reachable"
		if noReview {
			reason = "--no-review contacted no model"
		}
		notes = append(notes, compareNote{text: "Realignment verdicts: none was produced — " + reason +
			", so every divergence above carries no verdict, and an unjudged divergence is not a justified one. " +
			"Everything above was established by reading files and stands without a model."})
	}

	return notes
}
