package report

import (
	"fmt"
	"strings"
)

// SnapshotRun is everything one `snapshot run` established: the subvolumes it
// set out to operate on, the steps it ran over them, and the two counts that
// summarize the lot (R1.6). It is the Payload behind KindSnapshotRun.
//
// # This is the payload the envelope was designed to be tested by
//
// design.md D6 picked `snapshot run` for its DISTANCE from the check path, and
// the distance is visible in the field list below: not one of these names is a
// package, a version, an ebuild or an atom, and no field of Run had to grow to
// carry them. Everything domain-specific about a snapshot run — a subvolume, a
// pipeline step, a ship target — is HERE, and everything the envelope states
// about it ("the run reached the end of its plan", "this many planned units
// were never reached") is stated in words that never mention btrfs. That is D1
// holding under a second vocabulary rather than being asserted about one.
//
// # Two columns, and they are not the manifest's two
//
// A step either did what it was asked or it did not, which happens to be the
// same arity `overlay manifest` counts with — and it is a coincidence, not a
// shared type. The manifest counts REGENERATED targets and this counts
// SUCCEEDED steps; the two words are not interchangeable, the units are not the
// same size, and a universal tally on the envelope would have had to pick one
// of the two vocabularies and make the other command lie in it. That is why the
// tally is a payload's business (D1) and why these two ints are declared here
// rather than reused from ManifestRun.
//
// # It says nothing about whether the run finished
//
// "The run reached the end of its plan" and "this many planned units were never
// reached" are true of any batch, so they are the envelope's (Run.Complete,
// Run.NotEvaluated) and are deliberately absent here — exactly as they are
// absent from ManifestRun, and for the same reason: a payload that restated
// them would be a second place for a finished run to be described as a partial
// one, and the two could disagree inside a single document.
type SnapshotRun struct {
	// Subvolumes is every subvolume the run set out to operate on, in
	// configuration order, whatever became of each.
	//
	// # The Go name is plural and the JSON key is singular, deliberately
	//
	// The key names the QUESTION R1.6 asks — "which subvolume did this run
	// operate on?" — and `.payload.subvolume` is how a consumer asks it. The
	// Go name names what the field HOLDS, which is a list, because a run
	// covers every subvolume its configuration names and a single string could
	// only be a join or a lie. snapshot.StageResult.Err with `json:"error"` is
	// the same deliberate split in this repository: the two names have
	// different readers, and each is written for its own.
	//
	// # It is the PLAN, and Steps below is what the run reached
	//
	// The two are not the same list and must not be derived from each other. A
	// run whose create failed for the first subvolume skips that subvolume's
	// remaining steps; a run cancelled between subvolumes never produces a step
	// for the rest. In both cases this field still names every subvolume the
	// run set out to cover, which is what lets a reader see WHICH one is
	// missing from the rows — the envelope's NotEvaluated says only how many.
	Subvolumes []string `json:"subvolume"`
	// Steps is every pipeline step that reached an outcome, in the order the
	// run established them.
	//
	// A nil slice reaches the JSON export as null and an empty one as [], and
	// the two say different things: the first is a producer that established
	// nothing, the second a run that ran no step at all. Neither is rewritten
	// into the other on the way out (R4.3).
	//
	// The ORDER is information rather than presentation — "prune failed after
	// create succeeded" and "create failed and prune never ran" are different
	// runs, and position is the only thing that distinguishes them in a flat
	// list — so nothing here or downstream sorts it.
	Steps []SnapshotStep `json:"steps"`
	// Ok is how many steps did what they were asked, and Failed is how many did
	// not (R1.6).
	//
	// They are FIELDS although the rows below could be walked to re-derive
	// them, for the reason ManifestRun.Ok gives at length: this document is
	// read by a program holding the exported file, and a program holding a file
	// cannot call a method. A count that exists only as a method is a count the
	// JSON export does not carry.
	//
	// Neither carries omitempty, and that is load-bearing: a zero dropped from
	// the document reads as "the producer never said", which conflates "no step
	// failed" with "nobody counted" — and a run in which everything succeeded
	// is the common case that would lose its zero.
	Ok     int `json:"ok"`
	Failed int `json:"failed"`
}

