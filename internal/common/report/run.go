package report

import "fmt"

// SchemaVersion is the version number every exported document carries at its
// root (R4.1). It is what a consumer reads before it reads anything else.
//
// # Why it starts at 2
//
// This is the first release that declares a version at all, and the number is
// still 2. That is not an off-by-one. The shape shipped today — the model's own
// fields at the document root, with no envelope around them — IS schema 1 in
// the field, even though nothing in it says so. Numbering the new shape 1 would
// make the version indistinguishable from the version that has no version: a
// consumer meeting `{"schema": 1, ...}` could not tell whether it held the new
// document or an old one that had grown a key by coincidence. Starting at 2
// leaves 1 meaning exactly what it already means (D3).
//
// # Why it is a name and not a literal
//
// Every producer sets Run.Schema, and a 2 typed at each of them is the same
// defect as a column width typed into a format string: three places to find on
// the day it becomes 3, and nothing that fails when one is missed.
const SchemaVersion = 2

// Kind names what produced a report. It is the JSON discriminator, and the only
// reason a consumer can tell two exports apart without being told (R4.1, R4.4).
//
// # A kind names the PRODUCER, never the contents
//
// The value is fixed per command and derived from nothing a run establishes —
// not its title, not how many rows it found, not whether it finished. That is
// what makes `.kind == "overlay.manifest"` a filter rather than a guess: a
// discriminator that varied with a run's contents would answer one string for
// an empty run and another for a full one, so a consumer filtering on it would
// receive some of a command's documents and silently miss the rest. The
// converse failure is just as quiet — two commands answering one string hand a
// consumer the wrong document under the right name.
//
// # The values are dotted, and no kind is a prefix of another
//
// `domain.action` reads as a path and groups usefully when sorted. The
// constraint that travels with it is that no kind may begin with another kind:
// `.kind | startswith("overlay.manifest")` is the natural jq for "any manifest
// run", and a later `overlay.manifest.dry` would answer true to it. Both the
// equality and the prefix rule are checked mechanically in run_test.go, because
// neither is something the compiler can say anything about.
type Kind string

