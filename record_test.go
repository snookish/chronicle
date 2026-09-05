package chronicle

import (
	"bytes"
	"testing"
)

func TestMarshalUnmarshal(t *testing.T) {
	ts := int64(1234567890000000)
	key := []byte("user:42")
	val := []byte(`{"name":"ada"}`)

	b, err := Marshal(nil, ts, key, val, false)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	h, k, v, n, err := Unmarshal(b)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if n != len(b) {
		t.Fatalf("n=%d want %d", n, len(b))
	}
	if h.Timestamp != ts || h.IsTombstone {
		t.Fatalf("header mismatch %+v", h)
	}
	if !bytes.Equal(k, key) || !bytes.Equal(v, val) {
		t.Fatalf("key/val mismatch")
	}
	if h.ValueLen() != len(val) {
		t.Fatalf("value len %d", h.ValueLen())
	}
}

func TestTombstoneDistinguishesEmptyValue(t *testing.T) {
	ts := int64(1)
	key := []byte("k")

	bEmpty, err := Marshal(nil, ts, key, []byte{}, false)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	h, _, v, _, err := Unmarshal(bEmpty)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if h.IsTombstone {
		t.Fatal("empty value seen as tombstone")
	}
	if v == nil || len(v) != 0 {
		t.Fatalf("empty value should be empty slice, got %v", v)
	}

	bTomb, err := Marshal(nil, ts, key, nil, true)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	h2, _, v2, _, err := Unmarshal(bTomb)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !h2.IsTombstone || v2 != nil {
		t.Fatalf("tombstone mismatch h=%+v v=%v", h2, v2)
	}
	if bytes.Equal(bEmpty, bTomb) {
		t.Fatal("empty value and tombstone should differ")
	}
}

func TestCorruptDetection(t *testing.T) {
	b, err := Marshal(nil, 42, []byte("k"), []byte("v"), false)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	b[0] ^= 0xFF
	if _, _, _, _, err := Unmarshal(b); err != ErrCorrupt {
		t.Fatalf("want ErrCorrupt got %v", err)
	}
	if _, _, _, _, err := Unmarshal(b[:10]); err != ErrTooSmall {
		t.Fatalf("want ErrTooSmall got %v", err)
	}
	b2, err := Marshal(nil, 42, []byte("k"), []byte("v"), false)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	b2[len(b2)-1] ^= 0xFF
	if _, _, _, _, err := Unmarshal(b2); err != ErrCorrupt {
		t.Fatalf("want ErrCorrupt on value corruption got %v", err)
	}
}

func TestMarshalAppend(t *testing.T) {
	b, err := Marshal(nil, 1, []byte("a"), []byte("1"), false)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	b, err = Marshal(b, 2, []byte("b"), []byte("2"), false)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	_, _, _, n, err := Unmarshal(b)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	_, k2, v2, _, err := Unmarshal(b[n:])
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(k2) != "b" || string(v2) != "2" {
		t.Fatal("second record mismatch")
	}
}

func FuzzMarshalUnmarshal(f *testing.F) {
	f.Add(int64(0), []byte("k"), []byte("v"), false)
	f.Add(int64(0), []byte("k"), []byte(""), true)
	f.Fuzz(func(t *testing.T, ts int64, key, val []byte, tomb bool) {
		if len(key) > 64*1024 {
			t.Skip()
		}
		if len(val) > 1<<20 {
			t.Skip()
		}
		if tomb {
			val = nil
		}
		b, err := Marshal(nil, ts, key, val, tomb)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		h, k, v, n, err := Unmarshal(b)
		if err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if n != len(b) {
			t.Fatalf("n mismatch")
		}
		if h.IsTombstone != tomb {
			t.Fatalf("tombstone flag mismatch")
		}
		if !bytes.Equal(k, key) {
			t.Fatalf("key mismatch")
		}
		if tomb {
			if v != nil {
				t.Fatalf("tombstone value should be nil")
			}
		} else if !bytes.Equal(v, val) {
			t.Fatalf("value mismatch")
		}
	})
}