// SnapshotStep is what the run established about one step of its pipeline.
//
// # A step here is a STAGE, not a subprocess
//
// The two are different granularities and only one of them is legible. A stage
// is create, prune or ship: the thing the run set out to do, named by the
// manager that sequences them. A subprocess is `btrbk` or `snapper`, and one
// stage may invoke none of them, one, or several — the ssh ship moves no bytes
// itself, and a snapper subvolume takes several calls to create one snapshot.
// A report built from subprocesses would print "snapper, snapper, snapper" and
// leave the operator to guess which was the prune, which is why the step this
// type carries is the caller's semantic one.
type SnapshotStep struct {
	// Subvolume is the subvolume this step operated on.
	//
	// It may be empty, and empty is a fact rather than a gap: a step that
	// sweeps a remote holding objects from every subvolume is not attributable
	// to one, and the producer leaves it blank to say so. Nothing here invents
	// a placeholder for it — a made-up "all" would be this type answering a
	// question the run did not.
	Subvolume string `json:"subvolume"`
	// Step is what the run was doing: create, prune, ship or gfs, in the
	// producer's own words.
	Step string `json:"step"`
	// Target is the ship destination this step served, and empty for a step
	// that serves no destination.
	//
	// It is the destination's NAME — the one configured for it, or the driver
	// word when it was left unnamed — and never its address. A ship target's
	// address carries a host, a user and a remote path, and this value travels
	// into a file an operator may attach to a bug report.
	//
	// THE SCOPE OF THAT DECISION IS THIS FIELD AND THE THREE BESIDE IT. This
	// comment used to close by generalising it to "every other field here", so
	// that nothing in the type "may carry a credential or a host identifier" —
	// and Error, twelve lines below, is producer stderr taken verbatim, where a
	// failed btrbk ssh send prints `user@host:/path`. The generalisation was
	// therefore false about the one field most likely to carry an address, and
	// a false safety claim is worse than none: it is the sentence a reader
	// consults before deciding an export is safe to attach.
	//
	// Subvolume, Step and Target are each a name this package controls the shape
	// of, and each honours it. Error is not, and says so itself (S046-R2.3
	// applied to the type's own documentation). Narrowed at sub-task 15.7.
	Target string `json:"target"`
	// Success reports that the step did what it was asked.
	//
	// It is derived once, by the adapter, from the producer's own status word,
	// and the payload's counts are taken from THAT SAME expression — so the
	// word printed on a row and the number printed above it cannot describe
	// different runs.
	Success bool `json:"success"`
	// Error is why this step failed, in the producer's own words, and empty for
	// a step that did not fail.
	//
	// It carries the failing command's own stderr, joined on by the runner,
	// which is the only text that says what actually went wrong. It is taken
	// verbatim and never reworded: a rewritten diagnostic is one the operator
	// cannot search for.
	//
	// VERBATIM CUTS BOTH WAYS, and this field is the exception to Target's
	// no-address rule rather than a case of it. Whatever the failing command
	// wrote is what lands here — and a failed `btrbk ssh` send writes
	// `user@host:/path`, so a remote address CAN reach an export through this
	// field. Nothing scrubs it, deliberately: a diagnostic edited to be safe is
	// a diagnostic that no longer matches what the operator can search for or
	// reproduce.
	//
	// The consequence is stated rather than hidden, because the alternative is a
	// reader who trusts the type's own comment and attaches the export anyway:
	// an export that includes a FAILED ship step should be read before it is
	// shared. Redaction, if it is ever wanted, belongs at the producer that
	// knows which text is an address — not here, where it is an opaque string.
	//
	// It carries no subvolume and no step name: both sit beside it in this same
	// row, and a copy of either in here would print twice on every failure.
	Error string `json:"error"`
}

