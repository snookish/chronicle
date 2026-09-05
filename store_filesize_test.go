package chronicle

import "testing"

func TestFileSizeGuard(t *testing.T) {
	dir := t.TempDir()

	// Tiny limit so second put would push over.
	db, err := Open(dir, WithMaxDataFileSize(100))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	// First put should fit.
	if err := db.Put([]byte("k1"), []byte("value-one")); err != nil {
		t.Fatalf("first put: %v", err)
	}

	// Second put should hit the guard.
	if err := db.Put([]byte("k2"), make([]byte, 80)); err == nil {
		t.Fatal("want error when file would get too large")
	}
}
