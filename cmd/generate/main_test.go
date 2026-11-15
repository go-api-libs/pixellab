package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestRun makes the specification and the library from PixelLab's specification and the recordings committed,
// offline, as go generate does, and finds them as committed: what the repository holds is what the generator makes
// of what it holds, no file of it edited by hand or left behind.
func TestRun(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	tmp := t.TempDir()

	for _, name := range []string{originalPath, interactionsPath} {
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}

		if err := os.MkdirAll(filepath.Join(tmp, filepath.Dir(name)), 0o755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(tmp, name), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.MkdirAll(filepath.Join(tmp, "pkg", "pixellab"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(tmp)

	isClaudeCode = true // read the specification committed rather than fetch it

	if err := run(t.Context()); err != nil {
		t.Fatal(err)
	}

	generated, err := filepath.Glob("pkg/pixellab/*")
	if err != nil {
		t.Fatal(err)
	}

	if len(generated) == 0 {
		t.Fatal("nothing generated")
	}

	for _, name := range append(generated, specPath, interactionsPath) {
		got, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}

		want, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Errorf("%s is generated but not committed: %v", name, err)
			continue
		}

		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from what the generator makes: run go generate", name)
		}
	}
}