// Sections is the snapshot run as structure: what it says, in order, with every
// value at full length and not one decision about how it will look.
//
// # One block, because the run establishes one thing
//
// The check produces four sections because it answers four questions. A
// snapshot run answers one — for each step, did it do what it was asked — and a
// second heading would sit above content the first already carries. A report
// padded with empty blocks trains a reader to skim past the one that matters.
//
// # ShowAll decides what is LISTED and never what is counted
//
// A step that succeeded has nothing to report about, which is precisely the
// class of unit SectionOptions.ShowAll governs: listed when the flag asks,
// counted otherwise, with the omission stated in words (S046-R2.3, S044-R8.3). The count
// comes from the run's own Ok, never from the rows, so the number above the
// table cannot move when the listing does (R8.3).
//
// A FAILED step is always listed, whatever the flag says. It is the reason an
// operator is reading a snapshot report at all — a run that worked needs no
// reader — and a report that hid failures behind a flag would be answering a
// question nobody asked.
func (r SnapshotRun) Sections(opts SectionOptions) []Section {
	return []Section{snapshotSection(r, opts.ShowAll)}
}

// snapshotSection is the block: the subvolumes the run covered, the counts, the
// rows they were taken over, and the sentences saying what was left out and why.
func snapshotSection(r SnapshotRun, listEvery bool) Section {
	// R1.6 has two halves and this is the first of them. It is stated BEFORE
	// anything else and on every path out of this function, including the two
	// that return no rows at all, because "which subvolume did this run operate
	// on" must not be a question whose answer depends on --all: a run in which
	// every step succeeded lists no row by default, and the subvolume would
	// otherwise vanish from the report of the run that went perfectly.
	s := Section{Title: "Snapshot Run", Lead: []string{snapshotScope(r)}}

	// A run with no step still produces a report, and the report still says so
	// in a sentence rather than as an empty table under a heading — a report is
	// what an operator gets INSTEAD of a crash (R1.4), so reading one must not
	// be where the crash arrives.
	//
	// # The guard is on the COUNTS, not on the rows
	//
	// Ok and Failed are separate fields from Steps and nothing in the type makes
	// the three agree; the adapter sets all of them from one traversal, which is
	// why production never reaches the gap. But the sentence below is a CLAIM
	// ABOUT THE COUNTS, and asserting it from the length of a different field is
	// how a report starts lying about the one thing R1.6 asks it to state. A
	// payload carrying "2 succeeded, 1 failed" and no rows would have announced
	// that nothing was counted — false, and false in the direction that matters,
	// because the reader is told a run did nothing when it did.
	if r.Ok+r.Failed == 0 && len(r.Steps) == 0 {
		s.Lead = append(s.Lead, "No step has an outcome to report, so nothing is counted as succeeded or as failed.")
		return s
	}

	s.Lead = append(s.Lead, snapshotTally(r))

	// Counts without the rows they were taken over. The counts are still
	// stated, because they are what the run established; what is missing is the
	// per-step detail, and R2.3 says a report states what it omitted rather
	// than omitting it silently.
	if len(r.Steps) == 0 {
		s.Notes = []string{"No per-step row reached this report, so the counts above are stated without the list they were taken over."}
		return s
	}

	s.Rows.Headers = []string{"SUBVOLUME", "STEP", "STATE"}
	for _, step := range r.Steps {
		if step.Success && !listEvery {
			// Counted in the note below, not listed here (S044-R8.3).
			continue
		}
		s.Rows.Rows = append(s.Rows.Rows, Row{
			Cells:  []string{step.Subvolume, snapshotStepLabel(step), snapshotState(step)},
			Detail: snapshotDetail(step),
		})
	}

	s.Notes = snapshotNotes(r, listEvery)

	return s
}

// snapshotScope is R1.6's first half: which subvolume the run operated on.
//
// # It counts as well as naming, and the count is len()
//
// This is the one number in the report taken over a list rather than
// established by the run, and it is safe precisely because the list IS the
// fact: Subvolumes is the plan the run was given, so its length is how many
// subvolumes were planned, by definition. That is a different thing from Ok and
// Failed, which describe what HAPPENED and could not be recovered from any
// slice this payload holds.
//
// # A run with no subvolume says so rather than printing an empty list
//
// snapshot.Config warns at load time that an empty engine.subvolumes means
// nothing will be snapshotted, and it does not refuse to run. So this arm is
// reachable, and what it must not do is print a sentence that trails off into
// nothing — a reader meeting "The run operated on 0 subvolume(s): ." would be
// looking at a bug rather than at an answer.
func snapshotScope(r SnapshotRun) string {
	if len(r.Subvolumes) == 0 {
		return "The run was given no subvolume to operate on, so no step below is attributed to one."
	}
	return fmt.Sprintf("The run operated on %d subvolume(s): %s.",
		len(r.Subvolumes), strings.Join(r.Subvolumes, ", "))
}

