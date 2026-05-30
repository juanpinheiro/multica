package inplace

import "fmt"

// WaitReason builds the human-readable hint stored on a task parked in
// waiting_local_directory. It names the umbrella directory the task needs and,
// when known, the task holding it, so the UI can explain why the run is serial
// instead of running. holder is the id reported by the locker's wait callback;
// it may be empty if the lock was released between the callback and this call.
func WaitReason(dir, holder string) string {
	if holder == "" {
		return fmt.Sprintf("umbrella directory %s is in use by another task", dir)
	}
	return fmt.Sprintf("umbrella directory %s is in use by task %s", dir, holder)
}
