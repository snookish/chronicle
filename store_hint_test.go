package chronicle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHintFileWrittenAndUsed(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := db.Put([]byte("a"), []byte("one")); err != nil {
		t.Fatalf("put a: %v", err)
	}

	if err := db.Put([]byte("b"), []byte("two")); err != nil {
		t.Fatalf("put b: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	hintPath := filepath.Join(dir, "000000001.hint")
	if _, err := os.Stat(hintPath); err != nil {
		t.Fatalf("want hint file, got %v", err)
	}

	// Reopen should use hint and still see data.
	db2, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = db2.Close() }()

	if _, err := db2.Get([]byte("a")); err != nil {
		t.Fatalf("want a after hint, got %v", err)
	}

	if _, err := db2.Get([]byte("b")); err != nil {
		t.Fatalf("want b after hint, got %v", err)
	}
}

func TestHintFallbackIfCorrupt(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := db.Put([]byte("k"), []byte("v")); err != nil {
		t.Fatalf("put: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Corrupt the hint file.
	hintPath := filepath.Join(dir, "000000001.hint")
	if err := os.WriteFile(hintPath, []byte("bad data"), 0644); err != nil {
		t.Fatalf("corrupt hint: %v", err)
	}

	// Open should fall back to scanning the data file and still work.
	db2, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = db2.Close() }()

	if _, err := db2.Get([]byte("k")); err != nil {
		t.Fatalf("want k after corrupt hint fallback, got %v", err)
	}
}