// snapshotTally is R1.6's second half in one sentence: how the steps came out.
//
// The total is Ok+Failed rather than len(Steps) for the reason the guard above
// gives — the counts are what the run established, and a total taken over a
// different field could disagree with the two numbers it introduces.
func snapshotTally(r SnapshotRun) string {
	return fmt.Sprintf("%d step(s) ran: %d succeeded, %d failed.", r.Ok+r.Failed, r.Ok, r.Failed)
}

// snapshotStepLabel is the word the STEP column prints: the stage, and the ship
// destination it served when it served one.
//
// # The destination is folded into the step rather than given a column
//
// A fourth column would be empty on every create and every prune — the majority
// of rows in every run — and a column that is blank most of the time teaches a
// reader to stop looking at it, which is exactly when the one populated cell
// matters. Folding keeps both values on the row and costs nothing to a machine
// reader: the exported document carries Step and Target as separate keys, so
// nothing has to parse this string back apart.
func snapshotStepLabel(step SnapshotStep) string {
	if step.Target == "" {
		return step.Step
	}
	return fmt.Sprintf("%s (%s)", step.Step, step.Target)
}

// snapshotState is the word the STATE column prints. Only the WORDING is
// decided here; which of the two applies was decided by the run, and Success is
// the one field that says it.
func snapshotState(step SnapshotStep) string {
	if step.Success {
		return "succeeded"
	}
	return "failed"
}

// snapshotDetail is why a failed step failed: the producer's sentence, whole.
//
// # A successful step has no detail, and that is not an omission
//
// It returns the empty string, which every writer already omits rather than
// printing a bare indent. A step that did what it was asked produced no
// diagnostic to keep — the runner records an error only on failure — so there
// is nothing being held back here for a note to apologise for.
//
// # The sentence is FOLDED, never shortened
//
// Every non-whitespace character survives; only the line structure is collapsed,
// and it has to be. A Row.Detail is one line by definition — the plain writer
// prints it under its row and the Markdown writer makes it a table cell, and a
// raw newline corrupts the layout of both — while the verbatim string stays on
// SnapshotStep.Error, which is what the JSON export carries. A command's stderr
// arrives here with its own newlines in it, because the runner joins stderr onto
// the error, so this is not a hypothetical case.
func snapshotDetail(step SnapshotStep) string {
	if step.Success {
		return ""
	}
	return foldToOneLine(step.Error)
}

// snapshotNotes is what the table left out, and why (S046-R2.3, S044-R8.3).
func snapshotNotes(r SnapshotRun, listEvery bool) []string {
	var notes []string

	// Conditional on there having BEEN a success: a run in which every step
	// failed listed every one of its rows, so it has nothing to account for and
	// gets no sentence at all.
	if r.Ok > 0 {
		if listEvery {
			// The count is stated even when every row is present, for the
			// reason the check's own up-to-date note is: a count never depends
			// on which rows were listed (R8.3), so the sentence keeps its shape
			// in both directions and only its tail changes.
			notes = append(notes, fmt.Sprintf("%d step(s) succeeded and are listed above.", r.Ok))
		} else {
			notes = append(notes, fmt.Sprintf("%d step(s) succeeded and are not listed; pass --all to list them.", r.Ok))
		}
	}

	// The other omission, stated once for the run because it is one decision
	// rather than a property of any row (R2.3). The producer times every stage
	// and persists the timings with the run result, which is where `snapshot
	// status` reads them from; carrying them here would put a value in the
	// report that differs between two runs of the same shape, and this report
	// is read to find out WHAT happened.
	return append(notes,
		"How long each step took is not carried here: the run's own result file keeps the timings, and 'snapshot status' is what reads them back.")
}
