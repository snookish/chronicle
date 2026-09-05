package chronicle

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
)

// HeaderSize is the size of the record header on disk.
// 20B little-endian header: CRC32(4) | Tstamp nanos int64(8) | Ksz u32(4) | Vsz u32(4).
// CRC covers header[4:] (tstamp|ksz|vsz) + key + value, excludes itself.
const (
	HeaderSize    = 20
	tombstoneMask = uint32(1 << 31)
	maxValueLen   = ^tombstoneMask
)

var (
	ErrCorrupt   = errors.New("record: crc mismatch")
	ErrKeyTooBig = errors.New("record: key too large")
	ErrTooSmall  = errors.New("record: too small for header")
)

// Header is the 20 byte header at the start of each record.
type Header struct {
	CRC         uint32
	Timestamp   int64
	KeySize     uint32
	ValueSize   uint32
	IsTombstone bool
}

// Marshal makes a full record from key and value.
func Marshal(dst []byte, timestamp int64, key, value []byte, isTombstone bool) ([]byte, error) {
	keySize := uint32(len(key))
	valueSize := uint32(len(value))
	if isTombstone {
		valueSize = tombstoneMask
	}

	var header [HeaderSize]byte
	binary.LittleEndian.PutUint64(header[4:12], uint64(timestamp))
	binary.LittleEndian.PutUint32(header[12:16], keySize)
	binary.LittleEndian.PutUint32(header[16:20], valueSize)

	crc := crc32.NewIEEE()
	if _, err := crc.Write(header[4:]); err != nil {
		return nil, err
	}
	if _, err := crc.Write(key); err != nil {
		return nil, err
	}
	if !isTombstone {
		if _, err := crc.Write(value); err != nil {
			return nil, err
		}
	}
	binary.LittleEndian.PutUint32(header[0:4], crc.Sum32())

	dst = append(dst, header[:]...)
	dst = append(dst, key...)
	if !isTombstone {
		dst = append(dst, value...)
	}
	return dst, nil
}

// Unmarshal reads one record from data. It returns header, key,
// value, bytes used and error. Value is nil for a tombstone.
func Unmarshal(data []byte) (Header, []byte, []byte, int, error) {
	if len(data) < HeaderSize {
		return Header{}, nil, nil, 0, ErrTooSmall
	}
	wantCRC := binary.LittleEndian.Uint32(data[0:4])
	timestamp := int64(binary.LittleEndian.Uint64(data[4:12]))
	keySize := binary.LittleEndian.Uint32(data[12:16])
	rawValueSize := binary.LittleEndian.Uint32(data[16:20])

	isTombstone := rawValueSize&tombstoneMask != 0
	valueSize := rawValueSize & maxValueLen

	need := HeaderSize + int(keySize) + int(valueSize)
	if isTombstone {
		need = HeaderSize + int(keySize)
	}
	if len(data) < need {
		return Header{}, nil, nil, 0, ErrTooSmall
	}

	key := data[HeaderSize : HeaderSize+int(keySize)]
	var value []byte
	if !isTombstone {
		value = data[HeaderSize+int(keySize) : need]
	}

	crc := crc32.NewIEEE()
	if _, err := crc.Write(data[4:20]); err != nil {
		return Header{}, nil, nil, 0, err
	}
	if _, err := crc.Write(key); err != nil {
		return Header{}, nil, nil, 0, err
	}
	if !isTombstone {
		if _, err := crc.Write(value); err != nil {
			return Header{}, nil, nil, 0, err
		}
	}
	if crc.Sum32() != wantCRC {
		return Header{}, nil, nil, 0, ErrCorrupt
	}

	h := Header{
		CRC:         wantCRC,
		Timestamp:   timestamp,
		KeySize:     keySize,
		ValueSize:   rawValueSize,
		IsTombstone: isTombstone,
	}
	return h, key, value, need, nil
}

// ValueLen returns the real value length, 0 for a tombstone.
func (h Header) ValueLen() int {
	return int(h.ValueSize & maxValueLen)
}
