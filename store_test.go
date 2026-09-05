package chronicle

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
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
	defer func() { _ = db.Close() }()

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

func TestConcurrentPutGet(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			t.Fatalf("close: %v", cerr)
		}
	}()

	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)

		go func(n int) {
			defer wg.Done()

			key := fmt.Appendf(nil, "k-%d", n%10)
			val := fmt.Appendf(nil, "v-%d", n)

			if err := db.Put(key, val); err != nil {
				t.Errorf("put %d: %v", n, err)
				return
			}

			if _, err := db.Get(key); err != nil && err != ErrNotFound {
				t.Errorf("get %d: %v", n, err)
			}
		}(i)
	}

	wg.Wait()

	ok, err := db.Exists([]byte("k-1"))
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !ok {
		t.Errorf("expected to exist")
	}
}

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

func TestFoldWithConcurrentPuts(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	for i := range 10 {
		key := []byte(string(rune('a' + i)))
		if err := db.Put(key, []byte("v")); err != nil {
			t.Fatalf("seed put: %v", err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		for i := range 20 {
			_ = db.Put([]byte("new"), []byte("x"))
			_ = i
		}
	}()

	if err := db.Fold(func(k, v []byte) error { return nil }); err != nil {
		t.Fatalf("fold: %v", err)
	}

	wg.Wait()
}

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

	path := filepath.Join(dir, "000000001.data")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}

	if _, err := f.Write([]byte{0x01, 0x02, 0x03}); err != nil {
		t.Fatalf("write torn: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("close torn: %v", err)
	}

	db2, err := Open(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = db2.Close() }()

	if _, err := db2.Get([]byte("a")); err != nil {
		t.Fatalf("want a after trim, got %v", err)
	}

	if _, err := db2.Get([]byte("b")); err != nil {
		t.Fatalf("want b after trim, got %v", err)
	}

	if err := db2.Put([]byte("c"), []byte("three")); err != nil {
		t.Fatalf("put c: %v", err)
	}

	if _, err := db2.Get([]byte("c")); err != nil {
		t.Fatalf("get c: %v", err)
	}
}

func TestSyncOption(t *testing.T) {
	dir := t.TempDir()

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

	db2, err := Open(dir, WithSyncOnWrite(false))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = db2.Close() }()

	if _, err := db2.Get([]byte("k")); err != nil {
		t.Fatalf("get after reopen: %v", err)
	}
}

func TestManualSync(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir, WithSyncOnWrite(false))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Put([]byte("k"), []byte("v")); err != nil {
		t.Fatalf("put: %v", err)
	}

	if err := db.Sync(); err != nil {
		t.Fatalf("sync: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if err := db.Sync(); err != ErrClosed {
		t.Fatalf("want ErrClosed after close, got %v", err)
	}
}
