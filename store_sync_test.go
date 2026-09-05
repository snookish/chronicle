package chronicle

import "testing"

func TestSyncOption(t *testing.T) {
	dir := t.TempDir()

	// Open with sync off, put should still work.
	db, err := Open(dir, WithSyncOnWrite(false))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := db.Put([]byte("k"), []byte("v")); err != nil {
		t.Fatalf("put: %v", err)
	}

	if _, err := db.Get([]byte("k")); err != nil {
		t.Fatalf("get: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Reopen should still see data even without per-put fsync.
	db2, err := Open(dir, WithSyncOnWrite(false))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = db2.Close() }()

	if _, err := db2.Get([]byte("k")); err != nil {
		t.Fatalf("get after reopen: %v", err)
	}
}
