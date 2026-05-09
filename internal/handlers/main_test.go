package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMain(m *testing.M) {
	// Change to the repo root so templates/*.html resolve correctly.
	// runtime.Caller(0) is resolved at compile time, giving us an absolute path.
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "runtime.Caller failed")
		os.Exit(1)
	}
	// filename = .../internal/handlers/main_test.go  →  repo root is two dirs up
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	if err := os.Chdir(repoRoot); err != nil {
		fmt.Fprintf(os.Stderr, "failed to chdir to %s: %v\n", repoRoot, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
