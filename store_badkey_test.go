package chronicle

import "testing"

func TestBadKeyAndSizeLimits(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir, WithMaxKeySize(10), WithMaxValueSize(10))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Put(nil, []byte("v")); err != ErrBadKey {
		t.Fatalf("want ErrBadKey for nil key, got %v", err)
	}
	if err := db.Put([]byte(""), []byte("v")); err != ErrBadKey {
		t.Fatalf("want ErrBadKey for empty key, got %v", err)
	}
	if err := db.Put([]byte("toolongkeytoolong"), []byte("v")); err != ErrBadKey {
		t.Fatalf("want ErrBadKey for long key, got %v", err)
	}
	if err := db.Put([]byte("k"), make([]byte, 20)); err != ErrBadKey {
		t.Fatalf("want ErrBadKey for big value, got %v", err)
	}
	if _, err := db.Get([]byte("")); err != ErrBadKey {
		t.Fatalf("want ErrBadKey on get empty, got %v", err)
	}
	if _, err := db.Exists([]byte("")); err != ErrBadKey {
		t.Fatalf("want ErrBadKey on exists empty, got %v", err)
	}
	if err := db.Delete([]byte("")); err != ErrBadKey {
		t.Fatalf("want ErrBadKey on delete empty, got %v", err)
	}
}
