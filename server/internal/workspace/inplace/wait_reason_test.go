package inplace

import (
	"strings"
	"testing"
)

func TestWaitReason_NamesPathAndHolder(t *testing.T) {
	got := WaitReason("/home/dev/code/meu-produto", "task-123")
	if !strings.Contains(got, "/home/dev/code/meu-produto") {
		t.Fatalf("reason %q does not name the held directory", got)
	}
	if !strings.Contains(got, "task-123") {
		t.Fatalf("reason %q does not name the holder", got)
	}
}

func TestWaitReason_EmptyHolderStaysReadable(t *testing.T) {
	got := WaitReason("/home/dev/code/meu-produto", "")
	if !strings.Contains(got, "/home/dev/code/meu-produto") {
		t.Fatalf("reason %q does not name the held directory", got)
	}
	if got == "" {
		t.Fatal("reason is empty for an unknown holder")
	}
}
