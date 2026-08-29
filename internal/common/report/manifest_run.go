package report

import (
	"fmt"
	"strings"
)

// ManifestRun is everything one `overlay manifest` run established: the targets
// it was given, what became of each of them, and the two counts that summarize
// the lot (R1.5). It is the Payload behind KindOverlayManifest.
//
// # The type and its Sections share one file, unlike AutoupdateCheck
//
// The check's fields are declared in model.go and its Sections in
// autoupdate_check.go, which is where story 044 left them: those fields predate
// the Payload interface by a whole story. A payload written after that seam
// exists has no such history, and one file per payload is what makes "a fourth
// kind of run is a new file" literally true. The alternative is a model.go that
// grows one struct per command, which is the edit-per-kind the interface exists
// to prevent (D1).
//
// # Two columns, and that is precisely why a tally is a payload's business
//
// The check counts four validation outcomes; a manifest target either had its
// Manifest regenerated or it did not. One universal tally on the envelope would
// have to either lose the four or invent columns this run has no answer for,
// which is the asymmetry Run's own doc comment states from the other side (D1).
//
// # It says nothing about whether the run finished
//
// "The run reached the end of its target list" and "this many targets were
// never reached" are true of any batch, so they are the envelope's
// (Run.Complete, Run.NotEvaluated) and are deliberately absent here. A payload
// that restated them would be a second place for a finished run to be described
// as a partial one, and the two could disagree inside a single document.
type ManifestRun struct {
	// Targets is every package the run was given, in the order it was given
	// them, whatever became of each.
	//
	// A nil slice reaches the JSON export as null and an empty one as [], and
	// the two say different things: the first is a producer that established
	// nothing, the second a run that was handed no target at all. Neither is
	// rewritten into the other on the way out (R4.3).
	Targets []ManifestTarget `json:"targets"`
	// Ok is how many targets had their Manifest regenerated, and Failed is how
	// many did not (R1.5).
	//
	// # They are FIELDS here although the producer derives them
	//
	// overlay.ManifestResult answers the same two questions with methods over
	// its own slice, and says at length why storing them there would be a
	// second copy of what the rows already state. Both are right, because the
	// two types are read by different readers. That one is read by Go code,
	// which can call a method; this one is read by a program holding the
	// exported document, which cannot — a count that exists only as a method
	// is a count the JSON export does not carry, and R1.5 asks for values
	// every renderer AND the export read from the report.
	//
	// # They are established by the run, never summed from Targets
	//
	// A dry run is the case that makes the difference load-bearing.
	// RegenerateManifests marks every previewed target Success false — nothing
	// ran, so nothing succeeded — so a sum taken here would report that every
	// package in the overlay failed a run that invoked no command at all. The
	// adapter that builds this payload branches on the preview BEFORE it
	// counts, exactly as the producer's own formatter does, and hands the
	// answer down. That is the same rule Run.Complete follows: a fact
	// established by the run travels from the run, rather than being re-derived
	// by whoever happens to hold the result.
	//
	// # Neither carries omitempty, and that is load-bearing
	//
	// A zero dropped from the document reads as "the producer never said",
	// which is exactly the conflation these two exist to remove: "no target
	// failed" and "nobody counted" are different answers, and a run where every
	// target succeeded is the common case that would lose its zero.
	Ok     int `json:"ok"`
	Failed int `json:"failed"`
	// DryRun reports that the run was a preview: the targets below were
	// resolved and listed, and nothing was invoked for any of them.
	//
	// It is a fact about what the run DID, not about how it is displayed —
	// which is what earns it a place in a model that holds no presentation. It
	// is also the one thing that explains the shape of everything else here: on
	// a preview Ok and Failed are both zero while Targets is full, and without
	// this field a reader meeting those three numbers would have to guess
	// whether the run failed silently or never started. Sections states it in
	// words for the same reason (R2.3).
	DryRun bool `json:"dry_run"`
}

// ManifestTarget is what the run established about one package.
type ManifestTarget struct {
	// Package is the atom, "<category>/<package>", as an operator types it.
	//
	// The two halves are joined by the adapter rather than carried apart,
	// because every reader of this field prints the atom: carrying them
	// separately would leave each renderer to re-join them with a separator of
	// its own, and two renderers spelling one package name differently is the
	// disagreement the model exists to make impossible.
	Package string `json:"package"`
	// Success reports that this target's Manifest was regenerated.
	//
	// It is false for a target that was merely previewed, and that reading is
	// the producer's: "did not succeed" covers "never attempted". Which of the
	// two happened is said by DryRun above, once for the run, rather than by a
	// third state per target — a distinction the run makes once cannot drift
	// between rows.
	Success bool `json:"success"`
	// Error is why this target failed, in the producer's own words, and empty
	// for a target that did not fail.
	//
	// It carries no atom: Package is right beside it, and a copy of the atom
	// in here would be printed twice on every failure line.
	Error string `json:"error"`
	// Output is what the failing command printed, verbatim — the diagnostic the
	// operator acts on, kept because by the time a report is rendered the
	// terminal that streamed it live is gone (R5.2).
	//
	// It is populated only on failure, and that is the producer's decision
	// rather than this type's: the capture is unbounded, and a whole-overlay
	// run would otherwise hold one copy per target for no reader. Sections
	// states that omission rather than leaving a reader to wonder where a
	// successful target's output went (R2.3).
	Output string `json:"output"`
}

