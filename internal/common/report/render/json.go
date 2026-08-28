package render

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/obentoo/bentoolkit/internal/common/report"
)

// jsonIndent is one level of indentation in the exported document.
//
// It is named because this package already has an indent, and the two must
// never be unified: that one counts DISPLAY CELLS a terminal renderer pads
// with, this one is bytes a machine reader ignores entirely. They agree on two
// only by coincidence, and a reader meeting the bare literal below would have no
// way to tell which of the two kinds it was.
const jsonIndent = "  "

// JSON writes run as one indented JSON document: the whole report, for a reader
// that is a program rather than a person (R4.1, R4.2).
//
// # The model IS the document
//
// There is no wire type here, and no struct literal copying field by field into
// one. report.Run is handed to encoding/json as it stands — envelope and
// payload together — which is what makes R4.2, "the document carries every fact
// the renderers read", true by construction rather than by a mapping somebody
// has to keep in sync (D3, and story 044's R9.4 before it).
//
// The payload half is the part that keeps proving this. Nothing in this file
// names a payload TYPE: report.Payload is an interface, and encoding/json
// reaches the concrete value's exported fields on its own. So a field added to
// any payload — the check's, the manifest run's, one written after this file
// was last read — reaches the export with no edit here. A mapping would reach
// it only when someone remembered, and the failure would be silent on both
// sides.
//
// The wire keys are therefore the model's json tags, and those tags are a
// contract: renaming one renames the key under a consumer who has already
// shipped a jq expression against it. The root's keys are declared in
// internal/common/report/run.go; the ones below "payload" are declared by
// whichever payload the run carries.
//
// # kind is what a consumer discriminates on; schema says which shape it is in
//
// Those two keys sit at the document ROOT, before anything domain-shaped
// (R4.1). kind names the command that produced the document, and it is the one
// key a consumer branches on: it is fixed per command and derived from nothing
// a run establishes, so `.kind == "autoupdate.check"` is a filter rather than a
// guess. It is also what makes every later addition free — a fourth command
// adds a value to report.Kind and changes no existing key.
//
// schema is 2, and not 1, because the shape this envelope replaces — the
// model's own fields at the root, with nothing wrapped around them — IS schema
// 1 in the field, even though nothing in it says so. Numbering the new shape 1
// would make the version indistinguishable from the version that has no
// version: a consumer meeting `{"schema": 1, ...}` could not tell which of the
// two documents it held.
//
// The move to the envelope is a BREAK, and it is announced as one: a consumer's
// `jq '.tally.proved'` becomes `jq '.payload.tally.proved'`. It is spent in the
// same release that migrates `overlay validate --json` onto this same document,
// so consumers face one announced migration rather than two at unrelated times
// (D3).
//
// # It writes; it does not read
//
// There is no decoder here and none beside report.Run, on purpose: decoding
// into a Payload needs a type switch driven by kind, which is a registry of
// every kind maintained by hand. Export is write-only today, so that registry
// would have no caller to keep it honest. The reader belongs with the first use
// case that needs one (D3).
//
// # It takes no Options, exactly as Markdown does not
//
// R3.4 says an export carries the complete report — every unit, every reason in
// full, no shortening, whatever the terminal was asked for. The absence of a
// parameter is how that is enforced: there is no Width here to shorten to and
// no ShowAll here to honour, so an export that mirrored screen truncation is
// not something to remember not to write. It cannot be written.
//
// # A nil slice reaches the wire as null, on purpose
//
// internal/autoupdate/validate has a Normalized() that turns nil slices into
// empty ones so jq never meets a null. It is deliberately not copied here. That
// is a transformation, and a transformation is the very thing this design
// removes: the round trip the payload is pinned on — marshal, unmarshal,
// compare whole — stops being lossless the moment the document says something
// the model did not. A consumer that trips over a null is looking at a producer
// that left a slice nil; that is where it gets fixed, not by a quiet rewrite on
// the way out.
//
// # The encoder's defaults are kept
//
// HTML escaping stays on, matching renderValidateJSON in cmd/bentoo — the CLI's
// other JSON surface. It costs readability on a reason containing > or &, and
// costs nothing in meaning, because the value round-trips identically. One tool
// emitting two dialects of JSON costs more.
//
// # The write is checked
//
// A report redirected to a full disk is an ordinary failure, and a renderer that
// swallowed it would report success for output nobody received.
func JSON(w io.Writer, run report.Run) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", jsonIndent)

	if err := enc.Encode(run); err != nil {
		return fmt.Errorf("writing the JSON report: %w", err)
	}
	return nil
}
