package chronicle

import "time"

// Clock gives the current time in nanos.
type Clock interface {
	Now() int64
}

type systemClock struct{}

func (systemClock) Now() int64 { return time.Now().UnixNano() }

// config holds settings for the store.
type config struct {
	maxDataFileBytes int64 // when active file reaches this size we would rotate
	maxKeyBytes      int   // biggest key allowed
	maxValueBytes    int   // biggest value allowed
	syncOnWrite      bool  // fsync after every put when true
	clock            Clock
}

func defaultConfig() config {
	return config{
		maxDataFileBytes: 64 << 20,
		maxKeyBytes:      64 << 10,
		maxValueBytes:    1 << 20,
		syncOnWrite:      true,
		clock:            systemClock{},
	}
}

func (c config) validate() error {
	if c.maxDataFileBytes <= 0 || c.maxKeyBytes <= 0 || c.maxValueBytes <= 0 {
		return ErrBadKey
	}
	if c.clock == nil {
		return ErrBadKey
	}
	return nil
}

// Option tweaks how the store is opened.
type Option func(*config)

// WithMaxDataFileSize sets the size limit for the active file.
func WithMaxDataFileSize(n int64) Option {
	return func(c *config) { c.maxDataFileBytes = n }
}

// WithMaxKeySize sets the max key length.
func WithMaxKeySize(n int) Option {
	return func(c *config) { c.maxKeyBytes = n }
}

// WithMaxValueSize sets the max value length.
func WithMaxValueSize(n int) Option {
	return func(c *config) { c.maxValueBytes = n }
}

// WithSyncOnWrite controls whether each put is fsynced.
func WithSyncOnWrite(v bool) Option {
	return func(c *config) { c.syncOnWrite = v }
}

// WithClock sets a custom clock. Handy for tests.
func WithClock(clk Clock) Option {
	return func(c *config) { c.clock = clk }
}
