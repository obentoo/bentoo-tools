package main

// What the report does with a MODEL's words and with a maintainer's own reason.
//
// S032-R5.2, S032-R5.3 and S032-R5.4, plus the declared reason that completes a
// FindingDeclaredDivergence line.
//
// These four values crossed the producer/renderer boundary during story 047 as
// Finding.Origin, Finding.Effect and Finding.Proposal, and nothing on this side
// read them: the values arrived, no line was built from them, and four standing
// requirements rendered as silence. No test failed, because none existed — the
// only witness was `unused` on the constants left behind. These cases are what
// makes losing them again a FAILURE rather than a lint warning.

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/obentoo/bentoolkit/internal/overlay"
)

func TestComparePackageFindingsRenderTheMaintainersReason(t *testing.T) {
	const reason = "keeps our gcc-17 build fix until ::gentoo carries it"

	reasons, extra := comparePackageFindings([]overlay.Finding{{
		Kind:   overlay.FindingDeclaredDivergence,
		Atom:   "dev-lang/go",
		Detail: "patched — declared by dev-lang/go",
		Effect: overlay.Effect{Text: reason, Source: overlay.EffectDeclared},
	}})

	want := "patched — declared by dev-lang/go: " + reason
	if got := reasons["dev-lang/go"]; got != want {
		t.Errorf("the declaration line is %q, want %q — the reason is what the entry claims the "+
			"divergence DOES, and the line without it names an entry and states nothing", got, want)
	}
	if len(extra["dev-lang/go"]) != 0 {
		t.Errorf("the declared reason also produced %q; it completes the declaration line rather "+
			"than standing beside it", extra["dev-lang/go"])
	}
}

func TestComparePackageFindingsRenderAModelsWords(t *testing.T) {
	const summary = "adds a musl build fix ::gentoo does not carry"
	const proposal = "musl build fix carried on top of ::gentoo"

	reasons, extra := comparePackageFindings([]overlay.Finding{{
		Kind:     overlay.FindingUndeclaredDivergence,
		Atom:     "dev-libs/foo",
		Detail:   "undeclared divergence — 12 added, 3 removed",
		Effect:   overlay.Effect{Text: summary, Source: overlay.EffectReviewed},
		Origin:   overlay.OriginOverlay,
		Proposal: proposal,
	}})

	lines := extra["dev-libs/foo"]
	if len(lines) != 2 {
		t.Fatalf("the model's words rendered as %+v, want the reading (S032-R5.2, S032-R5.3) "+
			"followed by the proposal (S032-R5.4)", lines)
	}
	for _, want := range []string{"model reading", "originates in the overlay", summary} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("the reading line %q does not name %q", lines[0], want)
		}
	}
	for _, want := range []string{"proposed declaration", "nothing here writes it", proposal} {
		if !strings.Contains(lines[1], want) {
			t.Errorf("the proposal line %q does not name %q", lines[1], want)
		}
	}
	if got := reasons["dev-libs/foo"]; strings.Contains(got, summary) {
		t.Errorf("a model's reading reached the finding's own sentence (%q); the sentence is what "+
			"this tool established by comparing two files, and the reading is a guess", got)
	}
}

func TestCompareEffectSourceDecidesWhichLineItIs(t *testing.T) {
	// The whole reason Effect carries a Source. An operator who cannot tell the
	// two apart will act on the wrong one, and the line that invites an action —
	// "declare this patched" — is the guess.
	reviewed := overlay.Finding{
		Effect: overlay.Effect{Text: "a model's reading", Source: overlay.EffectReviewed},
		Origin: overlay.OriginOverlay,
	}
	if tail := compareDeclaredTail(reviewed); tail != "" {
		t.Errorf("a model's reading became a declared tail (%q), where it reads as a commitment "+
			"somebody made on purpose", tail)
	}

	declared := overlay.Finding{
		Effect: overlay.Effect{Text: "a maintainer's commitment", Source: overlay.EffectDeclared},
		Origin: overlay.OriginOverlay,
	}
	if lines := compareReviewLines(declared); len(lines) != 0 {
		t.Errorf("a maintainer's own reason was printed under a model's lead: %q", lines)
	}
}

func TestCompareReviewLinesSayNothingAtEveryZeroValue(t *testing.T) {
	// Every run that asked for no review carries these zeros, so this is also
	// what keeps such a run's rendering unchanged.
	cases := map[string]overlay.Finding{
		"a run no review reached": {},
		"a reading with no origin": {
			Effect: overlay.Effect{Text: "something", Source: overlay.EffectReviewed},
		},
		"an origin with no reading": {Origin: overlay.OriginOverlay},
		"an origin nobody defined a sentence for": {
			Effect: overlay.Effect{Text: "something", Source: overlay.EffectReviewed},
			Origin: overlay.ReviewOrigin(99),
		},
	}
	for name, finding := range cases {
		if lines := compareReviewLines(finding); len(lines) != 0 {
			t.Errorf("%s rendered %q; a note answering neither S032-R5.2 nor S032-R5.3 must render "+
				"nothing, because a finding-shaped line stating nothing is worse than no line", name, lines)
		}
	}
}

func TestCompareReviewLinesRefuseAProposalOnAnyOtherOrigin(t *testing.T) {
	for _, origin := range []overlay.ReviewOrigin{overlay.OriginUpstream, overlay.OriginBoth} {
		lines := compareReviewLines(overlay.Finding{
			Effect:   overlay.Effect{Text: "a reading", Source: overlay.EffectReviewed},
			Origin:   origin,
			Proposal: "declare it patched",
		})
		if len(lines) != 1 {
			t.Fatalf("origin %v rendered %+v, want the reading on its own", origin, lines)
		}
		if strings.Contains(lines[0], "declare it patched") {
			t.Errorf("origin %v carried a proposal onto the line (%q). A copy that has fallen "+
				"behind ::gentoo needs the rebase first, and declaring it patched would record the "+
				"whole difference as intentional, suppressing that recommendation permanently",
				origin, lines[0])
		}
	}
}

func TestCompareCappedKeepsTextValidUTF8(t *testing.T) {
	long := strings.Repeat("ç", compareEffectCap+10)
	got := compareCapped(long)

	if !utf8.ValidString(got) {
		t.Errorf("compareCapped produced invalid UTF-8 (%q); the text reaches a JSON export as "+
			"well as a terminal, and a cut through a multi-byte rune corrupts both", got)
	}
	if n := utf8.RuneCountInString(got); n > compareEffectCap {
		t.Errorf("compareCapped returned %d runes, want at most %d", n, compareEffectCap)
	}
	if short := "fits inside the cap"; compareCapped(short) != short {
		t.Errorf("compareCapped altered a string already within the cap: %q", compareCapped(short))
	}
}
