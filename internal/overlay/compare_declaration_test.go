package overlay

import (
	"strings"
	"testing"
)

// The declaration line is what closes the gap between "the struct knows a
// package is patched" and "the operator does". Tasks 1-6 satisfied R2.2 in the
// struct — PatchedBy is set and asserted by compare_divergence_test.go — but the
// only path from that field to the terminal was the stale-declaration finding,
// which needs a local repository copy to fire. With an API-only provider
// Verified is always NotVerified, so a patched package rendered exactly like an
// unpatched one: same Verdict "keep", no column, no line. That is the story's
// opening ambiguity one level down, which is why R3.8 exists.
//
// Every case below asserts on what the run ESTABLISHED rather than on a helper,
// so a declaration that is composed but never reaches the report would fail here.
//
// Story 047, sub-task 5.5 (S047-R8.2): that used to mean FormatReport's output.
// It now means CompareReport.Findings, which is what `func comparePackageFindings`
// in cmd/bentoo turns into the row's REASON cell and the package's notes. The
// change removes the one weakness the old form had: FindingStaleDeclaration and
// FindingDeclaredDivergence both name the entry, so telling them apart meant
// matching the prose "declared by", and a reword would have made the "no second
// line" case pass over a report carrying both.

// declFindings is what a run over these results establishes.
func declFindings(results ...CompareResult) []Finding {
	report := &CompareReport{Results: results}
	EstablishFindings(report)
	return report.Findings
}

// declFinding returns pkg's declared-divergence finding, or the zero Finding and
// false when the run established none.
func declFinding(findings []Finding, pkg string) (Finding, bool) {
	for _, f := range findings {
		if f.Kind == FindingDeclaredDivergence && strings.HasSuffix(f.Atom, "/"+pkg) {
			return f, true
		}
	}
	return Finding{}, false
}

