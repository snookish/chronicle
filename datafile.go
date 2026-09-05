package chronicle

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// dataFilePath is where the single active file lives.
func dataFilePath(dir string) string {
	return filepath.Join(dir, "000000001.data")
}

// openDataFile opens the file and returns it plus where to append next.
func openDataFile(path string) (*os.File, int64, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, 0, fmt.Errorf("open data file: %w", err)
	}
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return nil, 0, errors.Join(
				fmt.Errorf("seek end: %w", err),
				fmt.Errorf("close after seek failed: %w", closeErr),
			)
		}
		return nil, 0, fmt.Errorf("seek end: %w", err)
	}
	return file, offset, nil
}
