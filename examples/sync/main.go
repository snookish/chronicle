package main

import (
	"fmt"
	"log"
	"os"

	"github.com/snookish/chronicle"
)

func main() {
	dir, err := os.MkdirTemp("", "chronicle-sync-*")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	// Open with sync off, then flush by hand when we want.
	db, err := chronicle.Open(dir, chronicle.WithSyncOnWrite(false))
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Put([]byte("a"), []byte("one")); err != nil {
		log.Fatal(err)
	}
	if err := db.Put([]byte("b"), []byte("two")); err != nil {
		log.Fatal(err)
	}

	// Make sure data is on disk.
	if err := db.Sync(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("wrote a and b, synced")

	if err := db.Close(); err != nil {
		log.Fatal(err)
	}

	// Reopen should see the same data.
	db2, err := chronicle.Open(dir)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = db2.Close()
	}()

	val, err := db2.Get([]byte("a"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("a -> %s\n", val)
}
