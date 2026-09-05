package chronicle

import "errors"

// Sentinel errors for the store.
var (
	ErrBadKey   = errors.New("chronicle: bad key")
	ErrNotFound = errors.New("chronicle: key not found")
	ErrClosed   = errors.New("chronicle: store is closed")
)