// _Requirements: R2.2, R3.8_
func TestRendersPatchedDeclaration(t *testing.T) {
	const (
		entry  = "app-editors/zed@stable"
		reason = "wayland IME patch we carry until upstream lands it"
	)

	patched := CompareResult{
		Category: "app-editors", Package: "zed",
		LocalVersion: "1.0", RemoteVersion: "1.0",
		Status: StatusUpToDate, Verdict: VerdictKeep,
		Patched: true, PatchedBy: entry, PatchedReason: reason,
		Verified: NotVerified,
	}

	t.Run("a patched package names its entry and reason with no verification", func(t *testing.T) {
		found, ok := declFinding(declFindings(patched), "zed")
		if !ok {
			t.Fatal("no declaration was established for a patched package; the report is silent about the divergence it was told about")
		}
		if found.Entry != entry {
			t.Errorf("the declaration names %q as its entry, want %q — that key is what the operator greps the registry for", found.Entry, entry)
		}
		if found.Effect.Text != reason {
			t.Errorf("the declaration carries %q as its reason, want %q", found.Effect.Text, reason)
		}
		if found.Effect.Source != EffectDeclared {
			t.Errorf("the reason arrives as %v, not as the maintainer's own; a declaration read as a model's guess is a commitment demoted to an opinion", found.Effect.Source)
		}
	})

	t.Run("a stale declaration keeps its warning and gains no declaration line", func(t *testing.T) {
		stale := patched
		stale.Verified = VerifiedIdentical

		findings := declFindings(stale)

		if findingOfKind(findings, FindingStaleDeclaration) == nil {
			t.Errorf("the stale-declaration finding (R4.2) disappeared:\n%+v", findings)
		}
		if second, ok := declFinding(findings, "zed"); ok {
			t.Errorf("a stale declaration got a second, contradicting finding %q: one says the divergence is described, the other says it no longer exists", second.Detail)
		}
		// The reason must not be REPEATED to the operator. It is asserted on
		// Detail because Detail is the sentence that reaches them: `func
		// comparePackageFindings` in cmd/bentoo files a finding's Detail as the
		// row's REASON cell or as one of the package's notes, and reads no other
		// field of it.
		for _, f := range findings {
			if strings.Contains(f.Detail, reason) {
				t.Errorf("the report repeats the reason %q of a declaration it has just called stale — that text is precisely the part that is no longer true: %q", reason, f.Detail)
			}
		}

		// MEASURED, and recorded rather than asserted as a defect: the stale
		// finding CARRIES the reason as a value, on Effect{Text, EffectDeclared},
		// where the old rendered line did not. That is the finding holding the
		// facts, which is what a finding is for — the entry, the two versions and
		// the maintainer's sentence are all on it, and a consumer that wants to
		// show what the declaration USED to say can.
		//
		// The risk it creates is real and belongs beside the value: a renderer
		// that printed Effect.Text for every kind — a reasonable thing to write,
		// since Effect is documented as "what the divergence does" — would print,
		// under a heading saying the divergence no longer exists, the sentence
		// describing it. Nothing does that today. If one ever should, this is the
		// finding it must special-case.
		staleFinding := findingOfKind(findings, FindingStaleDeclaration)
		if staleFinding == nil {
			t.Fatal("no stale finding to check the carried reason on")
		}
		if staleFinding.Effect.Text != "" && staleFinding.Effect.Source != EffectDeclared {
			t.Errorf("the stale finding carries %+v; if it carries the superseded reason at all it must say whose sentence it is", staleFinding.Effect)
		}
	})

	t.Run("an unpatched package gains no line", func(t *testing.T) {
		clean := CompareResult{
			Category: "www-client", Package: "firefox",
			LocalVersion: "1.0", RemoteVersion: "1.0",
			Status: StatusUpToDate, Verdict: VerdictRedundant,
		}

		if found, ok := declFinding(declFindings(clean), "firefox"); ok {
			t.Errorf("an unpatched package got a declaration finding %q", found.Detail)
		}
	})

	t.Run("an over-long reason is capped while the entry stays whole", func(t *testing.T) {
		const longEntry = "app-office/libreoffice-l10n-meta@stable"
		long := patched
		long.PatchedBy = longEntry
		long.PatchedReason = strings.Repeat("x", patchedReasonCap*3)

		found, ok := declFinding(declFindings(long), "zed")
		if !ok {
			t.Fatal("no declaration was established")
		}
		if found.Entry != longEntry {
			t.Errorf("the declaring entry arrived as %q; a shortened key looks usable and is not, which is worse than carrying none", found.Entry)
		}

		// The CAP CHANGED DIRECTION, deliberately (story 047, sub-task 5.5,
		// S047-R8.2). It used to be applied here, on the way into the line;
		// Finding carries the reason AT FULL LENGTH because a value truncated on
		// the way into the report is truncated in the JSON export too, where there
		// is no width to respect and nothing to un-truncate it from. Deciding the
		// width is the RENDERER's, and internal/common/report wraps prose to the
		// device — pinned by TestCompareRunLongExplanationsStayInProse there and
		// visible in the wrapped leads of every plain golden.
		//
		// So the assertion is inverted rather than dropped: what must hold now is
		// that nothing here silently shortens the maintainer's sentence.
		if found.Effect.Text != long.PatchedReason {
			t.Errorf("the declared reason was shortened before it reached a consumer (%d of %d characters); a value cut here is cut in the JSON export too",
				len(found.Effect.Text), len(long.PatchedReason))
		}
		if strings.Contains(found.Effect.Text, "…") {
			t.Error("the finding carries an ellipsis; the cut is the renderer's to make and its mark is the renderer's to add")
		}
	})

	t.Run("a patched entry with no reason still names the entry", func(t *testing.T) {
		// Reachable in production: LoadPackagesConfig never calls
		// ValidatePackageConfig, so R1.3 does not protect this path and an entry
		// whose reason is whitespace can arrive here trimmed to "".
		silent := patched
		silent.PatchedReason = ""

		found, ok := declFinding(declFindings(silent), "zed")
		if !ok {
			t.Fatal("a reasonless declaration established nothing at all; the divergence is still declared")
		}
		if found.Entry != entry {
			t.Errorf("the finding names %q as its entry, want %q", found.Entry, entry)
		}
		// The dangling colon, at its source. The rendered line ended in ":" when
		// the reason was empty; the value form of the same defect is an
		// EffectDeclared over an empty sentence, which claims the maintainer said
		// something they did not. `func effect` returns the ZERO Effect instead,
		// and a renderer meeting it prints no colon because it has nothing to
		// introduce.
		if found.Effect != (Effect{}) {
			t.Errorf("a reasonless declaration carries %+v; a source label over an empty sentence claims the maintainer said something they did not", found.Effect)
		}
	})
}