// Sections is the manifest run as structure: what it says, in order, with every
// value at full length and not one decision about how it will look.
//
// # One block, because the run establishes one thing
//
// The check produces four sections because it answers four questions — what was
// scanned, what was planned, what the gates said, and how it all tallies. A
// manifest run answers one: for each target, was the Manifest regenerated. A
// second section would be a heading with nothing under it that the first does
// not already say, and a report padded with empty blocks trains a reader to skim
// past the one that matters.
//
// # ShowAll decides what is LISTED and never what is counted
//
// A target that succeeded has nothing to report about, which is exactly the
// class of unit SectionOptions.ShowAll governs: it is listed when the flag asks
// and counted otherwise. The count is taken from the run's own Ok, never from
// the rows, so the number under the table cannot move when the listing does
// (R8.3) — and the note says the list is short and why, so the omission is
// stated rather than silent (R2.3, R2.5).
//
// A FAILURE is always listed, whatever the flag says. It is the reason an
// operator is reading the report at all, and a report that hid failures behind
// a flag would be answering a question nobody asked.
//
// # A preview lists everything, and ShowAll has nothing to do with it
//
// On a dry run the list IS the finding: the operator asked what would be
// processed, and the answer is the names. There is no unit "with nothing to
// report about" to hold back, so the flag governs nothing here — which is why
// the preview arm below never reads it.
func (r ManifestRun) Sections(opts SectionOptions) []Section {
	return []Section{manifestSection(r, opts.ShowAll)}
}

// manifestSection is the block: the counts, the rows they were taken over, and
// the sentences saying what was left out of the rows and why.
func manifestSection(r ManifestRun, listEvery bool) Section {
	s := Section{Title: "Manifest Regeneration"}

	// A run with no row still produces a report, and the report still says so in
	// a sentence rather than as an empty table under a heading — a report is the
	// thing an operator gets INSTEAD of a crash, so reading one must not be
	// where the crash arrives.
	//
	// # Two different runs arrive here, and the sentence must be true of both
	//
	// One was handed no target at all. The other was handed plenty and was
	// interrupted before a single one reached an outcome, which is the case
	// S046-R1.4 made reachable: the producer hands back only the targets it
	// answered for, so an early enough interrupt hands back none.
	//
	// The sentence therefore states what is true of both — there is nothing to
	// count — and does NOT claim a cause. Saying "no target was selected" here
	// would tell an operator who selected three packages and pressed ctrl+c that
	// they had selected none. Which of the two happened is the ENVELOPE's to
	// say, and it does: Run.Sections puts the `Run Interrupted` block above this
	// one exactly when the second is the case, with the count this section has
	// no way to know.
	// # The guard is on the COUNTS, not on the rows, and that is not pedantry
	//
	// Ok and Failed are established by the run and Targets is a separate field
	// of the same struct, so "no rows" and "nothing counted" are two facts and
	// nothing in the type makes them agree. The adapter sets both from one
	// result today, which is why production never reaches the gap — but the
	// sentence below is a CLAIM ABOUT THE COUNTS, and asserting it from the
	// length of a different field is how a report starts lying about the one
	// thing R1.5 asks it to state. A payload carrying "3 regenerated, 1 failed"
	// and no rows would have printed "nothing is counted as regenerated or as
	// failed", which is false, and false in the direction that matters: the
	// reader is told a run did nothing when it did.
	if r.Ok+r.Failed == 0 && len(r.Targets) == 0 {
		s.Lead = []string{"No target has an outcome to report, so nothing is counted as regenerated or as failed."}
		return s
	}

	// Counts without the rows they were taken over. The counts are still stated,
	// because they are what R1.5 asks for and they are what the run
	// established; what is missing is the per-target detail, and R2.3 says a
	// report states what it omitted rather than omitting it silently.
	if len(r.Targets) == 0 {
		s.Lead = []string{fmt.Sprintf("%d target(s) processed: %d regenerated, %d failed.",
			r.Ok+r.Failed, r.Ok, r.Failed)}
		s.Notes = []string{"No per-target row reached this report, so the counts above are stated without the list they were taken over."}
		return s
	}

	if r.DryRun {
		return manifestPreview(s, r)
	}

	s.Lead = []string{fmt.Sprintf("%d target(s) processed: %d regenerated, %d failed.",
		r.Ok+r.Failed, r.Ok, r.Failed)}

	s.Rows.Headers = []string{"PACKAGE", "STATE"}
	for _, target := range r.Targets {
		if target.Success && !listEvery {
			// Counted in the note below, not listed here (R2.5).
			continue
		}
		s.Rows.Rows = append(s.Rows.Rows, Row{
			Cells:  []string{target.Package, manifestState(target)},
			Detail: manifestDetail(target),
		})
	}

	s.Notes = manifestNotes(r, listEvery)

	return s
}

