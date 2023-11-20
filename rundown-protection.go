package rundown_protection

import (
	"context"
	"sync/atomic"
	"time"
)

//------------------------------------------------------------------------------

const (
	rundownActive uint32 = 0x80000000
)

//------------------------------------------------------------------------------

type RundownProtection struct {
	counter   uint32
	waitAllCh chan struct{}
	doneCh    chan struct{}
}

//------------------------------------------------------------------------------

// Create creates a new rundown protection object.
func Create() *RundownProtection {
	r := &RundownProtection{}
	r.Initialize()
	return r
}

// Initialize initializes a rundown protection object.
func (r *RundownProtection) Initialize() {
	atomic.StoreUint32(&r.counter, 0)
	r.waitAllCh = make(chan struct{}, 1)
	r.doneCh = make(chan struct{})
	return
}

// Acquire increments the usage counter unless a rundown is in progress.
func (r *RundownProtection) Acquire() bool {
	for {
		val := atomic.LoadUint32(&r.counter)

		// If a rundown is in progress, cancel acquisition
		if (val & rundownActive) != 0 {
			return false
		}

		// Try to increment the reference counter
		if atomic.CompareAndSwapUint32(&r.counter, val, val+1) {
			return true
		}
	}
}

// Release decrements the usage counter.
func (r *RundownProtection) Release() {
	for {
		// Decrement usage counter but keep the rundown active flag if present
		val := atomic.LoadUint32(&r.counter)
		newVal := (val & rundownActive) | ((val & (^rundownActive)) - 1)
		if atomic.CompareAndSwapUint32(&r.counter, val, newVal) {
			// If a wait is in progress and the last reference was released, complete the wait
			if newVal == rundownActive {
				r.waitAllCh <- struct{}{}
			}
			return
		}
	}
}

// Wait initiates the shutdown process and waits until all acquisitions are released.
func (r *RundownProtection) Wait() {
	for {
		// Set rundown active flag
		val := atomic.LoadUint32(&r.counter)
		if (val & rundownActive) != 0 {
			panic("rundownProtection::wait already called")
		}

		if atomic.CompareAndSwapUint32(&r.counter, val, val|rundownActive) {
			// First signal our context wrapper
			close(r.doneCh)

			// If a reference is still being held, wait until released
			if val != 0 {
				// IMPORTANT NOTE: "fatal error: all goroutines are asleep - deadlock!" panic will be raised on the
				//                 channel operation below if, for e.g., you put the Wait call inside a mutex being
				//                 held and a goroutine tries to lock the same mutex.
				//
				// NOTE: It is possible the channel already contains a buffered object if the references being held
				//       are released before this line executes.
				<-r.waitAllCh
			}

			close(r.waitAllCh)
			return
		}
	}
}

// Deadline returns the time when work done on behalf of this context should be canceled.
func (_ *RundownProtection) Deadline() (deadline time.Time, ok bool) {
	return
}

// Done returns a channel that's closed when work done on behalf of this context should be canceled.
func (r *RundownProtection) Done() <-chan struct{} {
	return r.doneCh
}

// Err returns a non-nil error explaining why the channel was closed or nil if still open.
func (r *RundownProtection) Err() error {
	if (atomic.LoadUint32(&r.counter) & rundownActive) != 0 {
		return context.Canceled
	}
	return nil
}

// Value returns the value associated with this context for key, if any exists, or nil.
func (_ *RundownProtection) Value(_ any) any {
	return nil
}
