package report

import (
	"fmt"
	"strings"
)

// Mode is how a run renders itself: the four values the --ui flag, the
// BENTOO_UI environment variable and the ui.mode configuration key all accept
// (R3.1). The same four words in all three places, so a value that works in one
// works in the others.
//
// # Why a UI mode lives in the MODEL package
//
// A mode is configuration data — a string with four legal values — not
// formatting. It carries no width, no colour, no escape sequence and no
// terminal dimension, so it crosses none of the presentation rules
// boundary_test.go enforces.
//
// Keeping it here is what lets the renderer and internal/common/tui both depend
// on the resolution without depending on each other (D1): the answer to "which
// renderer" cannot live inside one of the renderers.
type Mode string

const (
	// ModeAuto is a question — "decide for me" — not an answer. It is legal
	// as an INPUT from any source, and ResolveMode never returns it: a
	// renderer cannot render a question.
	ModeAuto Mode = "auto"
	// ModePlain is line-by-line output with no cursor control. It is what a
	// pipe, a log file and a CI job get, and it is the fallback whenever a
	// richer mode was asked for and cannot be delivered.
	ModePlain Mode = "plain"
	// ModeInline redraws in place inside the normal scrollback, leaving the
	// run's output behind when it finishes. It needs a terminal.
	ModeInline Mode = "inline"
	// ModeFullscreen takes over the whole terminal for the duration of the
	// run. It needs a terminal, and it is opt-in only (R3.3): taking over
	// someone's screen is never the consequence of configuring nothing.
	ModeFullscreen Mode = "fullscreen"
)

// modes is the accepted set (R3.1), in the order the rejection message names
// them. The message is DERIVED from this slice rather than written out beside
// it, because S044-R3.9's whole point is that the message names the set that is
// actually accepted — a hand-written list drifts the day a fifth value is added
// and then tells the operator something false.
var modes = []Mode{ModeAuto, ModePlain, ModeInline, ModeFullscreen}

// acceptedModes renders the legal set for a human: "auto, plain, inline or
// fullscreen".
func acceptedModes() string {
	words := make([]string, 0, len(modes))
	for _, mode := range modes {
		words = append(words, string(mode))
	}

	// The short-set guard is not decoration: words[:len(words)-1] panics on an
	// empty slice, and a panic in the path whose job is to REJECT bad input
	// would turn a typo into a crash. It costs one branch to make the
	// rejection path total for any size of modes.
	last := len(words) - 1
	if last < 1 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:last], ", ") + " or " + words[last]
}

// ModeInputs is every input the mode decision reads, taken AS DATA.
//
// # ResolveMode does not read the environment, and that is the point
//
// There is no os.Getenv in this file. Env and Interactive are supplied by the
// caller, which is what makes the whole precedence matrix testable without a
// TTY and without mutating the environment of a running test binary — the
// reason this struct exists at all instead of four positional arguments.
//
// The environment reading belongs at the edge, in cmd/bentoo, next to the flag
// definitions.
type ModeInputs struct {
	// Flag is the --ui value, empty when the flag was not passed.
	Flag string
	// Env is the BENTOO_UI value, empty when the variable is unset. Per the
	// same convention tui.Enabled follows, an empty value means "not set".
	Env string
	// Config is the ui.mode configuration value, empty when the key is
	// absent. Absent is the case S044-R3.7 protects: it must produce exactly
	// today's behaviour.
	Config string
	// NoTUI is the opt-out layer, and the CALLER MUST FOLD ALL THREE OPT-OUTS
	// INTO IT:
	//
	//	NoTUI: noTUIFlag ||
	//		os.Getenv("BENTOO_NO_TUI") != "" ||
	//		os.Getenv("NO_COLOR") != ""
	//
	// All three, not just the flag. tui.Enabled returns false on THREE
	// opt-outs — the --no-tui flag, NO_COLOR and BENTOO_NO_TUI — and this
	// struct deliberately has no NO_COLOR field of its own. Drop NO_COLOR
	// from that expression and a user who set it, and configured nothing
	// about ui.mode, silently starts getting inline output where they get
	// plain today: precisely the change S044-R3.7 forbids.
	//
	// It is applied as ModePlain at the flag layer (R3.4), so --no-tui is an
	// alias for --ui=plain rather than a second mechanism competing with it.
	NoTUI bool
	// Interactive is the stdout-TTY answer ALONE — output.IsTerminal(), the
	// same probe tui.Enabled ends on.
	//
	// It is NOT tui.Enabled's full return value. That answer already folds
	// the opt-outs in, and folding them in twice would report an opted-out
	// run as a terminal that "cannot support" the mode, which is a false
	// sentence: the terminal was fine, the operator opted out. The opt-outs
	// go in NoTUI; the capability goes here.
	Interactive bool
}

