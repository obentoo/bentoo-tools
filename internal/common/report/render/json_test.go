package render

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/obentoo/bentoolkit/internal/common/report"
)

// exportedPayload decodes the payload half of a document back into the check
// report.
//
// It is a decoder in a TEST and deliberately not one in the package: report.Run
// marshals but does not unmarshal, because decoding into a Payload interface
// needs a type switch driven by kind — a hand-maintained registry of every kind
// (D3). A test does not need that registry, because it knows which kind it just
// exported and can name the concrete type outright.
func exportedPayload(t *testing.T, doc []byte) report.AutoupdateCheck {
	t.Helper()

	// The key is spelled out here, where jsonKeyFor reads it from the tag
	// everywhere else in this file. A struct tag is a compile-time literal and
	// cannot be computed, so this one has to be typed — and the difference is
	// worth having rather than working around. jsonKeyFor FOLLOWS a tag rename
	// silently, which is what a test about the four tally counts should do; the
	// literal below does not, so renaming report.Run.Payload's tag turns this
	// round trip red. That is the honest alarm: "payload" is the string a
	// consumer has already typed into a jq expression.
	var envelope struct {
		Payload report.AutoupdateCheck `json:"payload"`
	}
	if err := json.Unmarshal(doc, &envelope); err != nil {
		t.Fatalf("the document JSON produced is not valid JSON: %v\n%s", err, doc)
	}
	return envelope.Payload
}

// TestJSONRoundTrip pins S044-R9.4: a machine reader sees the fields the renderers
// saw. Round-tripping the fixture and comparing it whole is the only assertion
// that stays true as the model grows — a field-by-field list would be checked
// against the fields somebody remembered to add to it.
func TestJSONRoundTrip(t *testing.T) {
	want := fixtureReport()

	var buf bytes.Buffer
	if err := JSON(&buf, finishedRun(want)); err != nil {
		t.Fatalf("JSON returned an error: %v", err)
	}

	got := exportedPayload(t, buf.Bytes())

	if !reflect.DeepEqual(got, want) {
		t.Errorf("the report did not survive the round trip\n--- want ---\n%+v\n--- got ---\n%+v", want, got)
	}
}

// TestJSONRoundTripKeepsTheReasonWhole is S044-R9.3 for the JSON path. The screen
// shows 96 cells; the record holds all 232.
func TestJSONRoundTripKeepsTheReasonWhole(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, finishedRun(fixtureReport())); err != nil {
		t.Fatalf("JSON returned an error: %v", err)
	}

	got := exportedPayload(t, buf.Bytes())

	for _, entry := range got.Plan {
		if entry.Package == "sys-apps/portage" && entry.Reason != planReason {
			t.Errorf("the exported reason is %d characters, want %d — the export shortened it", len(entry.Reason), len(planReason))
		}
	}
}

// TestJSONNamesTheFourTallyCounts pins R5.1 at the machine surface. A consumer
// that cannot find "inconclusive" cannot tell a toolkit limitation from the
// operator's policy, which is the whole distinction this story adds.
func TestJSONNamesTheFourTallyCounts(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, finishedRun(fixtureReport())); err != nil {
		t.Fatalf("JSON returned an error: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// The tally is a payload's own count, so it is reached through the
	// envelope's payload key rather than at the root (D3). The key is read from
	// the tag, like every other key here, rather than assumed.
	payload, ok := doc[jsonKeyFor(t, report.Run{}, "Payload")].(map[string]any)
	if !ok {
		t.Fatalf("the document has no payload object\n%s", buf.String())
	}

	tally, ok := payload[jsonKeyFor(t, report.AutoupdateCheck{}, "Tally")].(map[string]any)
	if !ok {
		t.Fatalf("the document has no tally object\n%s", buf.String())
	}

	for _, field := range []string{"Proved", "Errored", "Inconclusive", "Skipped"} {
		key := jsonKeyFor(t, report.Tally{}, field)
		if _, present := tally[key]; !present {
			t.Errorf("the tally object has no %q key", key)
		}
	}
}

// TestJSONDropsNoField is the assertion a round trip cannot make. A round trip
// survives `omitempty` intact — the zero value goes out as an absent key and
// comes back as the zero value, and nothing notices. But a consumer reading
// that document cannot tell "false" from "the producer did not say", which is
// the same conflation this story exists to remove from the tally.
//
// So: every exported field of every model type must appear as a key, even when
// its value is the zero one.
func TestJSONDropsNoField(t *testing.T) {
	// A report whose every field is the zero value — the case omitempty eats.
	var buf bytes.Buffer
	empty := report.AutoupdateCheck{
		Scanned: []report.PackageResult{{}},
		Plan:    []report.PlanEntry{{}},
		Results: []report.ValidationRow{{}},
	}
	if err := JSON(&buf, finishedRun(empty)); err != nil {
		t.Fatalf("JSON returned an error: %v", err)
	}
	doc := buf.String()

	types := []struct {
		name string
		v    any
	}{
		{"AutoupdateCheck", report.AutoupdateCheck{}},
		{"PackageResult", report.PackageResult{}},
		{"PlanEntry", report.PlanEntry{}},
		{"ValidationRow", report.ValidationRow{}},
		{"Tally", report.Tally{}},
	}

	for _, typ := range types {
		rt := reflect.TypeOf(typ.v)
		for i := range rt.NumField() {
			field := rt.Field(i)
			if !field.IsExported() {
				continue
			}
			key := jsonKeyFor(t, typ.v, field.Name)
			if !strings.Contains(doc, `"`+key+`"`) {
				t.Errorf("%s.%s is absent from the document as key %q — omitempty makes a zero value indistinguishable from an unanswered one",
					typ.name, field.Name, key)
			}
		}
	}
}

// jsonKeyFor answers what key a field serializes under, reading the struct tag
// rather than assuming a convention. If the model carries no tags, the Go name
// IS the wire contract, and this returns it.
func jsonKeyFor(t *testing.T, v any, fieldName string) string {
	t.Helper()

	field, ok := reflect.TypeOf(v).FieldByName(fieldName)
	if !ok {
		t.Fatalf("%T has no field %s — the fixture and the model have diverged", v, fieldName)
	}

	tag := field.Tag.Get("json")
	if tag == "" {
		return fieldName
	}
	if name, _, _ := strings.Cut(tag, ","); name != "" && name != "-" {
		return name
	}
	return fieldName
}
