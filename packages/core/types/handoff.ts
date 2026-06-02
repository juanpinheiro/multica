// A CommandResult records a single command invocation and its exit code.
export interface HandoffCommandResult {
  command: string;
  exit_code: number;
}

// A Handoff is the structured output a worker Run records when it finishes an
// Issue. It captures what was done, what remains, which commands ran, and any
// unexpected discoveries. Immutable once written.
export interface Handoff {
  id: string;
  workspace_id: string;
  issue_id: string;
  run_id: string;
  done: string[];
  left_undone: string[];
  commands: HandoffCommandResult[];
  discoveries: string[];
  created_at: string;
}

export interface ListHandoffsResponse {
  handoffs: Handoff[];
}

// HandoffState is the current observable state of an Issue as derived from its
// Handoffs. Mirrors server/internal/handoff.State.
export interface HandoffState {
  done: string[];
  left_undone: string[];
  discoveries: string[];
}