const (
	// KindAutoupdateCheck is `overlay autoupdate check`: the run that scans an
	// overlay's packages, plans the validation the pending updates earn, and
	// reports what the gates answered. Its payload carries the four-column
	// tally.
	KindAutoupdateCheck Kind = "autoupdate.check"
	// KindOverlayManifest is `overlay manifest`: the run that regenerates
	// Manifest files. Its payload counts what succeeded and what failed —
	// two columns, not four, which is why the tally is a payload's business
	// and not the envelope's.
	KindOverlayManifest Kind = "overlay.manifest"
	// KindSnapshotRun is `snapshot run`: the run that takes and prunes
	// subvolume snapshots. Its units are subvolumes rather than packages,
	// which is the distance that proves the envelope carries no domain.
	KindSnapshotRun Kind = "snapshot.run"
	// KindOverlayValidate is `overlay validate`: the run that asks whether
	// each ebuild still matches the source it points at, gate by gate. It was
	// RESERVED here by sub-task 1.3 and is emitted by sub-task 8.1, which
	// migrated `--json` off the schema of its own and onto this envelope
	// (R4.3, D8) — the flag now names the same document `--export` writes to a
	// `.json` path, at stdout.
	//
	// Its payload is the only one this package does not declare, and that is
	// the dependency direction rather than an omission: the facts a validation
	// run establishes live in internal/autoupdate/validate, which
	// boundary_test.go forbids this package from importing, so the type that
	// puts them in the payload position sits at the adapter
	// (cmd/bentoo/overlay_validate_report.go), and it stays there. Only the
	// JSON export carries that payload's fields, because that renderer reaches
	// them through encoding/json rather than through Sections.
	//
	// # Corrected by story 047's sub-task 2.4
	//
	// The sentence above used to read "Story 047 gives it a model of its own
	// here, with sections; until then only the JSON export carries it". That
	// was written during story 046, when "047" was the name of whatever came
	// next, and it is not what 047 turned out to be: 047 declares
	// KindOverlayCompare below and gives this kind nothing. The dependency
	// direction stated one paragraph up is unchanged, and it is still the
	// reason this payload sits at the adapter.
	//
	// It is corrected rather than deleted because the prediction was
	// load-bearing: render/contract_test.go's payloadTypes was told, on the
	// strength of it, that the entry to add for this kind would be
	// report.ValidateRun. A prediction left standing after the story it names
	// has landed reads as a plan rather than as a miss, and the next person to
	// widen that list would have widened it with the wrong name.
	//
	// It is on run_test.go's collision sweep as of 8.1. That was the condition
	// stated when it was reserved: a kind nothing produces is not yet part of
	// the contract a consumer filters on, and the change that starts emitting
	// it is the change that owes the mechanical check.
	KindOverlayValidate Kind = "overlay.validate"
	// KindOverlayCompare is `overlay compare`: the run that holds every
	// package the overlay carries against the copy a chosen repository ships,
	// and reaches a recommendation about each one. Its payload carries the
	// repository that was compared against, how much was scanned and how much
	// the two trees have in common, the packages themselves, and the
	// four-column verdict tally (S047-R1.1).
	//
	// # Its payload is the only one whose lists PARTITION what the run found
	//
	// The autoupdate check carries several slices too, and they are three
	// STAGES over one set of packages — what was scanned, what was planned,
	// what was validated — so one package appears in all three. The four lists
	// here are four DISJOINT sets, one per recommendation: remove ours,
	// re-apply ours, keep ours, and no opinion at all. They are four fields
	// rather than one list with a verdict column because the advice differs in
	// kind and not in degree — folding the rebase list into the redundant one
	// would recommend deleting an ebuild that carries changes the compared
	// repository has no copy of, which is the most expensive mistake this
	// report exists to prevent.
	//
	// # Declaring it cost no renderer edit, which is the claim it exists to test
	//
	// A kind is a discriminator and a payload is an implementation of
	// Sections, and those two are the whole of what render consumes. The only
	// files under internal/common/report/render that the sub-task declaring
	// this constant touched are that package's own guards, which had to learn
	// that a fifth kind exists; no renderer changed (S047-R1.2).
	//
	// # The value is FLAT, and stays flat as the command grows sub-flows
	//
	// A later `overlay.compare.realign` would answer true to a consumer
	// matching `.kind | startswith("overlay.compare")`, which is exactly the
	// collision the prefix rule above forbids — so every sub-flow shares this
	// one kind and says which flow it was in through the payload, never
	// through the discriminator. `overlay.manifest` and `overlay.validate`
	// beside it are SIBLINGS rather than prefixes, and sharing a domain that
	// way is what the dotted form is for (S046-R4.4).
	KindOverlayCompare Kind = "overlay.compare"
)

