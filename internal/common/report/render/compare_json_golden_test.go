package render

// Story 047, sub-task 5.2 — S047-R1.4: the fifth kind joins the envelope
// additively, leaving the schema at 2 and changing no existing key. This file
// is the instrument that makes that claim reviewable instead of asserted.
//
// # Why a golden, when json.go needed no edit at all
//
// Not one line of json.go was touched to export this payload, and that is the
// trap rather than the good news. The encoder is handed report.Run as it
// stands and reaches the payload's exported fields on its own — no registry, no
// mapping, nothing to keep in sync. So every exported field of
// report.CompareRun and every json tag on it became a public wire contract the
// moment the adapter shipped, with nothing forcing a human to look at what had
// been published. A field renamed, a shape changed, a list that starts arriving
// null: all three reach a consumer in silence. This golden turns each of them
// into a diff somebody has to approve.
//
// Regenerate with:
//
//	go test ./internal/common/report/render/ -run TestCompareJSONGolden -update
//
// and READ the diff. A golden accepted unread is a screenshot of a bug.
//
// # How this differs from compare_run_json_test.go — delete neither
//
// internal/common/report/compare_run_json_test.go holds compareRunKeys: a
// hand-listed set asserting WHICH keys CompareRun publishes, taken at the ZERO
// value, because the zero value is the only place an omitempty is visible. It
// catches a key appearing or disappearing.
//
// This golden is taken at a POPULATED value, so it pins what those keys are
// WORTH: the objects nested under keep_groups, the member lists inside them,
// the order the encoder emits, the actual numbers. A rename trips both. A
// change in MEANING — a count that starts double-counting, a pair of packages
// that stops collapsing into a group, a populated list that arrives null —
// trips only this one, because a key set cannot see values. Neither test
// subsumes the other.
//
// # It shares the text goldens' subject on purpose
//
// The fixture is comparePopulatedRun, declared in compare_golden_test.go and
// rendered by every text golden beside it. A second fixture written here would
// give the JSON and the text views different subjects, and a field could then
// change in one without the other's golden moving — which is exactly the
// blindness both files exist to remove.

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestCompareJSONGolden pins the whole exported document for the comparison
// report: the envelope, and every exported field of the payload beneath it.
//
// The two assertions beside the golden are the ones a diff alone states
// weakly. `schema` and `kind` would move inside the golden like any other
// bytes, and a reviewer regenerating in a hurry could approve their movement
// without noticing it was the compatibility promise that moved (S047-R1.4).
// Named here, they fail with a sentence rather than a diff hunk.
//
// Both are compared against LITERALS, not against report.SchemaVersion and
// report.KindOverlayCompare. The constants are the producer's own answer, and a
// test that asked them would agree with a renamed constant while every shipped
// `jq 'select(.kind == "overlay.compare")'` broke. The literal is the contract;
// the constant is one implementation of it.
func TestCompareJSONGolden(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, comparePopulatedRun()); err != nil {
		t.Fatalf("JSON returned an error: %v", err)
	}
	golden(t, "TestCompareJSONGolden", buf.Bytes())

	raw := buf.String()
	var root map[string]any
	if err := json.Unmarshal(buf.Bytes(), &root); err != nil {
		t.Fatalf("the exported document is not valid JSON: %v\n%s", err, raw)
	}

	if schema, ok := root["schema"].(float64); !ok || int(schema) != 2 {
		t.Errorf(`root["schema"] = %v, want 2 — a fifth kind is ADDITIVE, and bumping the schema for it would announce a break to every consumer of the other four (S047-R1.4)`,
			root["schema"])
	}
	if kind, _ := root["kind"].(string); kind != "overlay.compare" {
		t.Errorf(`root["kind"] = %q, want "overlay.compare" — this is the one key a consumer branches on, so its VALUE is the contract, not merely its presence`,
			kind)
	}
}

// TestCompareJSONGoldenListsArriveAsArrays is the direction guard the golden
// cannot state on its own.
//
// render/json.go refuses to normalize, deliberately: a nil slice reaches the
// wire as null and an empty one as [], and the two say different things — "the
// producer established nothing" against "the producer established an empty
// list". The compare adapter therefore normalizes on its own side, seeding all
// five lists with empty literals before it fills them, so an operator whose
// overlay has no redundant package reads `"redundant": []`.
//
// That guarantee lives in the producer and cannot be re-proved here, since this
// package holds no adapter. The golden beside this test shows one half of it
// since sub-task 7.1: every entry publishes `"further_findings": []` where the
// run established nothing further, which is a package saying "no more" rather
// than a producer saying nothing. Every OTHER list in comparePopulatedRun has
// members, so no other [] appears — read it expecting exactly that one, and its
// absence elsewhere as the fixture speaking rather than the encoder.
//
// What this asserts is the direction: in a document where every list HAS
// members, no key may be null. The day a field is
// restructured such that a populated run publishes a null, this fails with the
// reason attached, where the golden would have shown a null the reviewer had to
// recognise as wrong on sight.
func TestCompareJSONGoldenListsArriveAsArrays(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, comparePopulatedRun()); err != nil {
		t.Fatalf("JSON returned an error: %v", err)
	}

	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.HasSuffix(strings.TrimSpace(line), "null") {
			t.Errorf("a key in a fully populated document arrived as null: %s\nnull is the producer saying nothing, and every list in this run has members (S046-R4.2)",
				strings.TrimSpace(line))
		}
	}
}
