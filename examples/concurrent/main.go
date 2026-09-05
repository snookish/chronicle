package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/snookish/chronicle"
)

func main() {
	dir, err := os.MkdirTemp("", "chronicle-concurrent-*")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	db, err := chronicle.Open(dir)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	// Put from a few goroutines at once.
	var wg sync.WaitGroup
	for i := range 5 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Appendf(nil, "k-%d", n)
			val := fmt.Appendf(nil, "v-%d", n)
			if err := db.Put(key, val); err != nil {
				log.Printf("put %d: %v", n, err)
			}
		}(i)
	}
	wg.Wait()

	// Fold should see whatever got written without racing.
	fmt.Println("keys after concurrent puts:")
	if err := db.Fold(func(k, v []byte) error {
		fmt.Printf("  %s -> %s\n", k, v)
		return nil
	}); err != nil {
		log.Fatal(err)
	}
}
