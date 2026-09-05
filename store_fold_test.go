package chronicle

import "testing"

func TestFoldWalksLiveKeys(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Put([]byte("a"), []byte("one")); err != nil {
		t.Fatalf("put a: %v", err)
	}
	if err := db.Put([]byte("b"), []byte("two")); err != nil {
		t.Fatalf("put b: %v", err)
	}
	if err := db.Delete([]byte("a")); err != nil {
		t.Fatalf("delete a: %v", err)
	}

	seen := make(map[string]string)
	if err := db.Fold(func(k, v []byte) error {
		seen[string(k)] = string(v)
		return nil
	}); err != nil {
		t.Fatalf("fold: %v", err)
	}

	if len(seen) != 1 {
		t.Fatalf("want 1 key, got %d", len(seen))
	}
	if seen["b"] != "two" {
		t.Fatalf("want b=two, got %q", seen["b"])
	}
	if _, ok := seen["a"]; ok {
		t.Fatal("deleted key a should not be in fold")
	}
}