// ResolveMode returns the effective mode and, when it had to downgrade, one
// sentence for stderr. It never returns ModeAuto.
//
// Precedence is R3.2: the --ui flag, then BENTOO_UI, then ui.mode, then auto.
// The first source that speaks decides; the ones below it are not consulted.
//
// # Composition, and why S044-R3.7 holds by construction
//
// With nothing configured, this reproduces tui.Enabled exactly: NoTUI yields
// plain, and otherwise auto yields inline on a terminal and plain off one —
// the same two answers today's boolean gives, under names instead of a bool.
// A user who configured no ui.mode therefore sees no change, which is S044-R3.7
// satisfied by construction rather than by a compatibility branch.
//
// # auto never yields fullscreen (R3.3)
//
// Full screen takes over the terminal, so it is opt-in: no combination that
// leaves the mode unstated may produce it. auto on a terminal is inline.
//
// # Every stated source is validated, not just the deciding one
//
// A value outside the accepted set is refused wherever it appears, even when
// a higher-precedence source would have outranked it. Two reasons:
//
//   - S044-R3.9 is unconditional — "IF --ui is given a value outside the accepted
//     set" says nothing about --no-tui also being passed, so validating only
//     the source that happens to win would let `--ui=bogus --no-tui` run.
//   - A typo that is merely outranked today decides the run tomorrow, the
//     first time the flag above it is dropped. Naming it now costs one
//     message; hiding it costs a confusing run later.
//
// The message names the source it came from (--ui, BENTOO_UI or ui.mode) as
// well as the accepted set, because "which of my three places is wrong" is the
// question the operator actually has.
//
// # A refusal that is outranked is a WARNING, never an error (D10)
//
// Validating every source and letting every refusal ABORT the resolution are
// two different rules, and this file used to run them together: `--ui=fullscreen`
// with a stale `BENTOO_UI=bogus` failed, and every caller fell back to plain —
// a shell profile the operator forgot silently outranking the flag they just
// typed, on every command that produces a report. That is the inversion S046-R3.3
// exists to forbid and the one S046-R3.7 calls "refused in words, never in
// effect".
//
// So the refusal keeps its whole sentence and loses only its reach. A source
// that does not parse yields an ERROR only when no source above it named a
// mode; otherwise it yields a WARNING and the higher source decides. Both
// halves stay measurable: an ambient-only refusal still errors, because there
// the refused source IS the highest-precedence one that spoke, and `--ui=bogus`
// still errors whatever legal value sits below it, because nothing sits above
// `--ui`.
//
// Precedence is read off the three STRING sources alone. --no-tui is a boolean
// opt-out rather than a source that names a mode, so it neither rescues a
// refusal below it nor excuses one above it — which is what keeps
// `--ui=bogus --no-tui` rejected under S044-R3.9.
//
// # The opt-out outranks an explicit --ui, deliberately
//
// --no-tui together with --ui=fullscreen is a contradiction, and plain is the
// safe reading of it: the opt-out also carries NO_COLOR and BENTOO_NO_TUI (see
// ModeInputs.NoTUI), and a terminal opt-out that a flag could override would
// not be an opt-out. It downgrades silently because --no-tui IS --ui=plain
// (R3.4), and asking for plain and getting plain is not a downgrade.
//
// # The warning is returned, never printed
//
// This package has no output. The caller prints the sentence on stderr — on
// stderr so it never contaminates piped output, and ONCE, which holds because
// the mode is resolved once per run (D4).
//
// The empty string is the success signal: a resolve that delivered what was
// asked for returns "". A sentence on every call would train the operator to
// ignore the one call that matters.
func ResolveMode(in ModeInputs) (Mode, string, error) {
	req, err := requestedMode(in)
	if err != nil {
		return "", "", err
	}

	mode, downgrade := deliverableMode(req.mode, in.Interactive)
	return mode, req.warning(mode, downgrade), nil
}

// deliverableMode answers the second of the two questions ResolveMode asks:
// given what was requested, what can THIS terminal actually carry, and does the
// difference owe the operator a sentence.
//
// It is split out from ResolveMode so that the sentence an outranked refusal
// produces can be composed against the mode that was finally delivered rather
// than the one the precedence walk named — those differ whenever --no-tui or a
// non-terminal stdout has the last word, and a warning that named the wrong
// mode would be exactly the false statement S046-R3.7 exists to prevent.
func deliverableMode(requested Mode, interactive bool) (Mode, string) {
	// auto is the only value that reads the terminal to pick BETWEEN modes.
	// Fullscreen is not a candidate here at any interactivity (R3.3).
	if requested == ModeAuto {
		if interactive {
			return ModeInline, ""
		}
		return ModePlain, ""
	}

	// Plain always works — it is the mode that assumes nothing about the
	// terminal — and inline and fullscreen work on a terminal. Nothing was
	// downgraded in either case, so there is nothing to say.
	if requested == ModePlain || interactive {
		return requested, ""
	}

	// S044-R3.6: a mode was requested explicitly and this terminal cannot carry
	// it. That is a downgrade, not a failure — the run still produces its
	// report — so it returns a sentence rather than an error.
	return ModePlain, fmt.Sprintf(
		"%s output needs an interactive terminal and stdout is not one, so this run renders in plain",
		requested,
	)
}

