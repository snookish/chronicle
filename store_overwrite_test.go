package chronicle

import "testing"

func TestPutOverwritesValue(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	key := []byte("key")
	if err := db.Put(key, []byte("first")); err != nil {
		t.Fatalf("put first: %v", err)
	}
	if err := db.Put(key, []byte("second")); err != nil {
		t.Fatalf("put second: %v", err)
	}

	got, err := db.Get(key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got) != "second" {
		t.Fatalf("want second, got %q", got)
	}

	// Also after reopen should still be the last value.
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	db2, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = db2.Close() }()

	got2, err := db2.Get(key)
	if err != nil {
		t.Fatalf("get after reopen: %v", err)
	}
	if string(got2) != "second" {
		t.Fatalf("want second after reopen, got %q", got2)
	}
}
