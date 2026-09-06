package report

// Authored for story 046, sub-task 1.3 — R1.3, R4.1, R7.4.
//
// Written from the contract: design.md D1 declares Run, Kind and Payload field
// by field with their json tags, and D3 fixes the document D1 marshals into —
// schema and kind at the root, the domain half nested under "payload", and
// schema = 2 because "the current shape is schema 1 in the field, even though
// nothing declares it".
//
// The one name assumed beyond D1 is SchemaVersion, the constant holding that 2.
// D3 states the number and not its spelling; a literal 2 typed at every
// producer is the same defect as a typed column width, so the number is asked
// for through a name.
//
// Red on arrival: none of the envelope exists yet — Run, Kind, Payload,
// SectionOptions and SchemaVersion are all undefined in this package today.

import (
	"encoding/json"
	"strings"
	"testing"
)

// stubPayload is a domain half with no domain: enough to be nested, marshalled
// and asked for its sections, and nothing more.
//
// It exists so the envelope can be tested WITHOUT a payload — the whole claim
// of D1 is that Run knows nothing about what produced it, and a test that
// reached for AutoupdateCheck to prove that would have disproved it instead.
type stubPayload struct {
	Fact string `json:"fact"`
}

func (p stubPayload) Sections(SectionOptions) []Section {
	return []Section{{Title: "Stub", Lead: []string{p.Fact}}}
}

// rootOf marshals r and returns the document's top-level object.
func rootOf(t *testing.T, r Run) map[string]any {
	t.Helper()

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshalling the run: %v", err)
	}

	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("the envelope did not produce a JSON object: %v\n%s", err, data)
	}
	return root
}

// TestRunNamesItsSchemaAndKindAtTheRoot pins R4.1 where a machine reader looks
// first. A consumer holding this document must be able to answer "what is this,
// and which shape of it" from the two outermost keys, without knowing which
// command wrote it.
func TestRunNamesItsSchemaAndKindAtTheRoot(t *testing.T) {
	root := rootOf(t, Run{
		Schema:   SchemaVersion,
		Kind:     KindAutoupdateCheck,
		Title:    "Autoupdate check",
		Complete: true,
		Payload:  stubPayload{Fact: "one"},
	})

	if got, ok := root["schema"].(float64); !ok || int(got) != 2 {
		t.Errorf(`root["schema"] = %v, want 2 — schema 1 is the unversioned shape already in the field (D3)`, root["schema"])
	}
	if got, _ := root["kind"].(string); got != "autoupdate.check" {
		t.Errorf(`root["kind"] = %q, want %q`, got, "autoupdate.check")
	}
	if SchemaVersion != 2 {
		t.Errorf("SchemaVersion = %d, want 2", SchemaVersion)
	}
}

// TestRunNestsThePayload pins the other half of D3's break: the domain facts
// move DOWN one level, so `.tally.proved` becomes `.payload.tally.proved` and
// every subsequent kind adds a value rather than a key.
func TestRunNestsThePayload(t *testing.T) {
	root := rootOf(t, Run{
		Schema:  SchemaVersion,
		Kind:    KindAutoupdateCheck,
		Payload: stubPayload{Fact: "nested"},
	})

	payload, ok := root["payload"].(map[string]any)
	if !ok {
		t.Fatalf(`root["payload"] = %#v, want an object holding the domain half`, root["payload"])
	}
	if got, _ := payload["fact"].(string); got != "nested" {
		t.Errorf(`payload["fact"] = %q, want %q — the payload marshals as itself, not as a summary of itself`, got, "nested")
	}
	if _, present := root["fact"]; present {
		t.Error(`the payload's own key reached the ROOT — a payload flattened into the envelope is schema 1 again, and the kind stops meaning anything about the shape below it`)
	}
}

// TestIncompleteRunCarriesItsGap pins R1.4 at the envelope. An interrupted run
// still produces a document, and the number of planned units it never reached
// is a value in it rather than a sentence somewhere in a section.
//
// Both fields are asserted on a COMPLETE run too, because the failure this
// guards against is omitempty: a `complete: false` that vanished from the
// document is read as "the producer did not say", which is precisely the
// conflation the tally exists to remove.
func TestIncompleteRunCarriesItsGap(t *testing.T) {
	for _, tc := range []struct {
		name         string
		complete     bool
		notEvaluated int
	}{
		{"interrupted with three unreached", false, 3},
		{"interrupted with nothing lost", false, 0},
		{"complete", true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := rootOf(t, Run{
				Schema:       SchemaVersion,
				Kind:         KindAutoupdateCheck,
				Complete:     tc.complete,
				NotEvaluated: tc.notEvaluated,
				Payload:      stubPayload{},
			})

			complete, present := root["complete"]
			if !present {
				t.Fatal(`the document has no "complete" key — an absent false is indistinguishable from an unanswered question`)
			}
			if complete != tc.complete {
				t.Errorf(`root["complete"] = %v, want %v`, complete, tc.complete)
			}

			gap, present := root["not_evaluated"]
			if !present {
				t.Fatal(`the document has no "not_evaluated" key`)
			}
			if got, _ := gap.(float64); int(got) != tc.notEvaluated {
				t.Errorf(`root["not_evaluated"] = %v, want %d`, gap, tc.notEvaluated)
			}
		})
	}
}

