package chronicle

import "testing"

func TestDeletePersistsAfterReopen(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Put([]byte("k"), []byte("v")); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := db.Delete([]byte("k")); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	db2, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = db2.Close() }()

	if _, err := db2.Get([]byte("k")); err != ErrNotFound {
		t.Fatalf("want ErrNotFound after reopen, got %v", err)
	}
	if ok, err := db2.Exists([]byte("k")); err != nil {
		t.Fatalf("exists: %v", err)
	} else if ok {
		t.Fatal("should not exist after delete and reopen")
	}
}
