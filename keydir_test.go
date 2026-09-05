package chronicle

import (
	"sync"
	"testing"
)

func TestKeydirBasic(t *testing.T) {
	kd := newKeydir()
	kd.put("a", entry{fileID: 1, offset: 10, size: 5, timestamp: 100})
	e, ok := kd.get("a")
	if !ok || e.offset != 10 {
		t.Fatalf("want entry, got %v %v", e, ok)
	}
	kd.delete("a")
	if _, ok := kd.get("a"); ok {
		t.Fatal("should be deleted")
	}
}

func TestKeydirSnapshotIsolated(t *testing.T) {
	kd := newKeydir()
	kd.put("a", entry{fileID: 1, timestamp: 1})
	snap := kd.snapshot()
	kd.put("b", entry{fileID: 1, timestamp: 2})
	if _, ok := snap["b"]; ok {
		t.Fatal("snapshot should not see new keys")
	}
}

func TestKeydirConcurrent(t *testing.T) {
	kd := newKeydir()
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			kd.put("k", entry{fileID: int64(n)})
			_, _ = kd.get("k")
		}(i)
	}
	wg.Wait()
}
