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

	// If the file is new we need the directory to be durable.
	var isNew bool
	if _, serr := os.Stat(path); serr != nil && os.IsNotExist(serr) {
		isNew = true
	}

	file, offset, err := openDataFile(path)
	if err != nil {
		return nil, err
	}

	if isNew {
		if err := syncDir(directory); err != nil {
			_ = file.Close()
			return nil, err
		}
	}

	db := &DB{
		directory:  directory,
		config:     cfg,
		index:      newKeydir(),
		activeFile: file,
		nextOffset: offset,
	}

	if err := db.replay(); err != nil {
		if cerr := file.Close(); cerr != nil {
			return nil, fmt.Errorf("close after replay failed: %w", cerr)
		}
		return nil, err
	}

	return db, nil
}

// replay rebuilds the index from the file.
// If the last record is torn or bad we cut it off so the file stays clean.
func (db *DB) replay() error {
	path := dataFilePath(db.directory)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read data file: %w", err)
	}

	var offset int64
	for offset < int64(len(data)) {
		header, key, _, size, err := Unmarshal(data[offset:])
		if err != nil {
			// Check if header itself claims a huge size, treat as torn tail.
			if err == ErrTooSmall || err == ErrCorrupt {
				if err := os.Truncate(path, offset); err != nil {
					return fmt.Errorf("truncate torn tail at %d: %w", offset, err)
				}
				db.nextOffset = offset
				if _, serr := db.activeFile.Seek(offset, 0); serr != nil {
					return fmt.Errorf("seek after truncate: %w", serr)
				}
				break
			}
			return fmt.Errorf("replay failed at offset %d: %w", offset, err)
		}

		// Guard against absurd sizes that passed crc but exceed config.
		if int(header.KeySize) > db.config.maxKeyBytes || header.ValueLen() > db.config.maxValueBytes {
			if err := os.Truncate(path, offset); err != nil {
				return fmt.Errorf("truncate oversize record at %d: %w", offset, err)
			}
			db.nextOffset = offset
			if _, serr := db.activeFile.Seek(offset, 0); serr != nil {
				return fmt.Errorf("seek after truncate: %w", serr)
			}
			break
		}

		if header.IsTombstone {
			db.index.delete(string(key))
		} else {
			db.index.put(string(key), entry{
				fileID: 1,
				offset: offset,
				size:   int32(size),
			})
		}
		offset += int64(size)
	}

	db.nextOffset = offset
	return nil
}

// Put writes a key and its value.
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

// Delete writes a tombstone so the key goes away on replay too.
func (db *DB) Delete(key []byte) error {
	if len(key) == 0 || len(key) > db.config.maxKeyBytes {
		return ErrBadKey
	}

	db.mu.Lock()
	defer db.mu.Unlock()
	if db.isClosed {
		return ErrClosed
	}

	record, err := Marshal(nil, db.config.clock.Now(), key, nil, true)
	if err != nil {
		return fmt.Errorf("marshal tombstone: %w", err)
	}

	if _, err := db.activeFile.Write(record); err != nil {
		return fmt.Errorf("write tombstone: %w", err)
	}

	if db.config.syncOnWrite {
		if err := db.activeFile.Sync(); err != nil {
			return fmt.Errorf("sync tombstone: %w", err)
		}
	}

	db.nextOffset += int64(len(record))
	db.index.delete(string(key))
	return nil
}

// Exists checks if a key is present without reading the value.
func (db *DB) Exists(key []byte) (bool, error) {
	if len(key) == 0 || len(key) > db.config.maxKeyBytes {
		return false, ErrBadKey
	}

	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.isClosed {
		return false, ErrClosed
	}

	_, ok := db.index.get(string(key))
	return ok, nil
}

// Get gets the latest value for a key.
func (db *DB) Get(key []byte) ([]byte, error) {
	if len(key) == 0 || len(key) > db.config.maxKeyBytes {
		return nil, ErrBadKey
	}

	db.mu.RLock()
	if db.isClosed {
		db.mu.RUnlock()
		return nil, ErrClosed
	}

	entry, ok := db.index.get(string(key))
	db.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}

	path := dataFilePath(db.directory)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open data file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	buffer := make([]byte, entry.size)
	if _, err := file.ReadAt(buffer, entry.offset); err != nil {
		return nil, fmt.Errorf("read record: %w", err)
	}

	_, _, value, _, err := Unmarshal(buffer)
	if err != nil {
		return nil, fmt.Errorf("decode record: %w", err)
	}

	out := make([]byte, len(value))
	copy(out, value)
	return out, nil
}

// Fold walks every live key. The snapshot is taken up front so it doesn't block writes long.
func (db *DB) Fold(fn func(key, value []byte) error) error {
	if fn == nil {
		return nil
	}
	db.mu.RLock()
	if db.isClosed {
		db.mu.RUnlock()
		return ErrClosed
	}

	snapshot := db.index.snapshot()
	db.mu.RUnlock()

	path := dataFilePath(db.directory)
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open data file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	for k, ent := range snapshot {
		buffer := make([]byte, ent.size)
		if _, err := file.ReadAt(buffer, ent.offset); err != nil {
			return fmt.Errorf("read record for %q: %w", k, err)
		}

		_, _, value, _, err := Unmarshal(buffer)
		if err != nil {
			return fmt.Errorf("decode record for %q: %w", k, err)
		}

		if err := fn([]byte(k), value); err != nil {
			return err
		}
	}
	return nil
}

// Sync forces the file to disk. Useful when syncOnWrite is off.
func (db *DB) Sync() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.isClosed {
		return ErrClosed
	}

	if err := db.activeFile.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}
	return nil
}

// Close flushes and closes the file.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.isClosed {
		return ErrClosed
	}

	db.isClosed = true

	if err := db.activeFile.Sync(); err != nil {
		return fmt.Errorf("sync on close: %w", err)
	}

	if err := db.activeFile.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	return nil
}
