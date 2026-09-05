package chronicle

import (
	"bytes"
	"testing"
)

func TestPutGetCloseReopen(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	key := []byte("hello")
	val := []byte("world")

	if err := db.Put(key, val); err != nil {
		t.Fatalf("put: %v", err)
	}

	got, err := db.Get(key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !bytes.Equal(got, val) {
		t.Fatalf("got %q want %q", got, val)
	}

	ok, err := db.Exists(key)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !ok {
		t.Fatal("should exist")
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Reopen should rebuild index and still find data.
	db2, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() {
		if cerr := db2.Close(); cerr != nil {
			t.Fatalf("close2: %v", cerr)
		}
	}()

	got2, err := db2.Get(key)
	if err != nil {
		t.Fatalf("get after reopen: %v", err)
	}
	if !bytes.Equal(got2, val) {
		t.Fatalf("after reopen got %q want %q", got2, val)
	}
}

func TestDeleteGet(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	key := []byte("k1")
	if err := db.Put(key, []byte("v1")); err != nil {
		t.Fatalf("put: %v", err)
	}

	if err := db.Delete(key); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, err := db.Get(key); err != ErrNotFound {
		t.Fatalf("want ErrNotFound after delete, got %v", err)
	}

	ok, err := db.Exists(key)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if ok {
		t.Fatal("should not exist after delete")
	}
}

func TestCloseTwice(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}

	if err := db.Close(); err != ErrClosed {
		t.Fatalf("want ErrClosed second time, got %v", err)
	}
}
