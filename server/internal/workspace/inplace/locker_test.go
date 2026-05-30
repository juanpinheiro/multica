package inplace_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/multica-ai/multica/server/internal/workspace/inplace"
)

// acquireTimeout bounds a blocking Acquire in tests so a regression that wedges
// the locker fails fast instead of hanging the suite.
const acquireTimeout = 2 * time.Second

func TestAcquireFastPathRecordsHolder(t *testing.T) {
	l := inplace.NewLocker()
	dir := t.TempDir()

	release, err := l.Acquire(context.Background(), dir, "task-1", nil)
	if err != nil {
		t.Fatalf("Acquire = %v, want nil", err)
	}
	if got := l.Holder(dir); got != "task-1" {
		t.Fatalf("Holder = %q, want %q", got, "task-1")
	}

	release()
	if got := l.Holder(dir); got != "" {
		t.Fatalf("Holder after release = %q, want empty", got)
	}
}

func TestHolderEmptyWhenUnlocked(t *testing.T) {
	l := inplace.NewLocker()
	if got := l.Holder(t.TempDir()); got != "" {
		t.Fatalf("Holder of unlocked dir = %q, want empty", got)
	}
}

func TestSecondAcquirerBlocksAndFiresWaitOnce(t *testing.T) {
	l := inplace.NewLocker()
	dir := t.TempDir()

	first, err := l.Acquire(context.Background(), dir, "first", nil)
	if err != nil {
		t.Fatalf("first Acquire = %v", err)
	}

	var waitCalls int32
	waited := make(chan string, 1)
	acquired := make(chan inplace.ReleaseFunc, 1)
	go func() {
		release, err := l.Acquire(context.Background(), dir, "second", func(holder string) {
			atomic.AddInt32(&waitCalls, 1)
			waited <- holder
		})
		if err != nil {
			t.Errorf("second Acquire = %v", err)
			return
		}
		acquired <- release
	}()

	select {
	case holder := <-waited:
		if holder != "first" {
			t.Fatalf("wait callback holder = %q, want %q", holder, "first")
		}
	case <-time.After(acquireTimeout):
		t.Fatal("wait callback never fired")
	}

	select {
	case <-acquired:
		t.Fatal("second acquirer took the lock while first still held it")
	default:
	}

	first()

	select {
	case release := <-acquired:
		if got := l.Holder(dir); got != "second" {
			t.Fatalf("Holder after handoff = %q, want %q", got, "second")
		}
		release()
	case <-time.After(acquireTimeout):
		t.Fatal("second acquirer never acquired after release")
	}

	if n := atomic.LoadInt32(&waitCalls); n != 1 {
		t.Fatalf("wait callback fired %d times, want 1", n)
	}
}

func TestReleaseIsIdempotent(t *testing.T) {
	l := inplace.NewLocker()
	dir := t.TempDir()

	release, err := l.Acquire(context.Background(), dir, "task", nil)
	if err != nil {
		t.Fatalf("Acquire = %v", err)
	}
	release()
	release() // second call must be a safe no-op

	if got := l.Holder(dir); got != "" {
		t.Fatalf("Holder after double release = %q, want empty", got)
	}
}

func TestCancellationWhileWaiting(t *testing.T) {
	l := inplace.NewLocker()
	dir := t.TempDir()

	first, err := l.Acquire(context.Background(), dir, "first", nil)
	if err != nil {
		t.Fatalf("first Acquire = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	waited := make(chan struct{}, 1)
	result := make(chan error, 1)
	go func() {
		_, err := l.Acquire(ctx, dir, "waiter", func(string) { waited <- struct{}{} })
		result <- err
	}()

	<-waited
	cancel()

	select {
	case err := <-result:
		if err != context.Canceled {
			t.Fatalf("cancelled Acquire = %v, want context.Canceled", err)
		}
	case <-time.After(acquireTimeout):
		t.Fatal("cancelled Acquire never returned")
	}

	if got := l.Holder(dir); got != "first" {
		t.Fatalf("cancelled waiter stole the lock; Holder = %q, want %q", got, "first")
	}

	first()

	// A later acquirer must not be wedged by the cancelled one.
	release, err := l.Acquire(context.Background(), dir, "later", nil)
	if err != nil {
		t.Fatalf("later Acquire after cancellation = %v, want nil", err)
	}
	release()
}

func TestDistinctKeysDoNotContend(t *testing.T) {
	l := inplace.NewLocker()
	dirA, dirB := t.TempDir(), t.TempDir()

	releaseA, err := l.Acquire(context.Background(), dirA, "a", nil)
	if err != nil {
		t.Fatalf("Acquire A = %v", err)
	}
	defer releaseA()

	ctx, cancel := context.WithTimeout(context.Background(), acquireTimeout)
	defer cancel()
	releaseB, err := l.Acquire(ctx, dirB, "b", func(string) {
		t.Fatal("distinct key fired the wait callback")
	})
	if err != nil {
		t.Fatalf("Acquire B = %v, want nil (no contention)", err)
	}
	releaseB()
}

func TestSymlinkAliasesCollapseToOneLock(t *testing.T) {
	l := inplace.NewLocker()
	realDir := t.TempDir()
	link := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(realDir, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlinks unavailable: %v", err)
		}
		t.Fatalf("symlink: %v", err)
	}

	first, err := l.Acquire(context.Background(), realDir, "first", nil)
	if err != nil {
		t.Fatalf("Acquire real = %v", err)
	}

	waited := make(chan struct{}, 1)
	acquired := make(chan inplace.ReleaseFunc, 1)
	go func() {
		release, err := l.Acquire(context.Background(), link, "second", func(string) { waited <- struct{}{} })
		if err != nil {
			t.Errorf("Acquire via alias = %v", err)
			return
		}
		acquired <- release
	}()

	select {
	case <-waited:
	case <-time.After(acquireTimeout):
		t.Fatal("alias did not contend with real path; aliases did not collapse")
	}

	first()
	select {
	case release := <-acquired:
		release()
	case <-time.After(acquireTimeout):
		t.Fatal("alias never acquired after release")
	}
}
