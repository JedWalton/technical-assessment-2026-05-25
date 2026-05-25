package batch

import (
	"context"
	"sync"
	"time"

	"orderservice/internal/order"
)

// Writer persists a batch of orders (implemented by FileWriter in production).
type Writer interface {
	Write(ctx context.Context, orders []order.Order) error
}

// Service buffers orders and flushes them when the batch is full or on a
// periodic ticker.
type Service struct {
	writer     Writer
	clock      Clock
	batchSize  int
	flushEvery time.Duration

	mu  sync.Mutex
	buf []order.Order
}

// NewService returns a batching service. flushEvery is the interval between
// periodic flushes (e.g. DurationUntilNextMidnightUTC for end-of-day).
func NewService(writer Writer, clock Clock, batchSize int, flushEvery time.Duration) *Service {
	return &Service{
		writer:     writer,
		clock:      clock,
		batchSize:  batchSize,
		flushEvery: flushEvery,
	}
}

// Add appends an order to the buffer and flushes when batchSize is reached.
func (s *Service) Add(ctx context.Context, o order.Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	var flush bool
	s.mu.Lock()
	s.buf = append(s.buf, o)
	if len(s.buf) >= s.batchSize {
		flush = true
	}
	s.mu.Unlock()

	if flush {
		return s.Flush(ctx)
	}
	return nil
}

// Flush writes all buffered orders via the Writer and clears the buffer. No-op
// when the buffer is empty.
func (s *Service) Flush(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	if len(s.buf) == 0 {
		s.mu.Unlock()
		return nil
	}
	batch := append([]order.Order(nil), s.buf...)
	s.buf = nil
	s.mu.Unlock()

	return s.writer.Write(ctx, batch)
}

// Run flushes on each ticker tick until ctx is cancelled. The caller should
// invoke Flush after Run returns to drain any remaining orders.
func (s *Service) Run(ctx context.Context) {
	ticker := s.clock.NewTicker(s.flushEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C():
			_ = s.Flush(ctx)
		}
	}
}