// TestTwoKindsNeverCollapse is the first hostile half of R4.4: the case that
// makes "distinguishable by its named kind alone" fire WRONGLY.
//
// Two runs identical in every other respect — same title, same completeness,
// byte-identical payload — must still be told apart. A kind derived from
// anything the two share (the title, the payload's shape, the section headings)
// answers the same string for both, and a consumer filtering `.kind` then
// receives one command's document believing it holds another's.
func TestTwoKindsNeverCollapse(t *testing.T) {
	same := stubPayload{Fact: "identical"}

	manifest := rootOf(t, Run{Schema: SchemaVersion, Kind: KindOverlayManifest, Title: "Run", Payload: same})
	snapshot := rootOf(t, Run{Schema: SchemaVersion, Kind: KindSnapshotRun, Title: "Run", Payload: same})

	if manifest["kind"] == snapshot["kind"] {
		t.Fatalf("two different kinds produced the same discriminator %q — everything else about these two documents is identical, so the kind is the only thing that could have told them apart (R4.4)", manifest["kind"])
	}
}

// TestOneKindNeverSplits is the converse hostile half, and it is the one a
// suite usually leaves out. A rule that keeps two kinds apart by mixing the
// run's contents into the discriminator satisfies the test above and breaks
// here: the same command's empty run and full run would carry different kinds,
// and `.kind == "overlay.manifest"` would match only some of the documents
// that command writes.
func TestOneKindNeverSplits(t *testing.T) {
	empty := rootOf(t, Run{Schema: SchemaVersion, Kind: KindOverlayManifest, Title: "", Complete: false, NotEvaluated: 9, Payload: stubPayload{}})
	full := rootOf(t, Run{Schema: SchemaVersion, Kind: KindOverlayManifest, Title: "42 targets", Complete: true, Payload: stubPayload{Fact: "lots"}})

	if empty["kind"] != full["kind"] {
		t.Fatalf("the same kind rendered two ways: %q and %q — a discriminator that varies with a run's contents cannot be filtered on (R4.4)", empty["kind"], full["kind"])
	}
	if empty["kind"] != "overlay.manifest" {
		t.Errorf(`kind = %q, want %q`, empty["kind"], "overlay.manifest")
	}
}

// TestNoThirdKindCollidesWithAnother asks the question the pair above cannot:
// the two kinds compared there are told apart, but a THIRD kind arriving later
// reaches the same rendered answer by a route the pair never exercised.
//
// Two ways it collides, and both are checked because a fix for one can create
// the other:
//
//  1. Equality. Two constants spelling the same string make the discriminator
//     ambiguous for every consumer, and the compiler says nothing.
//  2. Prefix. `.kind | startswith("overlay.manifest")` is the natural jq for
//     "any manifest run", and a later "overlay.manifest.dry" would answer true
//     to it. A kind that is a prefix of another kind is a collision waiting for
//     the story that adds the fourth command.
//
// A kind added without being added HERE is the failure this cannot catch, and
// that is stated rather than left implicit: the list below is the contract.
func TestNoThirdKindCollidesWithAnother(t *testing.T) {
	declared := map[string]Kind{
		"KindAutoupdateCheck": KindAutoupdateCheck,
		"KindOverlayManifest": KindOverlayManifest,
		"KindSnapshotRun":     KindSnapshotRun,
		// Added by sub-task 8.1, the change that started emitting it. It was
		// deliberately absent while it was only reserved — a kind nothing
		// produces is not part of the contract a consumer filters on — and
		// run.go said so, naming this list and this task. `overlay.validate`
		// is the case the PREFIX rule below is for: it shares a domain with
		// `overlay.manifest`, so the two are neighbours in exactly the way a
		// `.kind | startswith("overlay.")` consumer would notice.
		"KindOverlayValidate": KindOverlayValidate,
		// Added by story 047's sub-task 2.4, the change that declares it —
		// same condition the entry above was held to, applied one story later.
		//
		// It is the first kind whose own command has SUB-FLOWS in prospect,
		// which makes it the first real test of the PREFIX rule rather than a
		// hypothetical one: `overlay.compare.realign` would answer true to a
		// consumer matching startswith("overlay.compare"), so the sub-flows
		// share this one kind and the check below is what keeps that decision
		// from being quietly reversed. `overlay.manifest` and
		// `overlay.validate` beside it are siblings and not prefixes, which is
		// the case the rule must NOT fire on (S046-R4.4, S047-R1.1).
		"KindOverlayCompare": KindOverlayCompare,
	}

	for aName, a := range declared {
		if strings.TrimSpace(string(a)) == "" {
			t.Errorf("%s is empty — an unnamed kind is indistinguishable from a document that named none", aName)
		}
		for bName, b := range declared {
			if aName == bName {
				continue
			}
			if a == b {
				t.Errorf("%s and %s are both %q — two kinds spelling one string cannot be told apart (R4.4)", aName, bName, a)
			}
			if strings.HasPrefix(string(a), string(b)) {
				t.Errorf("%s (%q) begins with %s (%q) — a consumer matching a kind by prefix cannot separate them", aName, a, bName, b)
			}
		}
	}
}

// TestPayloadIsAskedForItsSections pins the half of D1 that keeps the renderers
// free of the domain: the envelope reaches its payload through Sections and
// through nothing else, so a fourth kind is a new payload rather than an edit
// to a renderer (R7.4).
func TestPayloadIsAskedForItsSections(t *testing.T) {
	var p Payload = stubPayload{Fact: "asked"}

	blocks := p.Sections(SectionOptions{})
	if len(blocks) != 1 {
		t.Fatalf("Sections() returned %d blocks, want 1", len(blocks))
	}
	if blocks[0].Title != "Stub" || len(blocks[0].Lead) != 1 || blocks[0].Lead[0] != "asked" {
		t.Errorf("Sections() = %+v, want the block the payload built", blocks[0])
	}
}
