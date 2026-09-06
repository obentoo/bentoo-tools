package render

// Authored for story 046, sub-task 2.4 — R4.1, R4.2.
//
// Written from the contract: design.md D3 prints the document this test asserts
// on, before and after —
//
//	{"schema": 2, "kind": "autoupdate.check", "complete": true,
//	 "payload": {"scanned": [...], "plan": [...], "tally": {...}}}
//
// — and the Components block fixes the signature as JSON(w, Run). D3 also fixes
// the two properties that are easiest to lose on the way: the envelope
// marshals but does not unmarshal, and a nil slice reaches the wire as null
// rather than being normalized into an empty one.
//
// Red on arrival: JSON takes report.Report today, and report.Run does not exist.
//
// # Why the payload here is a stub and not AutoupdateCheck
//
// Task 2.4 lands before task 3.1, so AutoupdateCheck does not exist yet — and
// that ordering is the requirement rather than an inconvenience. A JSON test
// that reached for a payload type would be a renderer that knew one, which is
// exactly what R7.4 forbids.

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/obentoo/bentoolkit/internal/common/report"
)

// envelopePayload is a domain half with one of each shape that has ever been
// got wrong: a scalar, a nested object, a populated slice and a nil one.
type envelopePayload struct {
	Subvolume string          `json:"subvolume"`
	Counts    envelopeCounts  `json:"counts"`
	Steps     []string        `json:"steps"`
	Skipped   []string        `json:"skipped"`
	Extra     map[string]bool `json:"extra"`
}

type envelopeCounts struct {
	Ok     int `json:"ok"`
	Failed int `json:"failed"`
}

func (p envelopePayload) Sections(report.SectionOptions) []report.Section {
	return []report.Section{{Title: "Steps", Lead: []string{p.Subvolume}}}
}

func envelopeRun() report.Run {
	return report.Run{
		Schema:       report.SchemaVersion,
		Kind:         report.KindSnapshotRun,
		Title:        "snapshot run",
		Complete:     false,
		NotEvaluated: 2,
		Payload: envelopePayload{
			Subvolume: "/home",
			Counts:    envelopeCounts{Ok: 3, Failed: 1},
			Steps:     []string{"create", "prune"},
			Skipped:   nil,
			Extra:     nil,
		},
	}
}

func exportedDocument(t *testing.T, run report.Run) (raw string, root map[string]any) {
	t.Helper()

	var buf bytes.Buffer
	if err := JSON(&buf, run); err != nil {
		t.Fatalf("JSON returned an error: %v", err)
	}

	raw = buf.String()
	if err := json.Unmarshal([]byte(raw), &root); err != nil {
		t.Fatalf("the exported document is not valid JSON: %v\n%s", err, raw)
	}
	return raw, root
}

// TestJSONEnvelopeNamesSchemaAndKindAtTheRoot pins R4.1 at the surface a
// consumer actually reads. Two keys, outermost, and neither of them derived
// from the command's name — which is the whole reason the discriminator exists.
func TestJSONEnvelopeNamesSchemaAndKindAtTheRoot(t *testing.T) {
	raw, root := exportedDocument(t, envelopeRun())

	if got, ok := root["schema"].(float64); !ok || int(got) != 2 {
		t.Errorf(`root["schema"] = %v, want 2 (D3)`, root["schema"])
	}
	if got, _ := root["kind"].(string); got != string(report.KindSnapshotRun) {
		t.Errorf(`root["kind"] = %q, want %q`, got, report.KindSnapshotRun)
	}
	if !strings.Contains(raw, `"schema"`) || !strings.Contains(raw, `"kind"`) {
		t.Errorf("the document does not carry both keys literally:\n%s", raw)
	}
}

// TestJSONEnvelopeNestsThePayload pins the schema break itself: the domain half
// moves one level down, so `.tally.proved` becomes `.payload.tally.proved` and
// every kind added afterwards costs a consumer nothing.
func TestJSONEnvelopeNestsThePayload(t *testing.T) {
	_, root := exportedDocument(t, envelopeRun())

	payload, ok := root["payload"].(map[string]any)
	if !ok {
		t.Fatalf(`root["payload"] = %#v, want the domain half as an object`, root["payload"])
	}
	if got, _ := payload["subvolume"].(string); got != "/home" {
		t.Errorf(`payload["subvolume"] = %q, want "/home"`, got)
	}
	if _, leaked := root["subvolume"]; leaked {
		t.Error(`a payload key reached the ROOT — a flattened payload is schema 1 again, and the kind stops describing the shape below it`)
	}
}

// TestJSONEnvelopeDropsNoField is the assertion a round trip cannot make. A
// round trip survives omitempty intact: the zero value goes out as an absent
// key and comes back as the zero value, and nothing notices. A consumer reading
// that document cannot tell "false" from "the producer did not say" (R4.2).
//
// Every exported field of Run, and every exported field of the payload, must
// appear as a key even when its value is the zero one.
func TestJSONEnvelopeDropsNoField(t *testing.T) {
	empty := report.Run{Payload: envelopePayload{}}
	raw, root := exportedDocument(t, empty)

	for _, typ := range []struct {
		name string
		v    any
		in   map[string]any
	}{
		{"Run", report.Run{}, root},
		{"payload", envelopePayload{}, nil},
	} {
		rt := reflect.TypeOf(typ.v)
		for i := range rt.NumField() {
			field := rt.Field(i)
			if !field.IsExported() {
				continue
			}
			key, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			if key == "" {
				key = field.Name
			}
			if key == "-" {
				continue
			}
			if !strings.Contains(raw, `"`+key+`"`) {
				t.Errorf("%s.%s is absent from the document as key %q — omitempty makes a zero value indistinguishable from an unanswered one (R4.2)",
					typ.name, field.Name, key)
			}
		}
	}
}

// TestJSONEnvelopeKeepsANilSliceNull pins the decision D3 states outright: the
// export does not normalize. A nil slice is a producer that established
// nothing; an empty one is a producer that established an empty list, and a
// document that turns the first into the second has said something the model
// did not.
func TestJSONEnvelopeKeepsANilSliceNull(t *testing.T) {
	raw, root := exportedDocument(t, envelopeRun())

	payload, ok := root["payload"].(map[string]any)
	if !ok {
		t.Fatalf("no payload object in:\n%s", raw)
	}

	skipped, present := payload["skipped"]
	if !present {
		t.Fatalf(`payload has no "skipped" key:\n%s`, raw)
	}
	if skipped != nil {
		t.Errorf(`payload["skipped"] = %#v, want null — the export carries the model as it stands, transformations included (D3)`, skipped)
	}

	if steps, _ := payload["steps"].([]any); len(steps) != 2 {
		t.Errorf(`payload["steps"] = %#v, want the two values the producer established`, payload["steps"])
	}
}

// TestJSONEnvelopeStatesAnIncompleteRun pins R1.4 on the machine surface: the
// gap an interrupted run left is a value a script can act on, not a sentence in
// a section it would have to parse.
func TestJSONEnvelopeStatesAnIncompleteRun(t *testing.T) {
	_, root := exportedDocument(t, envelopeRun())

	if complete, present := root["complete"]; !present || complete != false {
		t.Errorf(`root["complete"] = %v (present=%v), want false`, root["complete"], present)
	}
	if gap, _ := root["not_evaluated"].(float64); int(gap) != 2 {
		t.Errorf(`root["not_evaluated"] = %v, want 2`, root["not_evaluated"])
	}
}
