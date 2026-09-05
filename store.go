package chronicle

import (
	"fmt"
	"os"
	"sync"
)

// DB is the store. It appends records to a file and keeps an index in memory.
type DB struct {
	directory  string
	config     config
	index      *keydir
	mu         sync.RWMutex
	activeFile *os.File
	nextOffset int64
	isClosed   bool
}

// Open opens the store in the given directory.
func Open(directory string, opts ...Option) (*DB, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("bad config: %w", err)
	}

	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	path := dataFilePath(directory)
	file, offset, err := openDataFile(path)
	if err != nil {
		return nil, err
	}

	db := &DB{
		directory:  directory,
		config:     cfg,
		index:      newKeydir(),
		activeFile: file,
		nextOffset: offset,
	}

	return db, nil
}

// Put appends a key and value to the log.
func (db *DB) Put(key, value []byte) error {
	if len(key) == 0 || len(key) > db.config.maxKeyBytes {
		return ErrBadKey
	}

	if len(value) > db.config.maxValueBytes {
		return ErrBadKey
	}

	db.mu.Lock()
	defer db.mu.Unlock()
	if db.isClosed {
		return ErrClosed
	}

	record, err := Marshal(nil, db.config.clock.Now(), key, value, false)
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}

	offset := db.nextOffset
	if _, err := db.activeFile.Write(record); err != nil {
		return fmt.Errorf("write record: %w", err)
	}

	if db.config.syncOnWrite {
		if err := db.activeFile.Sync(); err != nil {
			return fmt.Errorf("sync file: %w", err)
		}
	}

	db.nextOffset += int64(len(record))
	db.index.put(string(key), entry{
		fileID: 1,
		offset: offset,
		size:   int32(len(record)),
	})

	return nil
}
