package main

// Authored for story 046, sub-task 4.1 — R3.1, R8.4.
//
// Written from the contract: 4.1's objective — "A fresh, fully-wired command
// tree can be built on demand instead of existing once per process" — and the
// risk the task group states: "the command tree is a package-level global, so
// an in-process harness must rebuild it per run or leak flag state between
// tests."
//
// The name assumed: newRootCmd() *cobra.Command, in root.go.
//
// Red on arrival: newRootCmd does not exist.
//
// # What "independent" has to mean here, and why the second test is not a
// # restatement of the first
//
// rootCmd is a package-level global today, and so are overlayCmd, snapshotCmd,
// versionCmd and completionCmd — each registered onto the root by an init() in
// its own file. A newRootCmd that returns a fresh *cobra.Command and then hangs
// the SAME package-level overlayCmd off it satisfies "two calls return
// different pointers" completely, and leaks every flag under `overlay` between
// runs. That is the failure this file exists to catch, and it is the shape a
// first implementation naturally takes.

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// commandNamed finds a direct sub-command by its first Use word.
func commandNamed(parent *cobra.Command, name string) *cobra.Command {
	for _, cmd := range parent.Commands() {
		if strings.Fields(cmd.Use)[0] == name {
			return cmd
		}
	}
	return nil
}

// TestNewRootCmdReturnsIndependentTrees pins the root itself: two calls, two
// trees, and a flag set on one invisible to the other.
func TestNewRootCmdReturnsIndependentTrees(t *testing.T) {
	first, second := newRootCmd(), newRootCmd()

	if first == second {
		t.Fatal("newRootCmd() returned the same *cobra.Command twice — a tree shared between runs is the flag state this sub-task exists to stop leaking")
	}

	flag := first.PersistentFlags().Lookup("verbose")
	if flag == nil {
		t.Fatal("the built tree has no persistent --verbose flag — see TestNewRootCmdCarriesTheProductionShape")
	}
	if err := flag.Value.Set("true"); err != nil {
		t.Fatalf("setting --verbose on the first tree: %v", err)
	}
	first.PersistentFlags().Lookup("verbose").Changed = true

	if got := second.PersistentFlags().Lookup("verbose"); got == nil {
		t.Fatal("the second tree has no persistent --verbose flag")
	} else if got.Value.String() != "false" || got.Changed {
		t.Errorf("--verbose on the second tree is %q (changed=%v) after being set on the first — the two trees share flag state",
			got.Value.String(), got.Changed)
	}
}

// TestNewRootCmdSubcommandsAreIndependentToo is the assertion the pointer check
// above cannot make.
//
// `overlay`, `snapshot`, `version` and `completion` are package-level globals
// registered by init(). A newRootCmd that builds a fresh root and re-registers
// those same globals returns two roots that share every sub-command — so
// `overlay manifest --distdir=/tmp` in one test is still set in the next,
// which is precisely the leak, one level down from where the first test looks.
func TestNewRootCmdSubcommandsAreIndependentToo(t *testing.T) {
	first, second := newRootCmd(), newRootCmd()

	for _, name := range []string{"overlay", "snapshot", "version", "completion"} {
		a, b := commandNamed(first, name), commandNamed(second, name)
		if a == nil || b == nil {
			t.Errorf("%q is missing from one of the two trees (first=%v, second=%v)", name, a != nil, b != nil)
			continue
		}
		if a == b {
			t.Errorf("both trees hang the SAME %q command off them — a sub-command shared between runs carries its flags across, which is the leak one level below the root", name)
		}
	}

	// One level deeper again: the sub-commands OF a sub-command.
	overlayFirst, overlaySecond := commandNamed(first, "overlay"), commandNamed(second, "overlay")
	if overlayFirst == nil || overlaySecond == nil {
		t.Fatal("no overlay command to descend into")
	}
	if manifestFirst, manifestSecond := commandNamed(overlayFirst, "manifest"), commandNamed(overlaySecond, "manifest"); manifestFirst != nil && manifestFirst == manifestSecond {
		t.Error("both trees share one `overlay manifest` command — the flags of the command this story is about would carry between runs")
	}
}

// TestNewRootCmdCarriesTheProductionShape pins what the tree must CONTAIN, so
// that "constructible" does not quietly become "constructible and missing half
// the commands". A harness running against a smaller tree would report a
// command as broken when it was merely absent.
//
// The expected sets are written out rather than compared against the package
// globals: 4.1 may retire those globals, and a test that pinned them would have
// to be deleted by the change it is meant to check.
func TestNewRootCmdCarriesTheProductionShape(t *testing.T) {
	root := newRootCmd()

	if got := strings.Fields(root.Use)[0]; got != "bentoo" {
		t.Errorf("root command Use = %q, want bentoo", got)
	}

	for _, name := range []string{"overlay", "snapshot", "version", "completion"} {
		if commandNamed(root, name) == nil {
			t.Errorf("the built tree has no %q command", name)
		}
	}

	for _, flag := range []string{"verbose", "quiet", "no-color"} {
		if root.PersistentFlags().Lookup(flag) == nil {
			t.Errorf("the built tree has no persistent --%s flag — main.go declares it on the root today", flag)
		}
	}

	overlay := commandNamed(root, "overlay")
	if overlay == nil {
		t.Fatal("no overlay command")
	}
	for _, name := range []string{"add", "status", "commit", "push", "manifest", "validate", "autoupdate"} {
		if commandNamed(overlay, name) == nil {
			t.Errorf("the built tree has no `overlay %s`", name)
		}
	}
}
