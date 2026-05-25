package batch

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"orderservice/internal/order"
)

func TestServiceAdd_flushesWhenBatchSizeReached(t *testing.T) {
	t.Parallel()

	w := &recordingWriter{}
	svc := NewService(w, SystemClock{}, 2, time.Hour)

	o1 := order.Order{CustomerNumber: "C1", Address: "A1", Postcode: "AB1", PlacedAt: time.Now().UTC()}
	o2 := order.Order{CustomerNumber: "C2", Address: "A2", Postcode: "AB2", PlacedAt: time.Now().UTC()}

	if err := svc.Add(context.Background(), o1); err != nil {
		t.Fatalf("Add 1: %v", err)
	}
	if w.writeCount() != 0 {
		t.Fatalf("after first Add: writeCount = %d, want 0", w.writeCount())
	}
	if err := svc.Add(context.Background(), o2); err != nil {
		t.Fatalf("Add 2: %v", err)
	}
	if w.writeCount() != 1 {
		t.Fatalf("writeCount = %d, want 1", w.writeCount())
	}
	got := w.lastWrite()
	if len(got) != 2 {
		t.Fatalf("last write len = %d, want 2", len(got))
	}
	if got[0].CustomerNumber != "C1" || got[1].CustomerNumber != "C2" {
		t.Fatalf("orders = %+v", got)
	}
}

func TestServiceRun_tickerFlushesBuffer(t *testing.T) {
	t.Parallel()

	w := &recordingWriter{}
	clk := newFakeClock(time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC))
	svc := NewService(w, clk, 100, 10*time.Millisecond)

	o := order.Order{CustomerNumber: "C1", Address: "A1", Postcode: "AB1", PlacedAt: time.Now().UTC()}
	if err := svc.Add(context.Background(), o); err != nil {
		t.Fatalf("Add: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		svc.Run(ctx)
		close(done)
	}()

	waitFor(t, func() bool { return clk.lastTicker() != nil })
	ft := clk.lastTicker()
	select {
	case ft.ch <- clk.Now():
	case <-time.After(time.Second):
		t.Fatal("timed out sending tick")
	}

	waitFor(t, func() bool { return w.writeCount() == 1 })

	cancel()
	<-done
}

func TestServiceAdd_concurrentNoLostOrders(t *testing.T) {
	const (
		goroutines   = 50
		perGoroutine = 4
		batchSize    = 10
	)
	w := &recordingWriter{}
	svc := NewService(w, SystemClock{}, batchSize, time.Hour)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				o := order.Order{
					CustomerNumber: "C",
					Address:        "A",
					Postcode:       "PC",
					PlacedAt:       time.Now().UTC(),
				}
				if err := svc.Add(context.Background(), o); err != nil {
					t.Errorf("Add: %v", err)
				}
			}
		}(g)
	}
	wg.Wait()

	if err := svc.Flush(context.Background()); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	want := goroutines * perGoroutine
	got := w.totalOrders()
	if got != want {
		t.Fatalf("total orders written = %d, want %d", got, want)
	}
}

func TestServiceRun_stopsWhenContextCancelled(t *testing.T) {
	t.Parallel()

	w := &recordingWriter{}
	clk := newFakeClock(time.Now().UTC())
	svc := NewService(w, clk, 100, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		svc.Run(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}

func TestServiceFlush_writesPartialBatch(t *testing.T) {
	t.Parallel()

	w := &recordingWriter{}
	svc := NewService(w, SystemClock{}, 100, time.Hour)

	o := order.Order{CustomerNumber: "C1", Address: "A1", Postcode: "AB1", PlacedAt: time.Now().UTC()}
	if err := svc.Add(context.Background(), o); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := svc.Flush(context.Background()); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if w.writeCount() != 1 {
		t.Fatalf("writeCount = %d, want 1", w.writeCount())
	}
}

func TestServiceFlush_emptyIsNoOp(t *testing.T) {
	t.Parallel()

	w := &recordingWriter{}
	svc := NewService(w, SystemClock{}, 10, time.Hour)

	if err := svc.Flush(context.Background()); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if w.writeCount() != 0 {
		t.Fatalf("writeCount = %d, want 0", w.writeCount())
	}
}

func TestServiceRun_noGoroutineLeak(t *testing.T) {
	runtime.GC()
	before := runtime.NumGoroutine()

	w := &recordingWriter{}
	clk := newFakeClock(time.Now().UTC())
	svc := NewService(w, clk, 100, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		svc.Run(ctx)
		close(done)
	}()
	cancel()
	<-done

	runtime.GC()
	after := runtime.NumGoroutine()
	if after > before+2 {
		t.Fatalf("goroutines before=%d after=%d", before, after)
	}
}

func TestDurationUntilNextMidnightUTC(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 25, 15, 30, 0, 0, time.UTC)
	d := DurationUntilNextMidnightUTC(now)
	want := 8*time.Hour + 30*time.Minute
	if d != want {
		t.Fatalf("DurationUntilNextMidnightUTC() = %v, want %v", d, want)
	}
}

type recordingWriter struct {
	mu     sync.Mutex
	writes [][]order.Order
	err    error
}

func (w *recordingWriter) Write(_ context.Context, orders []order.Order) error {
	if w.err != nil {
		return w.err
	}
	cp := append([]order.Order(nil), orders...)
	w.mu.Lock()
	w.writes = append(w.writes, cp)
	w.mu.Unlock()
	return nil
}

func (w *recordingWriter) writeCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.writes)
}

func (w *recordingWriter) lastWrite() []order.Order {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.writes) == 0 {
		return nil
	}
	return w.writes[len(w.writes)-1]
}

func (w *recordingWriter) totalOrders() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, batch := range w.writes {
		n += len(batch)
	}
	return n
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}
