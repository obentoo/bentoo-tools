package config

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

// The review budget (S048-R3) is the deadline `overlay compare`'s LLM review
// runs under. It becomes a configuration key here, and everything this file
// asserts follows from three sentences of the requirement:
//
//	S048-R3.1  a configured value replaces the default
//	S048-R3.2  absent, zero and negative all resolve to the documented default,
//	           and so does a nil pointer
//	S048-R3.3  the unit is an integer of SECONDS, as cache_ttl, http_timeout
//	           and autoupdate.validate.timeout already are
//	S048-R3.4  the key nests inside an existing top-level block, so probeConfig
//	           needs no hand-maintained mirror entry
//
// The tests are ordered hostile-first: the two cases where the new block could
// be wired WRONGLY and still look right come before the cases where it plainly
// holds. `review:` and `validate:` are near-identical siblings under
// `autoupdate:` — same shape, same unit, same getter name — and the ways they
// break are the ways two near-identical things break: they collapse into one,
// or one of them stops being reachable at all.

// reviewConfigYAML builds a config body carrying an `autoupdate:` block, on top
// of the minimal overlay stanza LoadFrom needs.
func reviewConfigYAML(autoupdateBody string) string {
	return minimalValidateConfigYAML + "autoupdate:\n" + autoupdateBody
}

// TestReviewTimeout_DoesNotCollapseIntoTheValidateTimeout is the hostile half of
// S048-R3.1/R3.4, in the collapse direction: two DIFFERENT budgets written under
// two DIFFERENT keys must stay two budgets.
//
// The failure this forbids is the one a copy of ValidateConfig invites. The new
// block is written by mirroring `Validate ValidateConfig \`yaml:"validate"\“,
// and a yaml tag carried over with the rest reads `review:` as nothing and lets
// `validate:` answer for both. Every default test in this file would still pass
// under that wiring, because the defaults are what an unread block returns.
func TestReviewTimeout_DoesNotCollapseIntoTheValidateTimeout(t *testing.T) {
	cfg, _ := writeValidateConfig(t, reviewConfigYAML(
		"  review:\n"+
			"    timeout: 300\n"+
			"  validate:\n"+
			"    timeout: 900\n"))

	if got, want := cfg.Autoupdate.Review.GetTimeout(), 300*time.Second; got != want {
		t.Errorf("autoupdate.review.timeout: GetTimeout() = %v, want %v — the review budget is answering with something other than its own key (S048-R3.1)", got, want)
	}
	if got, want := cfg.Autoupdate.Validate.GetTimeout(), 900*time.Second; got != want {
		t.Errorf("autoupdate.validate.timeout: GetTimeout() = %v, want %v — the new review block moved the per-agent budget that was already there", got, want)
	}
	if cfg.Autoupdate.Review.GetTimeout() == cfg.Autoupdate.Validate.GetTimeout() {
		t.Errorf("both budgets answer %v; `review:` and `validate:` are two keys carrying two values and a single yaml tag reading for both collapses them into one",
			cfg.Autoupdate.Review.GetTimeout())
	}
}

// TestReviewTimeout_NeitherBlockMovesTheOther is the converse hostile half: one
// block written alone must leave the other at ITS OWN default, in both
// directions.
//
// A wiring that satisfies the collapse case above by pointing the two getters at
// one shared field would be caught here, and a wiring that satisfies this case
// by giving the review block the validate block's default would be caught by the
// second half of each pair — the defaults differ, and that difference is the
// evidence the two blocks are separate.
func TestReviewTimeout_NeitherBlockMovesTheOther(t *testing.T) {
	t.Run("review alone leaves validate at its default", func(t *testing.T) {
		cfg, _ := writeValidateConfig(t, reviewConfigYAML(
			"  review:\n"+
				"    timeout: 300\n"))

		if got, want := cfg.Autoupdate.Review.GetTimeout(), 300*time.Second; got != want {
			t.Errorf("GetTimeout() = %v, want %v — the configured review budget was not read (S048-R3.1)", got, want)
		}
		if got, want := cfg.Autoupdate.Validate.GetTimeout(), DefaultValidateTimeout*time.Second; got != want {
			t.Errorf("setting autoupdate.review.timeout moved the validate budget to %v, want %v — an unrelated key must not move another default", got, want)
		}
	})

	t.Run("validate alone leaves review at its default", func(t *testing.T) {
		cfg, _ := writeValidateConfig(t, reviewConfigYAML(
			"  validate:\n"+
				"    timeout: 900\n"))

		if got, want := cfg.Autoupdate.Validate.GetTimeout(), 900*time.Second; got != want {
			t.Errorf("GetTimeout() = %v, want %v — the validate budget stopped being read", got, want)
		}
		if got, want := cfg.Autoupdate.Review.GetTimeout(), DefaultReviewTimeout*time.Second; got != want {
			t.Errorf("setting autoupdate.validate.timeout moved the review budget to %v, want %v — the review budget answers DefaultReviewTimeout and nothing else (S048-R3.2)", got, want)
		}
	})
}