// Run is what every report carries, whatever produced it: the envelope.
//
// It is assembled in full before any part of it is rendered (R1.3), so a run
// that is interrupted, or that fails to write its output anywhere, still holds
// a complete description of what it established.
//
// # The envelope knows nothing about the domain
//
// Every field here is true of ANY batch — a set of packages, a set of
// subvolumes, a set of ebuilds. The facts that are true of only one kind of run
// sit behind Payload, reachable through Sections and through nothing else. That
// is what makes a fourth kind of run a new payload rather than an edit to a
// renderer (R7.4): a renderer able to name AutoupdateCheck would need a second
// branch for ManifestRun and a third for whatever comes after it.
//
// # Why Complete and NotEvaluated are here and the tally is not
//
// The asymmetry is deliberate, and it is the line between the two halves. "The
// run reached the end of its plan" and "this many planned units were never
// reached" are answerable for any batch, and S044-R4.3 already depends
// on both being answerable for an interrupted check. A tally is not: autoupdate
// counts four validation outcomes, manifest counts ok and failed, and one
// universal tally would either lose the four or invent columns manifest has no
// answer for. Each payload keeps its own count, spelled in its own vocabulary.
//
// # It marshals; it does not unmarshal
//
// There is no UnmarshalJSON here and no decoder anywhere else, on purpose.
// Payload is an interface, and decoding into an interface needs a type switch
// driven by kind — a registry of every kind, maintained by hand, wrong the
// first time someone adds a kind and forgets it.
//
// Nothing in this codebase reads one of these documents back: export is
// write-only today. A decoder written now would therefore be a registry with no
// caller, and its first real caller would be the one to discover it decodes the
// wrong thing. The reader belongs with the first use case that needs one, and
// it is that use case which should say what it needs — a round trip, or a
// single field, or only the kind — not this file, guessing in advance (D3).
type Run struct {
	// Schema is the document's version, always SchemaVersion. It is written
	// even though every document this release produces carries the same
	// number, because a version a consumer has to infer from the keys present
	// is the situation schema 1 is already in.
	Schema int `json:"schema"`
	// Kind names the command that produced this document. It is what a
	// consumer branches on, and the only key that says what the payload
	// below is.
	Kind Kind `json:"kind"`
	// Title is the run's heading in the reader's own words — "Autoupdate
	// check", "42 targets". It is a label, not a discriminator: two runs may
	// share one, and a consumer that matched on it would be matching on prose.
	Title string `json:"title"`
	// Complete reports that the run reached the end of its plan. A run stopped
	// early — by an interrupt, or by a failure it could not continue past —
	// sets this false, and the report still holds everything established up to
	// that point.
	//
	// It has no omitempty, and that is load-bearing: a `false` dropped from the
	// document reads as "the producer never said", which is exactly the
	// conflation this field exists to remove.
	Complete bool `json:"complete"`
	// NotEvaluated is how many planned units the run never reached. It is zero
	// for a complete run, and for an interrupted one it is the number that
	// turns a short list into a stated gap rather than a silent one (R1.4).
	//
	// It carries no omitempty for the same reason Complete does not.
	NotEvaluated int `json:"not_evaluated"`
	// Payload is the domain half, and it marshals NESTED under its own key.
	// The nesting is the whole point of the envelope: `.tally.proved` becomes
	// `.payload.tally.proved`, and every kind added after this one adds a
	// value to Kind rather than a key at the root (D3). A payload flattened up
	// here would be schema 1 again, with Kind saying nothing about the shape
	// beside it.
	Payload Payload `json:"payload"`
}

// Sections is the whole run as ordered blocks: the gap it left, if it left one,
// and then everything its payload has to say.
//
// # The envelope states the gap because only the envelope holds it
//
// Complete and NotEvaluated are fields of Run, so a payload cannot state that
// the run stopped early even though every section it produces is short by what
// the run never reached. Something above the payload therefore has to say it,
// and this is that something (R1.4). Saying it here rather than in each payload
// is also what makes one sentence serve packages, subvolumes and ebuilds alike
// — a payload that wrote its own would be three wordings to keep in agreement.
//
// # It goes FIRST, and it is prepended to a NEW slice
//
// Every block below the label is short by the units the run never reached, so a
// reader who meets the qualifier at the bottom has already drawn a conclusion
// from tables that were missing rows. The fullscreen viewport also cuts from the
// bottom, which makes the top the one place the label survives a terminal too
// short for the report.
//
// The new slice matters as much: append into the payload's own would let a
// report be edited by having been displayed, and the same blocks go on to the
// export a few lines later.
//
// # A run with no payload says what it can rather than panicking
//
// Payload is an interface, so a Run assembled by a producer that returned early
// can hold nil. That is a defect in the producer, and it is the producer's own
// tests that should catch it — but a report is the thing an operator gets INSTEAD
// of a crash (R1.4), so rendering one must not be the moment the crash arrives.
// A nil payload contributes no block, and an incomplete run still states its gap.
func (r Run) Sections(opts SectionOptions) []Section {
	var blocks []Section
	if r.Payload != nil {
		blocks = r.Payload.Sections(opts)
	}

	if r.Complete {
		return blocks
	}
	return append([]Section{interruptedSection(r.NotEvaluated)}, blocks...)
}

