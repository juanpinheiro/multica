package inplace

import (
	"context"
	"path/filepath"
	"sync"
)

// WaitFunc is invoked at most once per Acquire, when the call must block because
// the target directory is already locked. It receives the current holder's id so
// the caller can surface a wait reason (e.g. flip a task to a waiting state).
type WaitFunc func(holder string)

// ReleaseFunc releases a lock taken by Acquire. It is idempotent: calling it more
// than once, or via defer after an early return, is safe.
type ReleaseFunc func()

// Locker serializes work that shares the same on-disk directory. Locks are keyed
// on the symlink-resolved real path, so two routes to the same directory (a
// symlink and its target) collapse onto a single lock. A lock is held for the
// caller's whole lifetime — from acquire until the returned ReleaseFunc runs.
type Locker struct {
	mu    sync.Mutex
	locks map[string]*lock
}

// NewLocker returns a ready-to-use Locker.
func NewLocker() *Locker {
	return &Locker{locks: map[string]*lock{}}
}

// lock is the per-directory state. token carries a single value while the lock
// is free; taking it acquires the lock, returning it releases. holder and refs
// are guarded by Locker.mu; refs counts live participants (holder plus waiters)
// so the entry can be evicted once nothing references it.
type lock struct {
	token  chan struct{}
	holder string
	refs   int
}

// Acquire locks the real path of dir for holder, blocking until the lock is free
// or ctx is cancelled. When the lock is already held, onWait (if non-nil) fires
// exactly once with the current holder before Acquire blocks. On success it
// returns a ReleaseFunc; on cancellation it returns ctx.Err() without taking the
// lock and without wedging later acquirers.
func (l *Locker) Acquire(ctx context.Context, dir, holder string, onWait WaitFunc) (ReleaseFunc, error) {
	key := resolve(dir)
	lk, current, acquired := l.enter(key, holder)
	if acquired {
		return l.releaser(key, lk), nil
	}
	if onWait != nil {
		onWait(current)
	}
	return l.wait(ctx, key, lk, holder)
}

// enter registers a participant on key's lock and tries the fast path. It returns
// the lock, the current holder (when the fast path failed), and whether the lock
// was taken immediately.
func (l *Locker) enter(key, holder string) (lk *lock, current string, acquired bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	lk = l.locks[key]
	if lk == nil {
		lk = &lock{token: make(chan struct{}, 1)}
		lk.token <- struct{}{}
		l.locks[key] = lk
	}
	lk.refs++
	select {
	case <-lk.token:
		lk.holder = holder
		return lk, "", true
	default:
		return lk, lk.holder, false
	}
}

// wait blocks until key's lock frees and is taken for holder, or until ctx is
// cancelled — in which case it drops this participant without taking the lock and
// without wedging later acquirers.
func (l *Locker) wait(ctx context.Context, key string, lk *lock, holder string) (ReleaseFunc, error) {
	select {
	case <-lk.token:
		l.mu.Lock()
		lk.holder = holder
		l.mu.Unlock()
		return l.releaser(key, lk), nil
	case <-ctx.Done():
		l.mu.Lock()
		lk.refs--
		l.evict(key, lk)
		l.mu.Unlock()
		return nil, ctx.Err()
	}
}

// Holder returns the id of the current holder of dir's lock, or "" if the lock
// is free or has never been taken.
func (l *Locker) Holder(dir string) string {
	key := resolve(dir)
	l.mu.Lock()
	defer l.mu.Unlock()
	if lk := l.locks[key]; lk != nil {
		return lk.holder
	}
	return ""
}

// releaser returns the lock's one-shot release closure. The send never blocks:
// a held lock means the token slot is empty, so it is safe under l.mu.
func (l *Locker) releaser(key string, lk *lock) ReleaseFunc {
	var once sync.Once
	return func() {
		once.Do(func() {
			l.mu.Lock()
			defer l.mu.Unlock()
			lk.holder = ""
			lk.refs--
			lk.token <- struct{}{}
			l.evict(key, lk)
		})
	}
}

// evict drops a lock from the map once nothing references it, so the map does not
// grow without bound. The caller holds l.mu.
func (l *Locker) evict(key string, lk *lock) {
	if lk.refs == 0 {
		delete(l.locks, key)
	}
}

// resolve maps a directory to its lock key: the symlink-resolved real path, so
// aliased routes to one directory collapse onto a single lock. It falls back to
// the absolute, cleaned path when the directory cannot be resolved.
func resolve(dir string) string {
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		return real
	}
	if abs, err := filepath.Abs(dir); err == nil {
		return abs
	}
	return filepath.Clean(dir)
}
