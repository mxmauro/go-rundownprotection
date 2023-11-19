package rundown_protection_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	rp "github.com/randlabs/rundown-protection"
)

//------------------------------------------------------------------------------

func TestRundownProtection(t *testing.T) {
	var wg sync.WaitGroup

	// Create the rundown protection object
	r := rp.Create()

	// Run 5 goroutines which will complete after 5 seconds
	for i := 1; i <= 5; i++ {
		// Acquire rundown lock
		if !r.Acquire() {
			t.Fatalf("Unable to acquire rundown protection")
		}

		// Launch goroutine
		t.Logf("Starting goroutine #%v.", i)

		go func(idx int) {
			// Wait 5 seconds
			time.Sleep(5 * time.Second)

			t.Logf("Goroutine #%v completed.", idx)

			// Release rundown lock
			r.Release()
		}(i)
	}

	// The following goroutine will run after 10 seconds simulating acquisition of the rundown object
	// After the wait (shutdown) call.
	var acquireMustFailErr int32 = 0
	wg.Add(1)
	go func() {
		// Wait 10 seconds
		t.Logf("Initiating 10-second sleep.")
		time.Sleep(10 * time.Second)

		// Try to acquire after wait was called. Must fail.
		if r.Acquire() {
			atomic.StoreInt32(&acquireMustFailErr, 1)
		}

		wg.Done()
	}()

	t.Logf("Before rundown wait.")
	r.Wait()
	if atomic.LoadInt32(&acquireMustFailErr) != 0 {
		t.Fatalf("Acquired rundown protection after Wait call.")
	}

	if !errors.Is(r.Err(), context.Canceled) {
		t.Fatalf("Associated context is not canceled.")
	}

	t.Logf("Rundown wait completed. Waiting for the 10-second sleep to finish.")

	wg.Wait()
	t.Logf("Done!")
}
