package chronicle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplayTrimsTornTail(t *testing.T) {
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

	// Append a half record to mimic a torn write.
	path := filepath.Join(dir, "000000001.data")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	// Write only part of a header, not a full record.
	if _, err := f.Write([]byte{0x01, 0x02, 0x03}); err != nil {
		t.Fatalf("write torn: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close torn: %v", err)
	}

	// Reopen should trim the torn tail and not fail.
	db2, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() {
		_ = db2.Close()
	}()

	// Previous good data should still be there.
	if _, err := db2.Get([]byte("a")); err != nil {
		t.Fatalf("want a after trim, got %v", err)
	}
	if _, err := db2.Get([]byte("b")); err != nil {
		t.Fatalf("want b after trim, got %v", err)
	}

	// New writes after trim should work.
	if err := db2.Put([]byte("c"), []byte("three")); err != nil {
		t.Fatalf("put c: %v", err)
	}
	if _, err := db2.Get([]byte("c")); err != nil {
		t.Fatalf("get c: %v", err)
	}
}
