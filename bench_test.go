package chronicle

import (
	"fmt"
	"testing"
)

func BenchmarkPut(b *testing.B) {
	dir := b.TempDir()

	db, err := Open(dir, WithSyncOnWrite(false))
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	b.ResetTimer()

	for i := range b.N {
		key := fmt.Appendf(nil, "k-%d", i)
		val := []byte("value-data")

		if err := db.Put(key, val); err != nil {
			b.Fatalf("put: %v", err)
		}
	}
}

func BenchmarkGet(b *testing.B) {
	dir := b.TempDir()

	db, err := Open(dir, WithSyncOnWrite(false))
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	for i := range 1000 {
		key := fmt.Appendf(nil, "k-%d", i)
		if err := db.Put(key, []byte("value-data")); err != nil {
			b.Fatalf("seed put: %v", err)
		}
	}

	b.ResetTimer()

	for i := range b.N {
		key := fmt.Appendf(nil, "k-%d", i%1000)

		if _, err := db.Get(key); err != nil {
			b.Fatalf("get: %v", err)
		}
	}
}

func BenchmarkFold(b *testing.B) {
	dir := b.TempDir()

	db, err := Open(dir, WithSyncOnWrite(false))
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	defer func() { _ = db.Close() }()

	for i := range 500 {
		key := fmt.Appendf(nil, "k-%d", i)
		if err := db.Put(key, []byte("value-data")); err != nil {
			b.Fatalf("seed put: %v", err)
		}
	}

	for b.Loop() {
		if err := db.Fold(func(k, v []byte) error { return nil }); err != nil {
			b.Fatalf("fold: %v", err)
		}
	}
}
