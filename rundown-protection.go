/*
Golang implementation of a rundown protection for accessing a shared object

Source code and other details for the project are available at GitHub:

	https://github.com/RandLabs/rundown-protection

More usage please see README.md and tests.
*/

package rundown_protection

import (
	"sync/atomic"
)

//------------------------------------------------------------------------------

const (
	rundownActive uint32 = 0x80000000
)

//------------------------------------------------------------------------------

type RundownProtection struct {
	counter uint32
	done    chan struct{}
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
	r.done = make(chan struct{}, 1)
	return
}

// Acquire increments the usage counter unless a rundown is in progress.
func (r *RundownProtection) Acquire() bool {
	for {
		val := atomic.LoadUint32(&r.counter)

		// If a rundown is in progress, cancel
		if (val & rundownActive) != 0 {
			return false
		}

		// Try to increment the reference counter
		if atomic.CompareAndSwapUint32(&r.counter, val, val+1) {
			break
		}
	}
	return true
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
				r.done <- struct{}{}
			}
			break
		}
	}
	return
}

// Wait initiates the shutdown process and waits until all acquisitions are released.
func (r *RundownProtection) Wait() {
	var val uint32

	// Set rundown active flag
	for {
		val = atomic.LoadUint32(&r.counter)
		if atomic.CompareAndSwapUint32(&r.counter, val, val|rundownActive) {
			break
		}
	}

	// Wait if a reference is being held
	if val != 0 {
		// IMPORTANT NOTE: The next sentence will panic with "fatal error: all goroutines are asleep - deadlock!"
		//                 if your code generates one. For e.g., if you put the Wait call inside a mutex being held
		//                 and a goroutine tries to lock the same mutex.
		<-r.done
	}
	close(r.done)
	return
}
