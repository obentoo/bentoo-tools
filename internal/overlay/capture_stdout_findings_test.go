package overlay

// Authored for story 046, sub-tasks 7.1 and 7.2 — the one helper both need.
//
// It lives in its own file because two sub-tasks land at different times and
// each of their test files has to compile on its own; a helper defined in
// whichever landed first would make the second depend on the first.

import (
	"io"
	"os"
	"testing"
)

// captureOverlayStdout redirects os.Stdout for the duration of fn and returns
// everything written to it.
//
// The name carries the package because cmd/bentoo has a captureStdout of its
// own and the two are read side by side in review.
func captureOverlayStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating the capture pipe: %v", err)
	}

	original := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = original })

	captured := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(r)
		captured <- string(data)
	}()

	fn()

	_ = w.Close()
	os.Stdout = original
	out := <-captured
	_ = r.Close()
	return out
}
