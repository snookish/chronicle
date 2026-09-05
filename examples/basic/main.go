package main

import (
	"fmt"
	"log"
	"os"

	"github.com/snookish/chronicle"
)

func main() {
	dir, err := os.MkdirTemp("", "chronicle-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	// Open with small limits just for the example.
	db, err := chronicle.Open(dir)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	// Put a few keys.
	if err := db.Put([]byte("hello"), []byte("world")); err != nil {
		log.Fatal(err)
	}
	if err := db.Put([]byte("foo"), []byte("bar")); err != nil {
		log.Fatal(err)
	}

	// Get one back.
	val, err := db.Get([]byte("hello"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("hello -> %s\n", val)

	// Check existence without reading the value.
	ok, err := db.Exists([]byte("foo"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("foo exists? %v\n", ok)

	// Walk all live keys.
	fmt.Println("all keys:")
	if err := db.Fold(func(k, v []byte) error {
		fmt.Printf("  %s -> %s\n", k, v)
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	// Delete and confirm it's gone.
	if err := db.Delete([]byte("hello")); err != nil {
		log.Fatal(err)
	}
	if _, err := db.Get([]byte("hello")); err != nil {
		fmt.Printf("hello after delete: %v\n", err)
	}
}