// TestReviewTimeout_IsReachableAtAutoupdateReviewTimeout pins the KEY PATH, and
// with it the reason the key is nested at all (S048-R3.4).
//
// probeConfig (config.go:335) is a hand-maintained mirror of Config's TOP-LEVEL
// keys and no test guards it: a new top-level block absent from that mirror
// makes every command print "field <key> not found in type config.probeConfig"
// to stderr while loading the value anyway. Nesting inside AutoupdateConfig
// inherits the mirror for free — so the stderr assertion below is not decoration,
// it is the whole reason this key lives under `autoupdate:` and not beside it.
func TestReviewTimeout_IsReachableAtAutoupdateReviewTimeout(t *testing.T) {
	cfg, stderr := writeValidateConfig(t, reviewConfigYAML(
		"  review:\n"+
			"    timeout: 300\n"))

	if got, want := cfg.Autoupdate.Review.GetTimeout(), 300*time.Second; got != want {
		t.Errorf("autoupdate.review.timeout = 300 resolved to %v, want %v (S048-R3.1, S048-R3.3: the value is an int of seconds)", got, want)
	}
	if strings.Contains(stderr, "not found in type") {
		t.Errorf("loading a config that sets autoupdate.review.timeout warns on stderr: %q\n"+
			"the strict probe does not recognize the key, which is the probeConfig trap S048-R3.4 exists to avoid — "+
			"a key the operator sets correctly and every command complains about", strings.TrimSpace(stderr))
	}
}

// TestReviewTimeout_AbsentZeroAndNegativeAllAnswerTheDefault is S048-R3.2 in
// full: an absent block, a half-written block and a nonsense value are three
// spellings of "not configured", and all three must reach the SAME documented
// default. A zero left by a template, or a negative left by a subtraction, must
// not become a budget of zero seconds — that would kill every review instantly
// and look exactly like the timeout defect this story is fixing.
//
// The expected value is written as DefaultReviewTimeout rather than as a
// literal: the constant's VALUE is measured and set by a later sub-task, and a
// test that pinned today's number would have to be edited to record that
// measurement. What is pinned here is that the getter answers the constant.
func TestReviewTimeout_AbsentZeroAndNegativeAllAnswerTheDefault(t *testing.T) {
	if DefaultReviewTimeout <= 0 {
		t.Fatalf("DefaultReviewTimeout = %d; a non-positive default makes every review's budget the thing that kills it", DefaultReviewTimeout)
	}
	want := time.Duration(DefaultReviewTimeout) * time.Second

	tests := []struct {
		name string
		body string
	}{
		{name: "no autoupdate block at all", body: minimalValidateConfigYAML},
		{name: "an autoupdate block with no review key", body: reviewConfigYAML("  cache_ttl: 3600\n")},
		{name: "a review block that sets nothing else", body: reviewConfigYAML("  review: {}\n")},
		{name: "timeout: 0", body: reviewConfigYAML("  review:\n    timeout: 0\n")},
		{name: "timeout: -1", body: reviewConfigYAML("  review:\n    timeout: -1\n")},
		{name: "timeout: -3600", body: reviewConfigYAML("  review:\n    timeout: -3600\n")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _ := writeValidateConfig(t, tt.body)
			if got := cfg.Autoupdate.Review.GetTimeout(); got != want {
				t.Errorf("GetTimeout() = %v, want %v — %s must resolve to the documented default (S048-R3.2)", got, want, tt.name)
			}
		})
	}
}

// TestReviewTimeout_PositiveValueIsReadAsSeconds is the benign case, and it is
// last on purpose: it is the one every wiring above would also pass.
//
// It carries the unit (S048-R3.3). YAML has no duration literal, so the key is
// an int of seconds and the conversion happens in the getter, at the boundary —
// a value read as nanoseconds or as minutes would satisfy "the number was read"
// and still hand the reviewer the wrong deadline.
func TestReviewTimeout_PositiveValueIsReadAsSeconds(t *testing.T) {
	for _, seconds := range []int{1, 45, 300, 900} {
		cfg, _ := writeValidateConfig(t, reviewConfigYAML(
			"  review:\n    timeout: "+strconv.Itoa(seconds)+"\n"))

		got := cfg.Autoupdate.Review.GetTimeout()
		if want := time.Duration(seconds) * time.Second; got != want {
			t.Errorf("timeout: %d resolved to %v, want %v — the key is an integer of SECONDS, the unit cache_ttl, http_timeout and autoupdate.validate.timeout already use (S048-R3.3)",
				seconds, got, want)
		}
	}
}

// TestReviewTimeout_NilReceiverAnswersTheDefault follows the discipline
// (*ValidateConfig).GetTimeout already shows (config.go:870): a getter reached
// through a nil pointer answers the default rather than panicking, because the
// config is threaded through call sites that predate this block. S048-R3.2 names
// the nil pointer explicitly alongside the absent and half-written blocks.
func TestReviewTimeout_NilReceiverAnswersTheDefault(t *testing.T) {
	var r *ReviewConfig

	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("(*ReviewConfig).GetTimeout panicked on a nil receiver (%v); the sibling getter answers a default for one, and `overlay compare` reaches this through call sites that predate the block", rec)
		}
	}()

	if got, want := r.GetTimeout(), time.Duration(DefaultReviewTimeout)*time.Second; got != want {
		t.Errorf("a nil *ReviewConfig answers %v, want %v — the nil pointer is the third spelling of an unconfigured budget (S048-R3.2)", got, want)
	}
}
