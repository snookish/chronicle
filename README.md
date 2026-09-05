# Chronicle

A small Bitcask style key-value store in Go. It appends to a file and keeps an index in memory, so writes are sequential and reads are a single seek.

## How records look on disk

Each record:

```
CRC32 (4) | timestamp nanos (8) | key size (4) | value size (4) | key | value
```

- 20 byte header, little endian
- CRC is over header[4:] plus key and value
- Top bit of value size is the delete marker
- Empty value and delete are different on the wire

This keeps the file simple and easy to replay after a crash. A torn write at the end is cut off on open.

## Quick start

```go
db, err := chronicle.Open("./data")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

if err := db.Put([]byte("hello"), []byte("world")); err != nil {
    log.Fatal(err)
}

val, err := db.Get([]byte("hello"))
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(val))

ok, _ := db.Exists([]byte("hello"))
fmt.Println(ok)

if err := db.Delete([]byte("hello")); err != nil {
    log.Fatal(err)
}

if err := db.Fold(func(k, v []byte) error {
    fmt.Println(string(k), string(v))
    return nil
}); err != nil {
    log.Fatal(err)
}
```

With options:

```go
db, err := chronicle.Open("./data",
    chronicle.WithMaxDataFileSize(64<<20),
    chronicle.WithMaxKeySize(64<<10),
    chronicle.WithMaxValueSize(1<<20),
    chronicle.WithSyncOnWrite(false),
)
```
