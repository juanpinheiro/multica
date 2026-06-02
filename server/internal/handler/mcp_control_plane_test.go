package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The control-plane (MCP / UI) write surface: Milestone create/update, DoD
// assertion create, Initiative status-transition guarding, and Issue creation
// with a Milestone. These exercise the handlers added for issue 14.

func decodeJSON(t *testing.T, w *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
		t.Fatalf("decode response %q: %v", w.Body.String(), err)
	}
}

func TestCreateMilestone_AppendsAtEnd(t *testing.T) {
	f := newInitiativeGateFixture(t)
	featureID := f.makeFeature("draft")

	create := func() MilestoneResponse {
		w := httptest.NewRecorder()
		req := newRequest("POST", "/api/milestones", map[string]any{
			"feature_id": featureID,
			"title":      "A milestone",
		})
		testHandler.CreateMilestone(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("CreateMilestone: expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp MilestoneResponse
		decodeJSON(t, w, &resp)
		t.Cleanup(func() { testPool.Exec(f.ctx, `DELETE FROM milestone WHERE id = $1`, resp.ID) })
		return resp
	}

	first := create()
	if first.Position != 0 {
		t.Errorf("first milestone position = %d, want 0", first.Position)
	}
	if first.ValidationStatus != "pending" {
		t.Errorf("validation_status = %q, want pending", first.ValidationStatus)
	}
	second := create()
	if second.Position != 1 {
		t.Errorf("second milestone position = %d, want 1 (appended)", second.Position)
	}
}

func TestCreateMilestone_RejectsForeignFeature(t *testing.T) {
	newInitiativeGateFixture(t)
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/milestones", map[string]any{
		"feature_id": testWorkspaceID, // a valid UUID that is not a feature in this workspace
		"title":      "Orphan",
	})
	testHandler.CreateMilestone(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown feature, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateMilestone_ChangesTitle(t *testing.T) {
	f := newInitiativeGateFixture(t)
	featureID := f.makeFeature("draft")
	milestoneID := f.makeMilestone(featureID, 0, "pending")

	w := httptest.NewRecorder()
	req := newRequest("PATCH", "/api/milestones/"+milestoneID, map[string]any{"title": "Renamed"})
	req = withURLParam(req, "id", milestoneID)
	testHandler.UpdateMilestone(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateMilestone: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp MilestoneResponse
	decodeJSON(t, w, &resp)
	if resp.Title != "Renamed" {
		t.Errorf("title = %q, want Renamed", resp.Title)
	}
}

func TestCreateDodAssertion_PersistsUnderMilestone(t *testing.T) {
	f := newInitiativeGateFixture(t)
	featureID := f.makeFeature("draft")
	milestoneID := f.makeMilestone(featureID, 0, "pending")

	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/milestones/"+milestoneID+"/dod", map[string]any{
		"text": "The build passes",
	})
	req = withURLParam(req, "id", milestoneID)
	testHandler.CreateDodAssertion(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateDodAssertion: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp dodAssertionResponse
	decodeJSON(t, w, &resp)
	if resp.Text != "The build passes" {
		t.Errorf("text = %q, want 'The build passes'", resp.Text)
	}
	if resp.MilestoneID != milestoneID {
		t.Errorf("milestone_id = %q, want %q", resp.MilestoneID, milestoneID)
	}
	if resp.Status != "pending" {
		t.Errorf("status = %q, want pending (no verdict yet)", resp.Status)
	}
}

func TestUpdateFeature_LegalStatusTransition(t *testing.T) {
	f := newInitiativeGateFixture(t)
	featureID := f.makeFeature("draft")

	w := httptest.NewRecorder()
	req := newRequest("PATCH", "/api/features/"+featureID, map[string]any{"status": "ready"})
	req = withURLParam(req, "id", featureID)
	testHandler.UpdateFeature(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for draft→ready, got %d: %s", w.Code, w.Body.String())
	}
	if got := f.featureStatus(featureID); got != "ready" {
		t.Errorf("status = %q, want ready", got)
	}
}

func TestUpdateFeature_RejectsIllegalTransition(t *testing.T) {
	f := newInitiativeGateFixture(t)
	featureID := f.makeFeature("draft")

	w := httptest.NewRecorder()
	req := newRequest("PATCH", "/api/features/"+featureID, map[string]any{"status": "done"})
	req = withURLParam(req, "id", featureID)
	testHandler.UpdateFeature(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for draft→done, got %d: %s", w.Code, w.Body.String())
	}
	if got := f.featureStatus(featureID); got != "draft" {
		t.Errorf("status = %q, want draft (unchanged)", got)
	}
}

func TestUpdateFeature_RejectsUnknownStatus(t *testing.T) {
	f := newInitiativeGateFixture(t)
	featureID := f.makeFeature("draft")

	w := httptest.NewRecorder()
	req := newRequest("PATCH", "/api/features/"+featureID, map[string]any{"status": "not-a-status"})
	req = withURLParam(req, "id", featureID)
	testHandler.UpdateFeature(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown status, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateFeature_PersistsModeAndBudget(t *testing.T) {
	newInitiativeGateFixture(t)
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/features", map[string]any{
		"title":             "Budgeted Initiative",
		"mode":              "afk",
		"budget_runs":       5,
		"failure_tolerance": 2,
	})
	testHandler.CreateFeature(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateFeature: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp FeatureResponse
	decodeJSON(t, w, &resp)
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM feature WHERE id = $1`, resp.ID) })
	if resp.Mode != "afk" {
		t.Errorf("mode = %q, want afk", resp.Mode)
	}
	if resp.BudgetRuns != 5 {
		t.Errorf("budget_runs = %d, want 5", resp.BudgetRuns)
	}
	if resp.FailureTolerance != 2 {
		t.Errorf("failure_tolerance = %d, want 2", resp.FailureTolerance)
	}
}

func TestCreateFeature_RejectsInvalidMode(t *testing.T) {
	newInitiativeGateFixture(t)
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/features", map[string]any{
		"title": "Bad mode",
		"mode":  "turbo",
	})
	testHandler.CreateFeature(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid mode, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateIssue_WithMilestone(t *testing.T) {
	f := newInitiativeGateFixture(t)
	featureID := f.makeFeature("draft")
	milestoneID := f.makeMilestone(featureID, 0, "pending")

	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/issues", map[string]any{
		"title":        "Milestone-scoped issue",
		"feature_id":   featureID,
		"milestone_id": milestoneID,
	})
	testHandler.CreateIssue(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateIssue: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp IssueResponse
	decodeJSON(t, w, &resp)
	t.Cleanup(func() { testPool.Exec(context.Background(), `DELETE FROM issue WHERE id = $1`, resp.ID) })
	if resp.MilestoneID == nil || *resp.MilestoneID != milestoneID {
		t.Errorf("milestone_id = %v, want %q", resp.MilestoneID, milestoneID)
	}
}

func TestCreateIssue_RejectsForeignMilestone(t *testing.T) {
	f := newInitiativeGateFixture(t)
	featureID := f.makeFeature("draft")

	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/issues", map[string]any{
		"title":        "Bad milestone",
		"feature_id":   featureID,
		"milestone_id": testWorkspaceID, // valid UUID, not a milestone
	})
	testHandler.CreateIssue(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown milestone, got %d: %s", w.Code, w.Body.String())
	}
}
