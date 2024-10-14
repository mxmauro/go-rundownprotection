// See the LICENSE file for license details.

package rundownprotection_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	rp "github.com/mxmauro/go-rundownprotection"
)

//------------------------------------------------------------------------------

func TestRundownProtection(t *testing.T) {
	wg := sync.WaitGroup{}

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
		defer wg.Done()

		// Wait 10 seconds
		t.Logf("Initiating 10-second sleep.")
		time.Sleep(10 * time.Second)

		// Try to acquire after wait was called. Must fail.
		if r.Acquire() {
			atomic.StoreInt32(&acquireMustFailErr, 1)
		}
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

func TestContext(t *testing.T) {
	wg := sync.WaitGroup{}

	// Create the rundown protection object
	r := rp.Create()

	wg.Add(1)
	go func(ctx context.Context) {
		// Wait for context
		<-ctx.Done()

		wg.Done()
	}(r)

	r.Wait()
	t.Logf("Rundown wait completed.")
	wg.Wait()
	t.Logf("WaitGroup completed.")
}

func TestMultipleWaitCalls1(t *testing.T) {
	r := rp.Create()

	r.Acquire()
	go func() {
		time.Sleep(1 * time.Second)
		r.Release()
	}()

	r.Wait()

	// A new wait after the first one finished, will act as a no-op
	r.Wait()
}

func TestMultipleWaitCalls2(t *testing.T) {
	wg := sync.WaitGroup{}

	r := rp.Create()

	for i := 1; i <= 30; i++ {
		r.Acquire()
		wg.Add(1)

		go func() {
			defer wg.Done()

			time.Sleep(1 * time.Second)
			r.Release()
			r.Wait() // Multiple simultaneous waits are allowed
		}()
	}

	wg.Wait()
}

func TestMultipleWaitCalls2WithoutWG(t *testing.T) {
	r := rp.Create()

	for i := 1; i <= 30; i++ {
		r.Acquire()

		go func() {
			time.Sleep(1 * time.Second)
			r.Release()
			r.Wait() // Multiple simultaneous waits are allowed
		}()
	}

	r.Wait()
}
