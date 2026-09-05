package chronicle

import "testing"

func TestOperationsOnClosedStore(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if err := db.Put([]byte("k"), []byte("v")); err != ErrClosed {
		t.Fatalf("want ErrClosed on put after close, got %v", err)
	}
	if _, err := db.Get([]byte("k")); err != ErrClosed {
		t.Fatalf("want ErrClosed on get after close, got %v", err)
	}
	if _, err := db.Exists([]byte("k")); err != ErrClosed {
		t.Fatalf("want ErrClosed on exists after close, got %v", err)
	}
	if err := db.Fold(func(k, v []byte) error { return nil }); err != ErrClosed {
		t.Fatalf("want ErrClosed on fold after close, got %v", err)
	}
}