// modeRequest is what the precedence walk decided, before the terminal is
// consulted. It exists so that requestedMode can report a refusal it did NOT
// act on — the D10 case — without a second error return that every caller would
// have to remember is not fatal.
type modeRequest struct {
	// mode is what the highest-precedence source that spoke named, ModePlain
	// when --no-tui had the last word, or ModeAuto when nothing spoke.
	mode Mode
	// statedBy names the source that decided, for the sentence an outranked
	// refusal produces. Empty when no source named a mode.
	statedBy string
	// outranked holds the refusals that lost: sources whose value does not
	// parse but which a higher-precedence source had already overruled. They
	// are warnings and never failures (D10). A slice rather than one entry
	// because two ambient sources can both be unusable under one valid flag,
	// and naming only the first would drop the second silently — the omission
	// S046-R2.3 forbids one level up.
	outranked []error
}

// warning renders the one sentence the caller puts on stderr, for the mode that
// was actually delivered.
//
// Two sentences can be owed at once — an outranked refusal AND a terminal
// downgrade — and both are stated, joined, rather than one being dropped for
// fitting the single string this returns. The empty string stays the success
// signal.
func (r modeRequest) warning(delivered Mode, downgrade string) string {
	if len(r.outranked) == 0 {
		return downgrade
	}

	refusals := make([]string, 0, len(r.outranked))
	for _, err := range r.outranked {
		refusals = append(refusals, err.Error())
	}

	// "without effect" is the whole point of the sentence: the operator is
	// told their value was refused AND that the refusal changed nothing, so a
	// stale key in a shell profile never reads as the cause of the mode they
	// got.
	sentence := fmt.Sprintf("%s — outranked by %s, so refused without effect; this run renders in %s",
		strings.Join(refusals, "; "), r.statedBy, delivered)
	if downgrade == "" {
		return sentence
	}
	return sentence + "; " + downgrade
}

// requestedMode applies R3.2's precedence and S044-R3.9's refusal, returning the
// mode that was ASKED FOR — which may still be ModeAuto, and which ResolveMode
// then resolves against the terminal.
//
// Splitting "what was asked for" from "what can be delivered" is what keeps the
// two rules separable: precedence and validation live here, terminal capability
// lives in deliverableMode, and neither has to reason about the other.
//
// The error return is now narrow by construction: it fires only for a refusal
// that nothing outranks. Everything else a source got wrong comes back on
// modeRequest.outranked, which is a sentence rather than a stop (D10).
func requestedMode(in ModeInputs) (modeRequest, error) {
	// In precedence order (R3.2), so the FIRST invalid value reported is the
	// one from the highest-priority source — the one the operator most
	// likely just typed.
	sources := []struct {
		name  string
		value string
	}{
		{"--ui", in.Flag},
		{"BENTOO_UI", in.Env},
		{"ui.mode", in.Config},
	}

	// The empty Mode is a safe sentinel for "no source has spoken yet": it is
	// not one of the four legal values, so parseMode can never produce it.
	req := modeRequest{mode: Mode("")}
	for _, source := range sources {
		if source.value == "" {
			// Unset. An empty value is "not set", never a fifth mode.
			continue
		}
		mode, err := parseMode(source.name, source.value)
		if err != nil {
			if req.mode == "" {
				// Nothing above this source named a mode, so the refused
				// source IS the highest-precedence one that spoke and the
				// refusal decides the run. `--ui=bogus` always lands here:
				// nothing sits above --ui.
				return modeRequest{}, err
			}
			// A source above already decided. The refusal is stated and
			// discarded (S046-R3.3, S046-R3.7 via D10).
			req.outranked = append(req.outranked, err)
			continue
		}
		if req.mode == "" {
			// The highest-precedence source that spoke. The loop keeps
			// going to validate the rest; it does not keep choosing.
			req.mode = mode
			req.statedBy = source.name
		}
	}

	// The flag layer, after validation so that --ui=bogus is still refused
	// when --no-tui is also passed (S044-R3.9), and before the stated mode is
	// returned so that the opt-out outranks an explicit --ui.
	//
	// statedBy is deliberately LEFT as the string source that spoke: --no-tui
	// changes which mode is delivered, not which source outranked the refusal,
	// and the delivered mode is carried into the sentence separately.
	if in.NoTUI {
		req.mode = ModePlain
		return req, nil
	}
	if req.mode == "" {
		// Nothing spoke at any layer: auto (R3.2's last step).
		req.mode = ModeAuto
		return req, nil
	}
	return req, nil
}

// parseMode turns one source's raw value into a Mode, or explains why it is not
// one (S044-R3.9).
//
// The match is EXACT: no trimming, no case folding. Not an oversight — the same
// four words are read from three different places, and a normalization applied
// here would have to be applied identically by every future reader of ui.mode
// or the config would accept a value the flag rejects. Strictness is also the
// reversible choice: a rejected "Inline" can be accepted later, while a value
// silently accepted today cannot be rejected without breaking someone.
func parseMode(source, value string) (Mode, error) {
	for _, mode := range modes {
		if value == string(mode) {
			return mode, nil
		}
	}
	return "", fmt.Errorf("%s: %q is not a UI mode; the accepted values are %s", source, value, acceptedModes())
}
