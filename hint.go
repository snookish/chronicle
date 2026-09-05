package chronicle

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
)

// hintFilePath is next to the data file.
func hintFilePath(dir string) string {
	return filepath.Join(dir, "000000001.hint")
}

// writeHint writes the current index to a temp file and moves it in place.
func writeHint(dir string, index *keydir) error {
	tmp := hintFilePath(dir) + ".tmp"

	file, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create hint tmp: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	snap := index.snapshot()

	for key, ent := range snap {
		// layout: key size (4) | value size not needed | offset (8) | fileID (8) | key bytes | crc (4)
		// keep it simple: key size + offset + size + key + crc
		keyBytes := []byte(key)

		// Prepare header for crc: key size + offset + size + key
		buf := make([]byte, 4+8+4+len(keyBytes))

		binary.LittleEndian.PutUint32(buf[0:4], uint32(len(keyBytes)))
		binary.LittleEndian.PutUint64(buf[4:12], uint64(ent.offset))
		binary.LittleEndian.PutUint32(buf[12:16], uint32(ent.size))
		copy(buf[16:], keyBytes)

		crc := crc32.ChecksumIEEE(buf)

		if _, err := file.Write(buf); err != nil {
			return fmt.Errorf("write hint: %w", err)
		}

		var crcBuf [4]byte
		binary.LittleEndian.PutUint32(crcBuf[:], crc)

		if _, err := file.Write(crcBuf[:]); err != nil {
			return fmt.Errorf("write hint crc: %w", err)
		}
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync hint tmp: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close hint tmp: %w", err)
	}

	if err := os.Rename(tmp, hintFilePath(dir)); err != nil {
		return fmt.Errorf("rename hint: %w", err)
	}

	if err := syncDir(dir); err != nil {
		return fmt.Errorf("sync dir after hint: %w", err)
	}

	return nil
}

// readHint tries to load the index from the hint file.
// If the hint is missing or bad we return false so caller can fall back to scanning the data file.
func readHint(dir string, index *keydir) (bool, error) {
	path := hintFilePath(dir)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read hint: %w", err)
	}

	offset := 0

	for offset < len(data) {
		if len(data[offset:]) < 4+8+4+4 {
			return false, nil // truncated, treat as bad
		}

		keySize := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		fileOffset := int64(binary.LittleEndian.Uint64(data[offset+4 : offset+12]))
		size := int32(binary.LittleEndian.Uint32(data[offset+12 : offset+16]))

		if keySize < 0 || len(data[offset+16:]) < keySize+4 {
			return false, nil
		}

		key := string(data[offset+16 : offset+16+keySize])
		wantCRC := binary.LittleEndian.Uint32(data[offset+16+keySize : offset+16+keySize+4])

		// Verify crc over header+key part.
		buf := data[offset : offset+16+keySize]
		gotCRC := crc32.ChecksumIEEE(buf)
		if gotCRC != wantCRC {
			return false, nil
		}

		index.put(key, entry{
			fileID: 1,
			offset: fileOffset,
			size:   size,
		})

		offset += 16 + keySize + 4
	}

	return true, nil
}
