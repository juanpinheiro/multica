package agent

import (
	"context"
	"time"
)

// defaultResultTeardownGrace is how long a reader-loop backend waits for its
// process to exit on its own after the authoritative completion message arrives
// before force-cancelling the run. The completion message — not process exit —
// is the signal; a CLI that emits it and then hangs (stream-json never closes
// stdout, or a grandchild holds the pipe open) would otherwise block the reader
// until the 30-minute idle watchdog and be mislabelled as blocked. Mirrors the
// codex backend's cancel-on-turn-complete teardown.
const defaultResultTeardownGrace = 5 * time.Second

// resolveResultTeardownGrace returns the configured grace window, falling back
// to the default when unset. Tests inject a small value to keep the
// hung-after-result path fast.
func resolveResultTeardownGrace(override time.Duration) time.Duration {
	if override > 0 {
		return override
	}
	return defaultResultTeardownGrace
}

// scheduleResultTeardown gives the process a grace window to exit on its own
// after its authoritative completion message has been seen, then cancels the
// run context so stdout closes and the reader's scanner unblocks. cmd.WaitDelay
// force-closes any grandchild still holding the pipe. Returns immediately; the
// goroutine also exits if the process leaves on its own first.
func scheduleResultTeardown(ctx context.Context, cancel context.CancelFunc, grace time.Duration) {
	go func() {
		timer := time.NewTimer(grace)
		defer timer.Stop()
		select {
		case <-timer.C:
			cancel()
		case <-ctx.Done():
		}
	}()
}
