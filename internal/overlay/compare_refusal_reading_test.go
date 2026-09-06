package overlay

// Authored for story 047, sub-task 8.2 — S047-R3.3, S047-R4.1.
//
// The rule under test:
//
//   - S047-R3.3: "WHERE a package was not compared because its two versions
//     differ THE SYSTEM SHALL say so."
//
// and the word that was missing from it: SHALL say so EVEN WHEN NOBODY REVIEWS.
//
// The defect this pins: ReadingNotComparable was written in exactly one place,
// the walk inside AnnotateReviews, which returns at its first statement when the
// reviewer is nil. `--no-review` is one way to have no reviewer; no `claude` on
// PATH is the other, and it is most CI. On such a machine the content check had
// still run and still refused pairs, and the run recorded none of it — so the
// same overlay reported six unestablished facts on a developer's laptop and
// zero on the machine that gates the merge.
//
// So this test NEVER CALLS AnnotateReviews. That absence is the whole assertion:
// the fixture goes through CompareWithProvider and nothing else, which is what a
// machine without the CLI does.
//
// Reused rather than restated: reviewFixture and resultFor come from
// review_test.go. net-libs/nodejs is that fixture's refused pair — 1.0 here,
// 2.0 upstream — and the four packages beneath it are every other outcome the
// fixture holds, so "only the refusal is marked" is asserted over the whole
// report rather than over one hand-picked row.

import "testing"

// TestCompareRefusalIsRecordedWithoutAnyReviewer is the hostile half: the state
// that used to depend on a CLI being installed.
func TestCompareRefusalIsRecordedWithoutAnyReviewer(t *testing.T) {
	// The comparison alone. No reviewer, and no annotate pass to stand in for
	// one — exactly the run a machine with no `claude` performs.
	report, _, _ := reviewFixture(t)

	refused := resultFor(t, report, "net-libs/nodejs")
	if refused.Verified != NotVerified {
		t.Fatalf("the fixture is wrong: net-libs/nodejs is Verified %d, and this case needs the pair the content check refused", refused.Verified)
	}
	if refused.Reading != ReadingNotComparable {
		t.Errorf("net-libs/nodejs carries Reading %d after CompareWithProvider, want ReadingNotComparable (%d). "+
			"The content check ran and refused this pair; recording that only inside AnnotateReviews loses it on every "+
			"machine without a reviewer, and the run then reports itself complete over a fact it never established (S047-R3.3)",
			refused.Reading, ReadingNotComparable)
	}
}

// TestCompareRefusalMarksOnlyWhatWasRefused is the benign half, and it is what
// keeps the fix above from being bought by marking everything.
//
// A reading of NotComparable on a package the check DID compare would count
// toward NotEvaluated and report a whole run as holed. Each atom below is a
// different way of not being a refusal: two undeclared divergences the check
// compared and found different, one declared divergence, one identical pair.
func TestCompareRefusalMarksOnlyWhatWasRefused(t *testing.T) {
	report, _, _ := reviewFixture(t)

	for _, atom := range []string{"kde-plasma/spectacle", "kde-plasma/kwin", "app-editors/zed", "dev-libs/libixion"} {
		r := resultFor(t, report, atom)
		if r.Verified == NotVerified {
			t.Fatalf("the fixture is wrong: %s is NotVerified, and this case needs packages the check could compare", atom)
		}
		if r.Reading != ReadingNotRequested {
			t.Errorf("%s carries Reading %d out of the comparison, want ReadingNotRequested (%d): the check compared this "+
				"package, so nothing about it was refused, and a run that asked for no review must leave it saying nobody asked (S047-R5.3)",
				atom, r.Reading, ReadingNotRequested)
		}
	}
}
