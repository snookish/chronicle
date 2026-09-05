package chronicle

import "testing"

func TestPutAfterDelete(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	key := []byte("k")
	if err := db.Put(key, []byte("first")); err != nil {
		t.Fatalf("put first: %v", err)
	}
	if err := db.Delete(key); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := db.Put(key, []byte("second")); err != nil {
		t.Fatalf("put after delete: %v", err)
	}

	got, err := db.Get(key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got) != "second" {
		t.Fatalf("want second, got %q", got)
	}

	// Also survives reopen.
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
