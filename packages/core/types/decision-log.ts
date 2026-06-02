// A DecisionLogEntry is one architectural decision a retrospective Run records
// at an Initiative boundary. adr_refs and context_terms link the decision back to
// the ADRs and CONTEXT.md glossary terms it touches. Mirrors
// server/internal/decisionlog.Entry plus the persisted row fields.
export interface DecisionLogEntry {
  id: string;
  workspace_id: string;
  feature_id: string;
  run_id: string;
  title: string;
  decision: string;
  learning: string;
  adr_refs: string[];
  context_terms: string[];
  created_at: string;
}

export interface ListDecisionLogResponse {
  decisions: DecisionLogEntry[];
}
