package main

import "github.com/obentoo/bentoolkit/internal/common/report"

// exportFixture is one small but COMPLETE report: a package that was scanned
// and found behind, a package that was scanned and found current, the plan for
// the first, its verdict, and a tally that reconciles against the plan.
//
// Both halves matter to what the export tests measure. The up-to-date package
// is there because an export lists every scanned package regardless of --all
// (R9.3), so a fixture holding only the interesting one could not tell a
// renderer that honours the flag from one that ignores it. The long reason is
// there because "nothing was shortened" is only an assertion if something in the
// report was long enough to be worth shortening.
func exportFixture() report.AutoupdateCheck {
	const reason = "a minor bump earns the configure rung from the default depth policy, and the policy is what the plan line has to be checkable against"

	return report.AutoupdateCheck{
		Scanned: []report.PackageResult{
			{
				Package:          "app-misc/jq",
				Type:             "source",
				CurrentVersion:   "1.7.1",
				CandidateVersion: "1.8.0",
				HasUpdate:        true,
			},
			{
				Package:          "app-shells/fish",
				Type:             "source",
				CurrentVersion:   "4.0.2",
				CandidateVersion: "4.0.2",
			},
		},
		Plan: []report.PlanEntry{
			{
				Package:          "app-misc/jq",
				CurrentVersion:   "1.7.1",
				CandidateVersion: "1.8.0",
				Class:            "minor",
				Depth:            "configure",
				Reason:           reason,
			},
		},
		Results: []report.ValidationRow{
			{
				Package:          "app-misc/jq",
				CandidateVersion: "1.8.0",
				Outcome:          report.Proved,
				Depth:            "configure",
				Reason:           reason,
				SameReasonAsPlan: true,
			},
		},
		Tally: report.Tally{Proved: 1},
	}
}

// exportedFixtureRun is that report in the envelope a run which reached the end
// of its plan carries. Complete and NotEvaluated are the ENVELOPE's since
// sub-task 3.1, so the fixture states completeness here instead of inside the
// report — the same claim about the same run, made where the fact now lives.
//
// Since sub-task 3.2 the envelope is built in one place and this goes through
// it (finishedRun, overlay_autoupdate_check_test.go), so the fixture cannot
// drift from a real run's schema, kind or title.
func exportedFixtureRun() report.Run {
	return finishedRun(exportFixture())
}
