# Chronicle

A Bitcask-inspired key-value store in Go. Append-only files on disk, in-memory index for fast reads. Work in progress.

## Record format

Each record on disk is:

```
CRC32 (4 bytes) | Timestamp nanos int64 (8) | Key size u32 (4) | Value size u32 (4) | Key | Value
```

- 20 byte header, little endian
- CRC covers header after the CRC plus key and value
- High bit of value size is the tombstone flag for deletes
- Empty value and delete are different on the wire

This keeps writes sequential and reads to one disk seek.
