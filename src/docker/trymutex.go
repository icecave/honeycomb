package docker

import "context"

// TryMutex is a mutex that allows lock attempts to fail and/or timeout.
type TryMutex chan struct{}

// NewTryMutex returns a new TryMutex.
func NewTryMutex() TryMutex {
	mutex := make(TryMutex, 1)
	mutex <- struct{}{}

	return mutex
}

// Lock blocks the calling coroutine until it acquires the lock.
func (mutex TryMutex) Lock() {
	<-mutex
}

// LockWithContext blocks the calling coroutine until it acquires the lock, or
// the given context is cancelled or times out. A return value of true indicates
// that the lock was acquired.
func (mutex TryMutex) LockWithContext(ctx context.Context) bool {
	select {
	case <-mutex:
		return true
	case <-ctx.Done():
		return false
	}
}

// TryLock attempts to acquire the lock without blocking. A return value of true
// indicates that the lock was acquired.
func (mutex TryMutex) TryLock() bool {
	select {
	case <-mutex:
		return true
	default:
		return false
	}
}

// TryLockOrWait attempts to acquire the lock without blocking. If the lock is
// already acquired the calling goroutine blocks until the lock is released.
// A return value of true indicates that the lock was acquired.
func (mutex TryMutex) TryLockOrWait() bool {
	select {
	case <-mutex:
		return true
	default: // note: we could end up waiting twice here
		mutex <- <-mutex
		return false
	}
}

// TryLockOrWaitWithContext attempts to acquire the lock without blocking. If
// the lock is already acquired the calling goroutine blocks until the lock is
// released or the given context is cancelled or times out. A return value of
// true indicates that the lock was acquired.
func (mutex TryMutex) TryLockOrWaitWithContext(ctx context.Context) bool {
	select {
	case <-mutex:
		return true
	default:
		select {
		case <-ctx.Done():
		case token := <-mutex:
			mutex <- token
		}
		return false
	}
}

// Unlock releases a previously acquired lock.
func (mutex TryMutex) Unlock() {
	select {
	case mutex <- struct{}{}:
	default:
	}
}