// manifestPreview is the block a --dry-run produces: the targets, and the plain
// statement that nothing happened to any of them.
//
// # The count in the lead is len(Targets), and NOT Ok+Failed
//
// Those two are zero here, deliberately and correctly — no target succeeded and
// none failed, because none was attempted — so a lead that read them would
// announce "0 target(s)" over a table of names. What the operator asked for is
// how many the run WOULD process, and that is the length of the list under the
// sentence.
//
// # The note is the whole point of this arm
//
// Without it a reader meets a full table of packages beside two zero counts and
// has to guess which of the two is lying. Saying "nothing ran" once, in words,
// is what makes both readings correct at the same time, and it is the omission
// R2.3 asks to be stated: this run deliberately established nothing about any
// of the rows it is showing.
func manifestPreview(s Section, r ManifestRun) Section {
	s.Lead = []string{fmt.Sprintf("Dry run: %d target(s) would have their Manifest regenerated.", len(r.Targets))}

	s.Rows.Headers = []string{"PACKAGE"}
	for _, target := range r.Targets {
		s.Rows.Rows = append(s.Rows.Rows, Row{Cells: []string{target.Package}})
	}

	s.Notes = []string{
		"Nothing ran: no command was invoked and no Manifest was written, so no target above is counted as regenerated or as failed.",
	}

	return s
}

// manifestNotes is what the table left out, and why (R2.3, R2.5).
//
// Both sentences are conditional on there having been a success, because both
// are about the successful targets: a run in which everything failed listed
// every one of its rows and kept every one of their diagnostics, so it has
// nothing to apologise for and gets no note at all.
func manifestNotes(r ManifestRun, listEvery bool) []string {
	if r.Ok == 0 {
		return nil
	}

	var notes []string
	if listEvery {
		// The count is stated even when the rows are all present, for the
		// reason the check's own up-to-date note is: a count never depends on
		// which rows were listed (R8.3), so the sentence is the same shape in
		// both directions and only its tail changes.
		notes = append(notes, fmt.Sprintf("%d target(s) were regenerated and are listed above.", r.Ok))
	} else {
		notes = append(notes, fmt.Sprintf("%d target(s) were regenerated and are not listed; pass --all to list them.", r.Ok))
	}

	return append(notes,
		"What a target that succeeded printed is not kept: it was shown while the run was live, and holding a verbatim copy of every worker's output would grow with the overlay. A target that failed carries its own, in full.")
}

// manifestState is the word the STATE column prints. Only the wording is decided
// here; which of the two applies was decided by the run, and Success is the one
// field that says it.
func manifestState(target ManifestTarget) string {
	if target.Success {
		return "regenerated"
	}
	return "failed"
}

// manifestDetail is why a failed target failed: the producer's sentence, and
// then everything its command printed.
//
// # A successful target has no detail, which is not the same as having none to show
//
// It returns the empty string, which every writer already omits rather than
// printing a bare indent. The output that would have gone here was never kept
// (see ManifestTarget.Output), and manifestNotes is where that is said — once
// for the run rather than once per row, because it is one decision the producer
// made and not a property of any individual target.
//
// # The captured output is JOINED, not shortened
//
// Every non-whitespace character of it survives into the sentence; only the line
// structure is folded away, and it has to be. A Row.Detail is one line by
// definition — the plain writer prints it under its row and the Markdown writer
// makes it a table cell, and a raw newline corrupts the layout of both — while
// the verbatim bytes stay on ManifestTarget.Output, which is what the JSON
// export carries. So nothing is lost by the fold: the same run's document holds
// the output exactly as the command wrote it.
//
// This is the shape the check's failureDetail already established for gate
// findings: joining is not shortening, because no finding is dropped and no
// detail is truncated. What a writer with a line budget then does with a long
// sentence is a rendering decision, made in a renderer, and marked where it cuts
// (R6.4, R7.4).
func manifestDetail(target ManifestTarget) string {
	if target.Success {
		return ""
	}

	var parts []string
	if target.Error != "" {
		parts = append(parts, target.Error)
	}
	// Labelled, because the two halves are different kinds of statement: the
	// first is the toolkit's account of the failure and the second is the
	// child's own. A reader meeting them run together would have no way to tell
	// where one ended.
	if printed := foldToOneLine(target.Output); printed != "" {
		parts = append(parts, "output: "+printed)
	}

	return strings.Join(parts, "; ")
}

// foldToOneLine collapses every run of whitespace in s to a single space.
//
// strings.Fields splits on any whitespace and discards none of the text between,
// so the result holds every visible character the original did, in order — a
// fold of the layout rather than a loss of content. Leading and trailing
// whitespace disappears with it, which is what keeps a command whose output ends
// in a newline from contributing a detail that is nothing but a space.
func foldToOneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
