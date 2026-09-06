package main

// Authored for story 046 — the fixture the harness-driven tests share.
//
// It lives in its own file for the reason capture_stdout_findings_test.go in
// internal/overlay does: five sub-tasks land at different times and each of
// their test files has to compile on its own, so a fixture defined in whichever
// landed first would make the rest depend on the order they were written in.
//
// Materialize this file with the FIRST harness-driven sub-task (4.3); the ones
// after it use the same two helpers.
//
// writeExitTestEbuild and writeExitTestPackagesConfig are this package's own,
// from overlay_autoupdate_test.go — the hermetic check fixture story 043 built:
// an httptest server answering {"version": ...} and a registry pointing every
// declared package at it. Nothing here reaches the network.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// upstreamServer answers every request with one version, so a check run has a
// real upstream to compare against without leaving the machine.
func upstreamServer(t *testing.T, version string) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": version})
	}))
	t.Cleanup(server.Close)

	return server.URL
}

// alreadyCurrentPackage is the one package seedCheckOverlay seeds AT the
// upstream version, so every seeded overlay holds at least one package the
// check finds up to date.
//
// It is named after what it is for. A caller that passed this name itself would
// write the key twice into packages.toml — a duplicate-key error that stops the
// whole registry from loading, and would surface as "the check found nothing"
// rather than as the collision it is; seedCheckOverlay refuses that up front.
const alreadyCurrentPackage = "app-misc/already-current"

// seedCheckOverlay makes the harness's overlay into one a check run can work
// over: an ebuild per named package at the CURRENT version, plus one more at
// the UPSTREAM version, and a registry pointing every one of them at an
// upstream that answers with that upstream version.
//
// # Why the two versions differ
//
// A check whose upstream matches the overlay finds nothing to plan, and a
// report with an empty plan cannot show whether the export carried every unit.
// Every package the caller NAMES is therefore behind its upstream.
//
// # Why one package is seeded up to date anyway, unasked
//
// --all is the flag that lists the packages a run found UP TO DATE instead of
// counting them (report.SectionOptions.ShowAll). An overlay where every package
// is behind has no such packages, so the version-check section reads identically
// with the flag and without it — and a test comparing an export taken with --all
// against one taken without would then be comparing two runs the flag never
// touched. It would pass over any export, including a broken one.
//
// The extra package is what gives the flag something to change. It is seeded IN
// ADDITION to the caller's list rather than by promoting one of the caller's own
// packages, so `seedCheckOverlay(t, c, "1.0.0", "2.0.0", "app-misc/jq")` still
// means exactly what it says — jq, behind, with an update pending — for the
// callers that assert on it.
func seedCheckOverlay(t *testing.T, c *testCLI, currentVersion, upstreamVersion string, pkgs ...string) {
	t.Helper()

	overlay := c.Overlay()
	for _, pkg := range pkgs {
		if pkg == alreadyCurrentPackage {
			t.Fatalf("seedCheckOverlay: %s is the fixture's own up-to-date package and cannot also be named by the caller — packages.toml would carry the key twice and fail to load", pkg)
		}
		writeExitTestEbuild(t, overlay, pkg, currentVersion)
	}

	// At the UPSTREAM version, which is what makes this one up to date: the
	// checker compares the ebuild in the overlay against what the server
	// answers, and here they agree.
	writeExitTestEbuild(t, overlay, alreadyCurrentPackage, upstreamVersion)

	seeded := append(append([]string{}, pkgs...), alreadyCurrentPackage)
	writeExitTestPackagesConfig(t, overlay, upstreamServer(t, upstreamVersion), seeded)
}
