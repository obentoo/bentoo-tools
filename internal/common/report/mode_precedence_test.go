package report

import (
	"strings"
	"testing"
)

// mode_precedence_test.go fills the one cell the mode matrix never had: a VALID
// higher-precedence source paired with an INVALID lower one.
//
// S046-R3.3 orders the sources — --no-tui, then --ui, then BENTOO_UI, then
// ui.mode, then auto. S046-R3.7 says in as many words that a source S046-R3.3
// already outranks is "refused in words, never in effect". A refusal that
// aborts the whole resolution inverts both: the flag has already spoken, and a
// stale environment variable below it decides the run by making the answer
// unavailable — and the sentence the operator was owed becomes an error that
// stops the run instead.
//
// Every test here carries the TestModePrecedence prefix so `-run
// TestModePrecedence` selects the whole file and nothing else.

// TestModePrecedenceValidFlagSurvivesAnInvalidEnv is the hostile half of
// S046-R3.7: the source the operator just typed must not lose to the one they
// forgot they set.
//
// It is written against the pair that makes the rule fire wrongly — a legal
// --ui with an illegal BENTOO_UI — because the benign pair, where both parse,
// has been green since the package existed and says nothing about precedence
// under refusal.
func TestModePrecedenceValidFlagSurvivesAnInvalidEnv(t *testing.T) {
	in := ModeInputs{Flag: "fullscreen", Env: "bogus", Interactive: true}

	got, warning, err := ResolveMode(in)
	if err != nil {
		t.Fatalf("ResolveMode(%+v) returned an error: %v\n"+
			"    --ui named a legal mode and BENTOO_UI is beneath it in the precedence order, so the\n"+
			"    unusable ambient value may be refused but may not decide the run (S046-R3.3, S046-R3.7)", in, err)
	}
	if got != ModeFullscreen {
		t.Fatalf("ResolveMode(%+v) = %q, want %q: the explicit flag decides", in, got, ModeFullscreen)
	}
	if !strings.Contains(warning, "BENTOO_UI") || !strings.Contains(warning, "bogus") {
		t.Fatalf("ResolveMode(%+v) warned %q, want a sentence naming BENTOO_UI and the value it refused:\n"+
			"    the refusal loses its effect here, so it must keep its voice (S046-R3.7)", in, warning)
	}
}

// TestModePrecedenceValidEnvSurvivesAnInvalidConfig is the same rule one rung
// down, and it is not a duplicate: a fix that special-cased the flag alone
// would satisfy the test above and leave ui.mode outranking BENTOO_UI.
func TestModePrecedenceValidEnvSurvivesAnInvalidConfig(t *testing.T) {
	in := ModeInputs{Env: "inline", Config: "bogus", Interactive: true}

	got, warning, err := ResolveMode(in)
	if err != nil {
		t.Fatalf("ResolveMode(%+v) returned an error: %v\n"+
			"    BENTOO_UI named a legal mode and ui.mode sits beneath it (S046-R3.3)", in, err)
	}
	if got != ModeInline {
		t.Fatalf("ResolveMode(%+v) = %q, want %q: the environment decides when the flag is silent", in, got, ModeInline)
	}
	if !strings.Contains(warning, "ui.mode") || !strings.Contains(warning, "bogus") {
		t.Fatalf("ResolveMode(%+v) warned %q, want a sentence naming ui.mode and the value it refused (S046-R3.7)", in, warning)
	}
}

// TestModePrecedenceAmbientRefusalAloneStillErrors is the converse half. The
// fix above must narrow the refusal's REACH, not remove it: where the refused
// source IS the highest-precedence one that spoke, nothing outranks it and the
// error stands — that is the ambient-only behaviour Task 12 and Task 13 put in
// place, and a fix that swallowed it would trade one defect for another.
func TestModePrecedenceAmbientRefusalAloneStillErrors(t *testing.T) {
	cases := []struct {
		name string
		in   ModeInputs
	}{
		{name: "BENTOO_UI alone", in: ModeInputs{Env: "bogus", Interactive: true}},
		{name: "ui.mode alone", in: ModeInputs{Config: "bogus", Interactive: true}},
		{name: "both ambient sources unusable", in: ModeInputs{Env: "bogus", Config: "worse", Interactive: true}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ResolveMode(tc.in)
			if err == nil {
				t.Fatalf("ResolveMode(%+v) returned no error: no source outranks the refused one here, "+
					"so the refusal still decides (S046-R3.7)", tc.in)
			}
			if !strings.Contains(err.Error(), "bogus") {
				t.Fatalf("ResolveMode(%+v) errored %q, want the refused value named", tc.in, err)
			}
		})
	}
}

// TestModePrecedenceInvalidFlagAlwaysErrors pins the half that must NOT move.
// --ui=bogus is the operator's own typo at the top of the order, and nothing
// below it — a legal BENTOO_UI, a legal ui.mode, or --no-tui — turns it into a
// warning (S046-R3.2 is unconditional: a --ui outside the accepted set rejects
// the run before any work).
func TestModePrecedenceInvalidFlagAlwaysErrors(t *testing.T) {
	cases := []struct {
		name string
		in   ModeInputs
	}{
		{name: "with a legal env beneath it", in: ModeInputs{Flag: "bogus", Env: "plain", Interactive: true}},
		{name: "with a legal config beneath it", in: ModeInputs{Flag: "bogus", Config: "plain", Interactive: true}},
		{name: "with the opt-out also set", in: ModeInputs{Flag: "bogus", NoTUI: true, Interactive: true}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ResolveMode(tc.in)
			if err == nil {
				t.Fatalf("ResolveMode(%+v) returned no error: the highest-precedence source is the "+
					"unusable one, and no legal value beneath it makes a typo above it legal", tc.in)
			}
			if !strings.Contains(err.Error(), "--ui") {
				t.Fatalf("ResolveMode(%+v) errored %q, want the source named so the operator knows which "+
					"of their three places is wrong", tc.in, err)
			}
		})
	}
}

// TestModePrecedenceNoTUIOutranksAValidFlag pins the deliberate inversion the
// package documents: --no-tui heads the order in S046-R3.3, and an opt-out a flag
// could override would not be an opt-out. Present so a fix that re-orders the
// sources cannot quietly re-order this one too.
func TestModePrecedenceNoTUIOutranksAValidFlag(t *testing.T) {
	in := ModeInputs{Flag: "fullscreen", NoTUI: true, Interactive: true}

	got, _, err := ResolveMode(in)
	if err != nil {
		t.Fatalf("ResolveMode(%+v) returned an error: %v", in, err)
	}
	if got != ModePlain {
		t.Fatalf("ResolveMode(%+v) = %q, want %q: the opt-out outranks an explicit --ui", in, got, ModePlain)
	}
}
