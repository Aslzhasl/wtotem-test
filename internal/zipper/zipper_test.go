package zipper

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestZipDir_SkipsArtifacts(t *testing.T) {
	tmp := t.TempDir()
	// layout:
	//   a.txt
	//   report_foo.json (should skip)
	//   node_modules/x.js (skip dir)
	//   .git/config (skip)
	mustWrite(t, filepath.Join(tmp, "a.txt"), "hello")
	mustWrite(t, filepath.Join(tmp, "report_bar.json"), "{}")
	mustWrite(t, filepath.Join(tmp, "node_modules", "x.js"), "noop")
	mustWrite(t, filepath.Join(tmp, ".git", "config"), "noop")

	dst := filepath.Join(tmp, "source_code.zip")
	if err := New().ZipDir(tmp, dst); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.OpenReader(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}

	expectPresent(t, names, "a.txt")
	expectAbsent(t, names, "report_bar.json")
	expectAbsent(t, names, "node_modules/x.js")
	expectAbsent(t, names, ".git/config")
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func expectPresent(t *testing.T, list []string, want string) {
	t.Helper()
	for _, s := range list {
		if s == want {
			return
		}
	}
	t.Fatalf("expected %q in zip, got %v", want, list)
}

func expectAbsent(t *testing.T, list []string, not string) {
	t.Helper()
	for _, s := range list {
		if s == not {
			t.Fatalf("did not expect %q in zip, got %v", not, list)
		}
	}
}
