package overlay

// Authored for story 047, sub-task 1.1 — S047-R3.1, S047-R4.1.
//
// The rule, S047-R3.1: "WHEN a comparison row is rendered THE SYSTEM SHALL
// state which of four reading states it is in: not requested, not comparable,
// attempted and failed, or read."
//
// Four states are only four states if no two of them can be told apart by
// nothing. This file asks the hostile question first — CAN TWO STATES COLLAPSE
// INTO ONE? — because the defect story 047 exists to remove is exactly a
// collapse: eleven redundant packages, six never compared and five whose review
// was killed, all rendering identically.
//
// RED ON ARRIVAL: Reading, ReadingNotRequested, ReadingNotComparable,
// ReadingFailed, ReadingDone and CompareResult.Reading do not exist.
//
// Every name carries the TestReading prefix.

import "testing"

// TestReadingStatesCannotCollapse is the hostile half, written first.
//
// It is the case that makes the rule fire WRONGLY: two constants spelling one
// value. `iota` makes them distinct for free, and that is precisely why nobody
// checks — the day one of them is given an explicit value ("= ReadingFailed",
// a merge that duplicated a line), the compiler says nothing and every renderer
// downstream reports two different facts under one answer.
func TestReadingStatesCannotCollapse(t *testing.T) {
	named := []struct {
		name  string
		state Reading
	}{
		{"ReadingNotRequested", ReadingNotRequested},
		{"ReadingNotComparable", ReadingNotComparable},
		{"ReadingFailed", ReadingFailed},
		{"ReadingDone", ReadingDone},
	}

	for i, a := range named {
		for j, b := range named {
			if i >= j {
				continue
			}
			if a.state == b.state {
				t.Errorf("%s and %s are both %d — two reading states spelling one value cannot be told apart, "+
					"which is the conflation S047-R3.1 exists to remove: \"nobody looked\" would render as \"we looked\"",
					a.name, b.name, a.state)
			}
		}
	}
}

// TestReadingUnexaminedResultHasOneSpelling is the converse hostile half: the
// rule wrongly SPLITTING one fact into two.
//
// Every result nobody examined must arrive at ReadingNotRequested without
// anybody setting it. If the zero value were anything else — a fifth "unset"
// state, or NotRequested placed second in the const block — then "nobody asked
// about this package" would have two spellings: the one the annotation pass
// writes and the one every untouched result carries. A consumer counting
// unread packages would then be right about some of them and silently wrong
// about the rest.
func TestReadingUnexaminedResultHasOneSpelling(t *testing.T) {
	var fresh Reading
	if fresh != ReadingNotRequested {
		t.Errorf("the zero Reading is %d, want ReadingNotRequested (%d): a result nobody examined must read as "+
			"\"not requested\" without anyone setting it (design D3)", fresh, ReadingNotRequested)
	}

	// The same claim where it is actually read: on a result the comparison
	// produced and no annotation pass reached.
	untouched := CompareResult{Category: "kde-plasma", Package: "kwin"}
	if untouched.Reading != ReadingNotRequested {
		t.Errorf("a CompareResult nobody annotated carries Reading %d, want ReadingNotRequested (%d)",
			untouched.Reading, ReadingNotRequested)
	}
}

// TestReadingIsSeparateFromVerificationAndReview is the third hostile shape:
// the state being derived from a field that already means something else.
//
// design.md D3 refuses two collapses by name. Verification says whether the two
// files DIFFER; Reading says whether anybody explained the difference — and the
// zero ReviewNote already covers four unrelated situations (compare.go's doc on
// CompareResult.Review), so "is Review empty?" cannot answer "did anybody read
// this?".
//
// The fixture below is the pair that proves it: one result was read and found
// nothing worth a note, another was never asked. Both carry the zero ReviewNote
// and the same Verification. Only Reading tells them apart.
func TestReadingIsSeparateFromVerificationAndReview(t *testing.T) {
	read := CompareResult{
		Category: "kde-plasma", Package: "spectacle",
		Verified: VerifiedIdentical,
		Reading:  ReadingDone,
	}
	neverAsked := CompareResult{
		Category: "kde-plasma", Package: "kwin",
		Verified: VerifiedIdentical,
	}

	if read.Verified != neverAsked.Verified {
		t.Fatal("the fixture is wrong: both results must carry the same Verification, or the assertion below proves nothing")
	}
	if read.Review != neverAsked.Review {
		t.Fatal("the fixture is wrong: both results must carry the zero ReviewNote, which is the overload D3 refuses to re-read")
	}
	if read.Reading == neverAsked.Reading {
		t.Error("a package that was read and one that was never asked carry the same Reading. " +
			"Verification and the zero ReviewNote cannot separate them — that is why Reading is a THIRD field (D3)")
	}
}