// interruptedSection is the label R1.4 asks for: a run that stopped before the
// end of its plan says so, and says how much of that plan it never reached.
//
// # It reads one number, and nothing about the terminal
//
// Nothing here asks which mode was on screen, and nothing here could answer —
// that absence is the requirement rather than an omission, because the same
// interrupt has to read the same way in a terminal, in a pull request comment
// and in a log file. The JSON export needs no part of it: it serializes Complete
// and NotEvaluated already, which is the same fact in the form a machine reader
// can act on.
//
// # It says "unit", because the envelope does not know what was counted
//
// The sentence this replaces said "package(s)", and it could: it was built by
// the autoupdate check, out of the check's own plan. This one is built for any
// batch, and naming packages here would be the envelope claiming a domain it
// deliberately has none of. What each unit was is stated by the sections below,
// in the payload's own vocabulary.
//
// # The count is stated even when it is zero
//
// A run interrupted once its last unit had already been evaluated lost nothing,
// and "0" is what says so. Suppressing the number there would leave that case
// indistinguishable from a report that is silent about how much is missing.
func interruptedSection(notEvaluated int) Section {
	return Section{
		Title: "Run Interrupted",
		Lead: []string{
			fmt.Sprintf("This report is incomplete: the run was interrupted, and %d planned unit(s) were not evaluated.",
				notEvaluated),
		},
		Notes: []string{
			"The counts below cover only what the run reached; the rest are counted in no column.",
		},
	}
}

// Payload is the domain half: the facts one kind of run establishes, and how
// that kind says them as ordered blocks.
//
// One method, and it is the entire contract between a producer and every
// renderer. A payload turns itself into []Section — titled blocks of structure,
// with no width, no colour and no escape sequence in them — and each renderer
// consumes sections and nothing else. That is the seam R7.4 names: a new kind
// of run implements this interface, and no renderer changes.
//
// Sections is the only way anything in this package reaches a payload. The
// payload does also MARSHAL — encoding/json reaches its exported fields
// directly, which is what puts the domain facts under "payload" in the exported
// document — but no code here and none in render may branch on its concrete
// type. A type switch over payloads is the edit-per-kind this interface exists
// to prevent.
type Payload interface {
	// Sections returns the payload's blocks in the order they are meant to be
	// read. opts says what the report should SAY; nothing in it says how the
	// report should look.
	Sections(opts SectionOptions) []Section
}

// SectionOptions is what a report should SAY. render.Options is what the device
// ALLOWS. Keeping those two apart is the split this story exists to hold.
//
// Listing every package a run found up to date is a decision about CONTENT, and
// it is made once, here, where the sections are built. Deciding that a line may
// be 100 cells wide is a decision about the DEVICE, and it is made in a
// renderer, from that device's own measurement. A renderer that read ShowAll
// would be deciding content, and the same run would then say different things
// in plain and in fullscreen with nothing in the model able to explain why.
//
// It lives beside Payload rather than beside Section because it is what a
// payload is ASKED, not part of what a payload answers.
type SectionOptions struct {
	// ShowAll lists the units a run found nothing to report about — the
	// packages already up to date — instead of only counting them. It is the
	// --all flag.
	//
	// False prints the count alone, and that is the default because those
	// units are the bulk of a large run: listing them unasked is most of why a
	// check that found four updates printed 348 lines.
	ShowAll bool
	// SkipPlan omits the section describing what the run intended to do. It
	// exists for exactly one caller: the on-screen report of a run whose plan
	// was already printed before the confirmation prompt, so the operator is
	// not shown the same list twice.
	//
	// An export must never set it. A record missing the plan answers no
	// question later (R2.4), and the reader of the file is not the operator
	// who saw the plan go by on screen.
	SkipPlan bool
}
