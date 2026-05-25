package batch

import (
	"sync"
	"time"
)

// Clock provides time and tickers for periodic batch flush (testable seam).
type Clock interface {
	Now() time.Time
	NewTicker(d time.Duration) Ticker
}

// Ticker delivers periodic signals until stopped.
type Ticker interface {
	C() <-chan time.Time
	Stop()
}

// SystemClock uses the system clock (production implementation).
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }

func (SystemClock) NewTicker(d time.Duration) Ticker {
	return &realTicker{t: time.NewTicker(d)}
}

type realTicker struct {
	t *time.Ticker
}

func (r *realTicker) C() <-chan time.Time { return r.t.C }
func (r *realTicker) Stop()               { r.t.Stop() }

// DurationUntilNextMidnightUTC returns the duration from now until the next
// UTC midnight. Used as the default EOD flush interval in configuration.
func DurationUntilNextMidnightUTC(now time.Time) time.Duration {
	utc := now.UTC()
	next := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)
	return next.Sub(utc)
}

// fakeClock and fakeTicker support deterministic tests.
type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	tickers []*fakeTicker
}

func newFakeClock(now time.Time) *fakeClock {
	return &fakeClock{now: now.UTC()}
}

func (f *fakeClock) Now() time.Time { return f.now }

func (f *fakeClock) NewTicker(time.Duration) Ticker {
	ft := &fakeTicker{ch: make(chan time.Time, 1)}
	f.mu.Lock()
	f.tickers = append(f.tickers, ft)
	f.mu.Unlock()
	return ft
}

func (f *fakeClock) lastTicker() *fakeTicker {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.tickers) == 0 {
		return nil
	}
	return f.tickers[len(f.tickers)-1]
}

type fakeTicker struct {
	ch chan time.Time
}

func (f *fakeTicker) C() <-chan time.Time { return f.ch }
func (f *fakeTicker) Stop()               {}
