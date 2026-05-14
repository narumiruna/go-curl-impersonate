package curl

import (
	"errors"
	"sync"
	"testing"
)

func TestHandlePoolReusesReleasedHandles(t *testing.T) {
	pool := NewHandlePool()
	first, err := pool.Lease()
	if err != nil {
		t.Fatalf("Lease returned error: %v", err)
	}
	firstID := first.ID()
	first.Release()
	first.Release()

	second, err := pool.Lease()
	if err != nil {
		t.Fatalf("second Lease returned error: %v", err)
	}
	if second.ID() != firstID {
		t.Fatalf("second ID = %d, want reused ID %d", second.ID(), firstID)
	}
}

func TestHandlePoolCloseRejectsLease(t *testing.T) {
	pool := NewHandlePool()
	lease, err := pool.Lease()
	if err != nil {
		t.Fatalf("Lease returned error: %v", err)
	}
	pool.Close()
	lease.Release()
	if _, err := pool.Lease(); !errors.Is(err, ErrHandlePoolClosed) {
		t.Fatalf("Lease after close error = %v, want ErrHandlePoolClosed", err)
	}
}

func TestHandlePoolConcurrentLeaseRelease(t *testing.T) {
	pool := NewHandlePool()
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lease, err := pool.Lease()
			if err != nil {
				t.Errorf("Lease returned error: %v", err)
				return
			}
			lease.Release()
		}()
	}
	wg.Wait()
}
