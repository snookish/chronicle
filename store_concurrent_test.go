package chronicle

import (
	"fmt"
	"sync"
	"testing"
)

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
		t.Errorf("expected ok to be true")
	}
}
