package initiative

import "testing"

func TestStatusValid(t *testing.T) {
	for _, s := range AllStatuses() {
		if !s.Valid() {
			t.Errorf("AllStatuses() member %q reported invalid", s)
		}
	}
	for _, s := range []Status{"", "planned", "in_progress", "completed", "DRAFT"} {
		if s.Valid() {
			t.Errorf("Status(%q).Valid() = true, want false", s)
		}
	}
}

func TestCanTransition_Legal(t *testing.T) {
	legal := []struct{ from, to Status }{
		{StatusDraft, StatusReady},
		{StatusDraft, StatusCancelled},
		{StatusReady, StatusRunning},
		{StatusReady, StatusBlocked},
		{StatusReady, StatusCancelled},
		{StatusRunning, StatusInReview},
		{StatusRunning, StatusDone},
		{StatusRunning, StatusBlocked},
		{StatusRunning, StatusCancelled},
		{StatusInReview, StatusDone},
		{StatusInReview, StatusRunning},
		{StatusInReview, StatusCancelled},
		{StatusBlocked, StatusReady},
		{StatusBlocked, StatusRunning},
		{StatusBlocked, StatusCancelled},
	}
	for _, c := range legal {
		if !CanTransition(c.from, c.to) {
			t.Errorf("CanTransition(%s, %s) = false, want true", c.from, c.to)
		}
		if err := Transition(c.from, c.to); err != nil {
			t.Errorf("Transition(%s, %s) = %v, want nil", c.from, c.to, err)
		}
	}
}

func TestCanTransition_Illegal(t *testing.T) {
	illegal := []struct{ from, to Status }{
		{StatusDraft, StatusRunning},   // must pass through ready
		{StatusDraft, StatusDone},      // can't skip the whole lifecycle
		{StatusReady, StatusDone},      // can't skip running
		{StatusReady, StatusInReview},  // can't skip running
		{StatusRunning, StatusReady},   // no going back to ready
		{StatusRunning, StatusDraft},   // no going back to draft
		{StatusDone, StatusRunning},    // done is terminal
		{StatusDone, StatusReady},      // done is terminal
		{StatusCancelled, StatusReady}, // cancelled is terminal
		{StatusDraft, StatusDraft},     // self-transition is not a transition
		{StatusRunning, StatusRunning},
	}
	for _, c := range illegal {
		if CanTransition(c.from, c.to) {
			t.Errorf("CanTransition(%s, %s) = true, want false", c.from, c.to)
		}
		if err := Transition(c.from, c.to); err == nil {
			t.Errorf("Transition(%s, %s) = nil, want error", c.from, c.to)
		}
	}
}

func TestTransition_UnknownStatus(t *testing.T) {
	if err := Transition("planned", StatusReady); err == nil {
		t.Error("Transition from unknown status: want error, got nil")
	}
	if err := Transition(StatusDraft, "shipped"); err == nil {
		t.Error("Transition to unknown status: want error, got nil")
	}
}

func TestIsTerminal(t *testing.T) {
	terminal := map[Status]bool{
		StatusDraft: false, StatusReady: false, StatusRunning: false,
		StatusInReview: false, StatusBlocked: false,
		StatusDone: true, StatusCancelled: true,
	}
	for s, want := range terminal {
		if got := s.IsTerminal(); got != want {
			t.Errorf("Status(%s).IsTerminal() = %v, want %v", s, got, want)
		}
	}
}
