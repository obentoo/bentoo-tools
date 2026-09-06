package report

// Authored for story 046, sub-task 1.1 — R7.1, R7.4.
//
// Written from the contract: design.md D2 fixes the shape Section moves into
// the model with —
//
//	type Section struct { Title string; Lead []string; Rows Table; Notes []string }
//
// and the Data Models block adds Table{Headers} and the rows beneath it. The
// two names this file assumes beyond that block are Row (one record: its cells
// and its own detail, the shape render.row already has) and Table.Columns (the
// "a Table reports its column count" scenario needs a name to ask through). If
// the implementer spells either differently the rename is mechanical; what is
// specified here is the BEHAVIOUR, not the spelling.
//
// Red on arrival: report.Section, report.Table and report.Row do not exist yet
// — they are unexported types inside render.

import (
	"strings"
	"testing"
)

// TestSectionWithNoRowsKeepsItsLead pins the case story 044 built the section
// vocabulary for and this story must not lose: "nothing to validate" is said by
// a section that has a lead and no rows, not by an empty frame and not by
// silence (R2.3).
//
// A Section that dropped its lead once its table was empty would make an
// omission indistinguishable from a section that was never produced.
func TestSectionWithNoRowsKeepsItsLead(t *testing.T) {
	s := Section{
		Title: "Validation Plan",
		Lead:  []string{"No pending update to validate"},
	}

	if len(s.Rows.Rows) != 0 {
		t.Fatalf("the fixture is meant to have no rows, it has %d", len(s.Rows.Rows))
	}
	if len(s.Lead) != 1 || s.Lead[0] != "No pending update to validate" {
		t.Errorf("Lead = %q, want the single sentence the section was built with — a rowless section says its lead or says nothing", s.Lead)
	}
	if s.Title != "Validation Plan" {
		t.Errorf("Title = %q, want %q", s.Title, "Validation Plan")
	}
}

// TestTableColumnCount pins what a renderer has to ask before it can size
// anything: how many columns are there.
//
// The hostile case is the second one. A Table whose rows carry MORE cells than
// its headers name is the shape that loses data silently: a count taken from
// the headers alone drops the extra cell off the right-hand edge of every mode
// at once, and no golden file would show it because the value was never
// rendered in the first place. The count is therefore the widest thing the
// table holds, header row included.
func TestTableColumnCount(t *testing.T) {
	cases := map[string]struct {
		table Table
		want  int
	}{
		"headers alone": {
			table: Table{Headers: []string{"package", "version", "outcome"}},
			want:  3,
		},
		"a row wider than the headers": {
			// The hostile fixture: counting the headers alone answers 2 and
			// loses the third cell of every row.
			table: Table{
				Headers: []string{"package", "version"},
				Rows:    []Row{{Cells: []string{"app-misc/jq", "1.8.0", "proved"}}},
			},
			want: 3,
		},
		"a row narrower than the headers": {
			table: Table{
				Headers: []string{"package", "version", "outcome"},
				Rows:    []Row{{Cells: []string{"app-misc/jq"}}},
			},
			want: 3,
		},
		"rows with no headers": {
			table: Table{Rows: []Row{{Cells: []string{"a", "b"}}}},
			want:  2,
		},
		"nothing at all": {
			table: Table{},
			want:  0,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := tc.table.Columns(); got != tc.want {
				t.Errorf("Columns() = %d, want %d — a column count short of the widest row drops that row's last cell in every renderer at once", got, tc.want)
			}
		})
	}
}

// TestSectionNotesAreCarriedWholeAndInOrder pins R2.3 at the model level: what
// a section left out, and why, is a value the section carries — in the order it
// was established, at full length, with nothing formatted into it.
//
// The escape check is not decoration. Notes are the field most likely to be
// written by a producer that used to print, so a note arriving with a colour
// already chosen is exactly how presentation re-enters the model (R7.1).
func TestSectionNotesAreCarriedWholeAndInOrder(t *testing.T) {
	const long = "5 packages found up to date are not listed; re-run with --all to see them, " +
		"and the count is the default because listing them is most of why a check that found four updates printed 348 lines"

	s := Section{
		Title: "Version Check Results",
		Notes: []string{long, "2 packages could not be typed"},
	}

	if len(s.Notes) != 2 {
		t.Fatalf("Notes = %d entries, want 2", len(s.Notes))
	}
	if s.Notes[0] != long {
		t.Errorf("the first note came back changed:\n got: %q\nwant: %q", s.Notes[0], long)
	}
	if s.Notes[1] != "2 packages could not be typed" {
		t.Errorf("the notes are out of order: %q", s.Notes)
	}
	for _, note := range s.Notes {
		if strings.ContainsRune(note, 0x1b) {
			t.Errorf("a note carries an escape sequence: %q — the model states what was omitted, it does not colour it (R7.1)", note)
		}
	}
}
