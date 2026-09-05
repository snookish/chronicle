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

type Header struct {
	CRC         uint32
	Timestamp   int64
	KeySize     uint32
	ValueSize   uint32
	IsTombstone bool
}

// Marshal builds a record from the given key and value.
// If isTombstone is true the value is ignored and a delete marker is written.
func Marshal(dst []byte, ts int64, key, value []byte, isTombstone bool) ([]byte, error) {
	ksz := uint32(len(key))
	vsz := uint32(len(value))
	if isTombstone {
		vsz = tombstoneMask
	}

	var hdr [HeaderSize]byte
	binary.LittleEndian.PutUint64(hdr[4:12], uint64(ts))
	binary.LittleEndian.PutUint32(hdr[12:16], ksz)
	binary.LittleEndian.PutUint32(hdr[16:20], vsz)

	crc := crc32.NewIEEE()
	if _, err := crc.Write(hdr[4:]); err != nil {
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
	binary.LittleEndian.PutUint32(hdr[0:4], crc.Sum32())

	dst = append(dst, hdr[:]...)
	dst = append(dst, key...)
	if !isTombstone {
		dst = append(dst, value...)
	}
	return dst, nil
}

// Unmarshal reads one record from b and checks the crc.
// Returns the header, key, value and how many bytes were used.
func Unmarshal(b []byte) (Header, []byte, []byte, int, error) {
	if len(b) < HeaderSize {
		return Header{}, nil, nil, 0, ErrTooSmall
	}
	crcWant := binary.LittleEndian.Uint32(b[0:4])
	ts := int64(binary.LittleEndian.Uint64(b[4:12]))
	ksz := binary.LittleEndian.Uint32(b[12:16])
	vszRaw := binary.LittleEndian.Uint32(b[16:20])

	isTombstone := vszRaw&tombstoneMask != 0
	vsz := vszRaw & maxValueLen

	need := HeaderSize + int(ksz) + int(vsz)
	if isTombstone {
		need = HeaderSize + int(ksz)
	}
	if len(b) < need {
		return Header{}, nil, nil, 0, ErrTooSmall
	}

	var value []byte
	key := b[HeaderSize : HeaderSize+int(ksz)]

	if !isTombstone {
		value = b[HeaderSize+int(ksz) : need]
	}

	crc := crc32.NewIEEE()
	if _, err := crc.Write(b[4:20]); err != nil {
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
	if crc.Sum32() != crcWant {
		return Header{}, nil, nil, 0, ErrCorrupt
	}

	h := Header{
		CRC:         crcWant,
		Timestamp:   ts,
		KeySize:     ksz,
		ValueSize:   vszRaw,
		IsTombstone: isTombstone,
	}
	return h, key, value, need, nil
}

// ValueLen is the logical value length, 0 for a tombstone.
func (h Header) ValueLen() int {
	return int(h.ValueSize & maxValueLen)
}
