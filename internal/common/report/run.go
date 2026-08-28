package report

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
	// KindOverlayValidate is RESERVED for `overlay validate --json`, which
	// task 8 of story 046 migrates off its own JSON schema and onto this
	// envelope (R4.3). It is declared now, ahead of its one call site, so
	// that task sets a value where the document is built instead of
	// reopening this file.
	//
	// Its string was checked against the two rules above by hand — it equals
	// no other kind and is a prefix of none, `overlay.manifest` included. That
	// check is NOT yet mechanical: run_test.go's collision sweep works from a
	// list it declares itself, and this constant is deliberately absent from
	// it, because a reserved kind nothing produces is not yet part of the
	// contract a consumer filters on. Task 8 adds it to that list in the same
	// change that starts emitting it.
	KindOverlayValidate Kind = "overlay.validate"
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
// reached" are answerable for any batch, and story 044's R1.4 already depends
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
