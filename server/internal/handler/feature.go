package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
	"github.com/multica-ai/multica/server/pkg/protocol"
)

type FeatureResponse struct {
	ID           string  `json:"id"`
	WorkspaceID  string  `json:"workspace_id"`
	Title        string  `json:"title"`
	Description  *string `json:"description"`
	Icon         *string `json:"icon"`
	Status       string  `json:"status"`
	Priority     string  `json:"priority"`
	LeadType     *string `json:"lead_type"`
	LeadID       *string `json:"lead_id"`
	BranchSlug   *string `json:"branch_slug"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
	IssueCount   int64   `json:"issue_count"`
	DoneCount    int64   `json:"done_count"`
	// ResourceCount is a breadcrumb pointing at the sub-collection at
	// /api/features/{id}/resources. Resources themselves stay out of this
	// payload to keep parent metadata and child collections separate; clients
	// that need the list call ListProjectResources directly.
	ResourceCount int64 `json:"resource_count"`
}

func featureToResponse(p db.Feature) FeatureResponse {
	return FeatureResponse{
		ID:           uuidToString(p.ID),
		WorkspaceID:  uuidToString(p.WorkspaceID),
		Title:        p.Title,
		Description:  textToPtr(p.Description),
		Icon:         textToPtr(p.Icon),
		Status:       p.Status,
		Priority:     p.Priority,
		LeadType:     textToPtr(p.LeadType),
		LeadID:       uuidToPtr(p.LeadID),
		BranchSlug:   textToPtr(p.BranchSlug),
		CreatedAt:    timestampToString(p.CreatedAt),
		UpdatedAt:    timestampToString(p.UpdatedAt),
	}
}

func (h *Handler) loadFeatureIssueStats(ctx context.Context, projectID pgtype.UUID) (int64, int64) {
	stats, err := h.Queries.GetFeatureIssueStats(ctx, []pgtype.UUID{projectID})
	if err != nil || len(stats) == 0 {
		return 0, 0
	}
	return stats[0].TotalCount, stats[0].DoneCount
}

func (h *Handler) loadFeatureResourceCount(ctx context.Context, projectID pgtype.UUID) int64 {
	rows, err := h.Queries.GetFeatureResourceCounts(ctx, []pgtype.UUID{projectID})
	if err != nil || len(rows) == 0 {
		return 0
	}
	return rows[0].ResourceCount
}

type CreateFeatureRequest struct {
	Title        string                                `json:"title"`
	Description  *string                               `json:"description"`
	Icon         *string                               `json:"icon"`
	Status       string                                `json:"status"`
	Priority     string                                `json:"priority"`
	LeadType     *string                               `json:"lead_type"`
	LeadID       *string                               `json:"lead_id"`
	BranchSlug   *string                               `json:"branch_slug"`
	Resources    []CreateFeatureResourceRequestPayload `json:"resources,omitempty"`
}

// CreateFeatureResourceRequestPayload mirrors CreateProjectResourceRequest but
// is embedded inside the feature create payload. Kept as a separate type so a
// future change to the standalone request can't silently break this surface.
type CreateFeatureResourceRequestPayload struct {
	ResourceType string          `json:"resource_type"`
	ResourceRef  json.RawMessage `json:"resource_ref"`
	Label        *string         `json:"label"`
	Position     *int32          `json:"position"`
}

type UpdateFeatureRequest struct {
	Title        *string `json:"title"`
	Description  *string `json:"description"`
	Icon         *string `json:"icon"`
	Status       *string `json:"status"`
	Priority     *string `json:"priority"`
	LeadType     *string `json:"lead_type"`
	LeadID       *string `json:"lead_id"`
	BranchSlug   *string `json:"branch_slug"`
}

func (h *Handler) ListFeatures(w http.ResponseWriter, r *http.Request) {
	workspaceID := h.resolveWorkspaceID(r)
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace_id")
	if !ok {
		return
	}
	var statusFilter pgtype.Text
	if s := r.URL.Query().Get("status"); s != "" {
		statusFilter = pgtype.Text{String: s, Valid: true}
	}
	var priorityFilter pgtype.Text
	if p := r.URL.Query().Get("priority"); p != "" {
		priorityFilter = pgtype.Text{String: p, Valid: true}
	}
	features, err := h.Queries.ListFeatures(r.Context(), db.ListFeaturesParams{
		WorkspaceID: wsUUID,
		Status:      statusFilter,
		Priority:    priorityFilter,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list features")
		return
	}

	// Batch-fetch issue stats and resource counts for all projects
	statsMap := make(map[string]db.GetFeatureIssueStatsRow)
	resourceCountMap := make(map[string]int64)
	if len(features) > 0 {
		featureIDs := make([]pgtype.UUID, len(features))
		for i, p := range features {
			featureIDs[i] = p.ID
		}
		stats, err := h.Queries.GetFeatureIssueStats(r.Context(), featureIDs)
		if err == nil {
			for _, s := range stats {
				statsMap[uuidToString(s.FeatureID)] = s
			}
		}
		counts, err := h.Queries.GetFeatureResourceCounts(r.Context(), featureIDs)
		if err == nil {
			for _, c := range counts {
				resourceCountMap[uuidToString(c.FeatureID)] = c.ResourceCount
			}
		}
	}

	resp := make([]FeatureResponse, len(features))
	for i, p := range features {
		resp[i] = featureToResponse(p)
		if s, ok := statsMap[resp[i].ID]; ok {
			resp[i].IssueCount = s.TotalCount
			resp[i].DoneCount = s.DoneCount
		}
		resp[i].ResourceCount = resourceCountMap[resp[i].ID]
	}
	writeJSON(w, http.StatusOK, map[string]any{"features": resp, "total": len(resp)})
}

func (h *Handler) GetFeature(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	idUUID, ok := parseUUIDOrBadRequest(w, id, "feature id")
	if !ok {
		return
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return
	}
	feature, err := h.Queries.GetFeatureInWorkspace(r.Context(), db.GetFeatureInWorkspaceParams{
		ID: idUUID, WorkspaceID: wsUUID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "feature not found")
		return
	}
	resp := featureToResponse(feature)
	resp.IssueCount, resp.DoneCount = h.loadFeatureIssueStats(r.Context(), feature.ID)
	resp.ResourceCount = h.loadFeatureResourceCount(r.Context(), feature.ID)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateFeature(w http.ResponseWriter, r *http.Request) {
	var req CreateFeatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	workspaceID := h.resolveWorkspaceID(r)
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	status := req.Status
	if status == "" {
		status = "planned"
	}
	priority := req.Priority
	if priority == "" {
		priority = "none"
	}
	var leadType pgtype.Text
	var leadID pgtype.UUID
	if req.LeadType != nil {
		leadType = pgtype.Text{String: *req.LeadType, Valid: true}
	}
	if req.LeadID != nil {
		id, ok := parseUUIDOrBadRequest(w, *req.LeadID, "lead_id")
		if !ok {
			return
		}
		leadID = id
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace_id")
	if !ok {
		return
	}

	// Pre-validate every resource payload before opening a transaction so an
	// invalid ref produces a clean 400 with no DB work.
	normalizedRefs := make([]json.RawMessage, len(req.Resources))
	for i, res := range req.Resources {
		res.ResourceType = strings.TrimSpace(res.ResourceType)
		if res.ResourceType == "" {
			writeError(w, http.StatusBadRequest, "resources[].resource_type is required")
			return
		}
		ref, err := validateAndNormalizeResourceRef(res.ResourceType, res.ResourceRef)
		if err != nil {
			writeError(w, http.StatusBadRequest, "resources["+strconv.Itoa(i)+"]: "+err.Error())
			return
		}
		normalizedRefs[i] = ref
	}

	createParams := db.CreateFeatureParams{
		WorkspaceID:  wsUUID,
		Title:        req.Title,
		Description:  ptrToText(req.Description),
		Icon:         ptrToText(req.Icon),
		Status:       status,
		LeadType:     leadType,
		LeadID:       leadID,
		Priority:     priority,
		BranchSlug:   ptrToText(req.BranchSlug),
	}

	// Without resources, keep the simple non-tx path.
	if len(req.Resources) == 0 {
		feature, err := h.Queries.CreateFeature(r.Context(), createParams)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create feature")
			return
		}
		resp := featureToResponse(feature)
		h.publish(protocol.EventFeatureCreated, workspaceID, "member", userID, map[string]any{"feature": resp})
		writeJSON(w, http.StatusCreated, resp)
		return
	}

	// Transactional path: feature + all resources are atomic.
	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)

	feature, err := qtx.CreateFeature(r.Context(), createParams)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create feature")
		return
	}

	creator, _ := h.parseUserUUIDOrZero(userID)
	resourceRows := make([]db.FeatureResource, 0, len(req.Resources))
	for i, res := range req.Resources {
		var label pgtype.Text
		if res.Label != nil && strings.TrimSpace(*res.Label) != "" {
			label = pgtype.Text{String: strings.TrimSpace(*res.Label), Valid: true}
		}
		var position int32 = int32(i)
		if res.Position != nil {
			position = *res.Position
		}
		row, err := qtx.CreateFeatureResource(r.Context(), db.CreateFeatureResourceParams{
			FeatureID:    feature.ID,
			WorkspaceID:  feature.WorkspaceID,
			ResourceType: res.ResourceType,
			ResourceRef:  normalizedRefs[i],
			Label:        label,
			Position:     position,
			CreatedBy:    creator,
		})
		if err != nil {
			if isUniqueViolation(err) {
				writeError(w, http.StatusConflict, "resources["+strconv.Itoa(i)+"]: this resource is already attached")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to attach resource at index "+strconv.Itoa(i))
			return
		}
		resourceRows = append(resourceRows, row)
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit feature create")
		return
	}

	resourceResp := make([]FeatureResourceResponse, len(resourceRows))
	for i, row := range resourceRows {
		resourceResp[i] = featureResourceToResponse(row)
	}
	resp := featureToResponse(feature)
	resp.ResourceCount = int64(len(resourceResp))
	h.publish(protocol.EventFeatureCreated, workspaceID, "member", userID, map[string]any{"feature": resp})
	for _, rr := range resourceResp {
		h.publish(protocol.EventFeatureResourceCreated, workspaceID, "member", userID, map[string]any{
			"resource":   rr,
			"feature_id": resp.ID,
		})
	}
	// One-shot create echo: the parent FeatureResponse fields plus the just-
	// created resources. This is a transient creation echo, not a contract for
	// reads — GET /projects/{id} stays metadata-only with resource_count.
	writeJSON(w, http.StatusCreated, struct {
		FeatureResponse
		Resources []FeatureResourceResponse `json:"resources"`
	}{
		FeatureResponse: resp,
		Resources:       resourceResp,
	})
}

func (h *Handler) UpdateFeature(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	idUUID, ok := parseUUIDOrBadRequest(w, id, "feature id")
	if !ok {
		return
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return
	}
	prevFeature, err := h.Queries.GetFeatureInWorkspace(r.Context(), db.GetFeatureInWorkspaceParams{
		ID: idUUID, WorkspaceID: wsUUID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "feature not found")
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	var req UpdateFeatureRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var rawFields map[string]json.RawMessage
	json.Unmarshal(bodyBytes, &rawFields)

	params := db.UpdateFeatureParams{
		ID:           prevFeature.ID,
		Description:  prevFeature.Description,
		Icon:         prevFeature.Icon,
		LeadType:     prevFeature.LeadType,
		LeadID:       prevFeature.LeadID,
		BranchSlug:   prevFeature.BranchSlug,
	}
	if req.Title != nil {
		params.Title = pgtype.Text{String: *req.Title, Valid: true}
	}
	if req.Status != nil {
		params.Status = pgtype.Text{String: *req.Status, Valid: true}
	}
	if req.Priority != nil {
		params.Priority = pgtype.Text{String: *req.Priority, Valid: true}
	}
	if _, ok := rawFields["description"]; ok {
		if req.Description != nil {
			params.Description = pgtype.Text{String: *req.Description, Valid: true}
		} else {
			params.Description = pgtype.Text{Valid: false}
		}
	}
	if _, ok := rawFields["icon"]; ok {
		if req.Icon != nil {
			params.Icon = pgtype.Text{String: *req.Icon, Valid: true}
		} else {
			params.Icon = pgtype.Text{Valid: false}
		}
	}
	if _, ok := rawFields["lead_type"]; ok {
		if req.LeadType != nil {
			params.LeadType = pgtype.Text{String: *req.LeadType, Valid: true}
		} else {
			params.LeadType = pgtype.Text{Valid: false}
		}
	}
	if _, ok := rawFields["lead_id"]; ok {
		if req.LeadID != nil {
			leadUUID, ok := parseUUIDOrBadRequest(w, *req.LeadID, "lead_id")
			if !ok {
				return
			}
			params.LeadID = leadUUID
		} else {
			params.LeadID = pgtype.UUID{Valid: false}
		}
	}
	if _, ok := rawFields["branch_slug"]; ok {
		if req.BranchSlug != nil {
			params.BranchSlug = pgtype.Text{String: *req.BranchSlug, Valid: true}
		} else {
			params.BranchSlug = pgtype.Text{Valid: false}
		}
	}
	feature, err := h.Queries.UpdateFeature(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update feature")
		return
	}
	resp := featureToResponse(feature)
	resp.ResourceCount = h.loadFeatureResourceCount(r.Context(), feature.ID)
	h.publish(protocol.EventFeatureUpdated, workspaceID, "member", userID, map[string]any{"feature": resp})
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) DeleteFeature(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	idUUID, ok := parseUUIDOrBadRequest(w, id, "feature id")
	if !ok {
		return
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return
	}
	feature, err := h.Queries.GetFeatureInWorkspace(r.Context(), db.GetFeatureInWorkspaceParams{
		ID: idUUID, WorkspaceID: wsUUID,
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "feature not found")
		return
	}
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}
	if err := h.Queries.DeleteFeature(r.Context(), db.DeleteFeatureParams{
		ID:          feature.ID,
		WorkspaceID: feature.WorkspaceID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete feature")
		return
	}
	h.publish(protocol.EventFeatureDeleted, workspaceID, "member", userID, map[string]any{"feature_id": uuidToString(feature.ID)})
	w.WriteHeader(http.StatusNoContent)
}

// SearchFeatureResponse extends FeatureResponse with search metadata.
type SearchFeatureResponse struct {
	FeatureResponse
	MatchSource    string  `json:"match_source"`
	MatchedSnippet *string `json:"matched_snippet,omitempty"`
}

// buildFeatureSearchQuery builds a dynamic SQL query for feature search.
func buildFeatureSearchQuery(phrase string, terms []string, includeClosed bool) (string, []any) {
	phrase = strings.ToLower(phrase)
	for i, t := range terms {
		terms[i] = strings.ToLower(t)
	}

	argIdx := 1
	args := []any{}
	nextArg := func(val any) string {
		args = append(args, val)
		s := fmt.Sprintf("$%d", argIdx)
		argIdx++
		return s
	}

	escapedPhrase := escapeLike(phrase)
	phraseParam := nextArg(escapedPhrase)
	phraseContains := "'%' || " + phraseParam + " || '%'"
	phraseStartsWith := phraseParam + " || '%'"

	wsParam := nextArg(nil) // workspace_id placeholder

	var termParams []string
	if len(terms) > 1 {
		for _, t := range terms {
			et := escapeLike(t)
			termParams = append(termParams, nextArg(et))
		}
	}

	// --- WHERE clause ---
	var whereParts []string

	// Full phrase match: title or description
	phraseMatch := fmt.Sprintf(
		"(LOWER(p.title) LIKE %s OR LOWER(COALESCE(p.description, '')) LIKE %s)",
		phraseContains, phraseContains,
	)
	whereParts = append(whereParts, phraseMatch)

	// Multi-word AND match
	if len(termParams) > 1 {
		var termConditions []string
		for _, tp := range termParams {
			tc := "'%' || " + tp + " || '%'"
			termConditions = append(termConditions, fmt.Sprintf(
				"(LOWER(p.title) LIKE %s OR LOWER(COALESCE(p.description, '')) LIKE %s)",
				tc, tc,
			))
		}
		whereParts = append(whereParts, "("+strings.Join(termConditions, " AND ")+")")
	}

	whereClause := "(" + strings.Join(whereParts, " OR ") + ")"

	if !includeClosed {
		whereClause += " AND p.status NOT IN ('completed', 'cancelled')"
	}

	// --- ORDER BY ranking ---
	var rankCases []string

	// Tier 0: Exact title match
	rankCases = append(rankCases, fmt.Sprintf("WHEN LOWER(p.title) = %s THEN 0", phraseParam))

	// Tier 1: Title starts with phrase
	rankCases = append(rankCases, fmt.Sprintf("WHEN LOWER(p.title) LIKE %s THEN 1", phraseStartsWith))

	// Tier 2: Title contains phrase
	rankCases = append(rankCases, fmt.Sprintf("WHEN LOWER(p.title) LIKE %s THEN 2", phraseContains))

	// Tier 3: Title matches all words (multi-word only)
	if len(termParams) > 1 {
		var titleTerms []string
		for _, tp := range termParams {
			titleTerms = append(titleTerms, fmt.Sprintf("LOWER(p.title) LIKE '%s' || %s || '%s'", "%", tp, "%"))
		}
		rankCases = append(rankCases, fmt.Sprintf("WHEN (%s) THEN 3", strings.Join(titleTerms, " AND ")))
	}

	// Tier 4: Description contains phrase
	rankCases = append(rankCases, fmt.Sprintf("WHEN LOWER(COALESCE(p.description, '')) LIKE %s THEN 4", phraseContains))

	rankExpr := "CASE " + strings.Join(rankCases, " ") + " ELSE 5 END"

	// --- match_source expression ---
	matchSourceExpr := fmt.Sprintf(`CASE
		WHEN LOWER(p.title) LIKE %s THEN 'title'
		ELSE 'description'
	END`, phraseContains)

	if len(termParams) > 1 {
		var titleTerms []string
		for _, tp := range termParams {
			titleTerms = append(titleTerms, fmt.Sprintf("LOWER(p.title) LIKE '%s' || %s || '%s'", "%", tp, "%"))
		}
		matchSourceExpr = fmt.Sprintf(`CASE
			WHEN LOWER(p.title) LIKE %s THEN 'title'
			WHEN (%s) THEN 'title'
			ELSE 'description'
		END`,
			phraseContains, strings.Join(titleTerms, " AND "),
		)
	}

	limitParam := nextArg(nil)
	offsetParam := nextArg(nil)

	query := fmt.Sprintf(`SELECT p.id, p.workspace_id, p.title, p.description, p.icon,
		p.status, p.priority, p.lead_type, p.lead_id,
		p.created_at, p.updated_at, p.branch_slug,
		COUNT(*) OVER() AS total_count,
		%s AS match_source
	FROM feature p
	WHERE p.workspace_id = %s AND %s
	ORDER BY %s, p.updated_at DESC
	LIMIT %s OFFSET %s`,
		matchSourceExpr,
		wsParam,
		whereClause,
		rankExpr,
		limitParam,
		offsetParam,
	)

	return query, args
}

func (h *Handler) SearchFeatures(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	workspaceID := h.resolveWorkspaceID(r)

	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "q parameter is required")
		return
	}

	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 50 {
		limit = 50
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	includeClosed := r.URL.Query().Get("include_closed") == "true"

	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace_id")
	if !ok {
		return
	}
	terms := splitSearchTerms(q)

	sqlQuery, args := buildFeatureSearchQuery(q, terms, includeClosed)
	args[1] = wsUUID
	args[len(args)-2] = limit
	args[len(args)-1] = offset

	rows, err := h.DB.Query(ctx, sqlQuery, args...)
	if err != nil {
		slog.Warn("search features failed", "error", err, "workspace_id", workspaceID, "query", q)
		writeError(w, http.StatusInternalServerError, "failed to search features")
		return
	}
	defer rows.Close()

	type featureSearchRow struct {
		feature     db.Feature
		totalCount  int64
		matchSource string
	}

	var results []featureSearchRow
	for rows.Next() {
		var row featureSearchRow
		if err := rows.Scan(
			&row.feature.ID,
			&row.feature.WorkspaceID,
			&row.feature.Title,
			&row.feature.Description,
			&row.feature.Icon,
			&row.feature.Status,
			&row.feature.Priority,
			&row.feature.LeadType,
			&row.feature.LeadID,
			&row.feature.CreatedAt,
			&row.feature.UpdatedAt,
			&row.feature.BranchSlug,
			&row.totalCount,
			&row.matchSource,
		); err != nil {
			slog.Warn("search features scan failed", "error", err)
			writeError(w, http.StatusInternalServerError, "failed to search features")
			return
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("search features rows error", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to search features")
		return
	}

	var total int64
	if len(results) > 0 {
		total = results[0].totalCount
	}

	// Batch-fetch issue stats and resource counts
	statsMap := make(map[string]db.GetFeatureIssueStatsRow)
	resourceCountMap := make(map[string]int64)
	if len(results) > 0 {
		featureIDs := make([]pgtype.UUID, len(results))
		for i, r := range results {
			featureIDs[i] = r.feature.ID
		}
		stats, err := h.Queries.GetFeatureIssueStats(ctx, featureIDs)
		if err == nil {
			for _, s := range stats {
				statsMap[uuidToString(s.FeatureID)] = s
			}
		}
		counts, err := h.Queries.GetFeatureResourceCounts(ctx, featureIDs)
		if err == nil {
			for _, c := range counts {
				resourceCountMap[uuidToString(c.FeatureID)] = c.ResourceCount
			}
		}
	}

	resp := make([]SearchFeatureResponse, len(results))
	for i, row := range results {
		pr := featureToResponse(row.feature)
		if s, ok := statsMap[pr.ID]; ok {
			pr.IssueCount = s.TotalCount
			pr.DoneCount = s.DoneCount
		}
		pr.ResourceCount = resourceCountMap[pr.ID]
		spr := SearchFeatureResponse{
			FeatureResponse: pr,
			MatchSource:     row.matchSource,
		}
		if row.matchSource == "description" {
			desc := ""
			if row.feature.Description.Valid {
				desc = row.feature.Description.String
			}
			if desc != "" {
				snippet := extractSnippet(desc, q)
				spr.MatchedSnippet = &snippet
			}
		}
		resp[i] = spr
	}

	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	writeJSON(w, http.StatusOK, map[string]any{
		"features": resp,
		"total":    total,
	})
}

// IssueSummary is a compact issue representation used in feature issue groupings.
type IssueSummary struct {
	ID         string  `json:"id"`
	Identifier string  `json:"identifier"`
	Title      string  `json:"title"`
	Status     string  `json:"status"`
	Priority   string  `json:"priority"`
	RepoID     *string `json:"repo_id,omitempty"`
	RepoName   *string `json:"repo_name,omitempty"`
}

// BlockedIssueSummary extends IssueSummary with identifiers of unsatisfied blockers.
type BlockedIssueSummary struct {
	IssueSummary
	BlockedBy []string `json:"blocked_by"`
}

// PRSummary is a compact pull-request representation used in feature issue groupings.
type PRSummary struct {
	Number  int32   `json:"number"`
	HtmlURL string  `json:"html_url"`
	State   string  `json:"state"`
	Title   string  `json:"title"`
	RepoID  *string `json:"repo_id,omitempty"`
}

// FeatureIssuesResponse groups a feature's child issues into ready and blocked sets,
// plus any GitHub PRs linked to those issues.
type FeatureIssuesResponse struct {
	ReadyNow     []IssueSummary        `json:"ready_now"`
	Blocked      []BlockedIssueSummary `json:"blocked"`
	PullRequests []PRSummary           `json:"pull_requests"`
}

// GetFeatureIssues returns a feature's child issues grouped by dependency readiness.
// Issues with unsatisfied 'blocks'/'blocked_by' dependencies are in the blocked set;
// all others (including done issues) are in the ready set. The 'related' dependency
// type is non-gating and does not affect grouping.
func (h *Handler) GetFeatureIssues(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workspaceID := h.resolveWorkspaceID(r)
	idUUID, ok := parseUUIDOrBadRequest(w, id, "feature id")
	if !ok {
		return
	}
	wsUUID, ok := parseUUIDOrBadRequest(w, workspaceID, "workspace id")
	if !ok {
		return
	}
	if _, err := h.Queries.GetFeatureInWorkspace(r.Context(), db.GetFeatureInWorkspaceParams{
		ID: idUUID, WorkspaceID: wsUUID,
	}); err != nil {
		writeError(w, http.StatusNotFound, "feature not found")
		return
	}

	issues, err := h.Queries.ListIssues(r.Context(), db.ListIssuesParams{
		WorkspaceID: wsUUID,
		FeatureID:   idUUID,
		Limit:       1000,
		Offset:      0,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list feature issues")
		return
	}

	if len(issues) == 0 {
		writeJSON(w, http.StatusOK, FeatureIssuesResponse{
			ReadyNow:     []IssueSummary{},
			Blocked:      []BlockedIssueSummary{},
			PullRequests: []PRSummary{},
		})
		return
	}

	prefix := h.getIssuePrefix(r.Context(), wsUUID)

	issueIDs := make([]string, len(issues))
	for i, iss := range issues {
		issueIDs[i] = uuidToString(iss.ID)
	}

	blockedByMap := h.loadBlockedByMap(r.Context(), issueIDs)
	repoMap := h.loadIssueRepoMap(r.Context(), issueIDs)

	var readyNow []IssueSummary
	var blocked []BlockedIssueSummary
	for _, iss := range issues {
		sum := IssueSummary{
			ID:         uuidToString(iss.ID),
			Identifier: prefix + "-" + strconv.Itoa(int(iss.Number)),
			Title:      iss.Title,
			Status:     iss.Status,
			Priority:   iss.Priority,
		}
		if info, ok := repoMap[sum.ID]; ok {
			id, name := info[0], info[1]
			sum.RepoID = &id
			sum.RepoName = &name
		}
		if blockers, isBlocked := blockedByMap[sum.ID]; isBlocked {
			blocked = append(blocked, BlockedIssueSummary{IssueSummary: sum, BlockedBy: blockers})
		} else {
			readyNow = append(readyNow, sum)
		}
	}

	if readyNow == nil {
		readyNow = []IssueSummary{}
	}
	if blocked == nil {
		blocked = []BlockedIssueSummary{}
	}
	prs := h.loadFeaturePRs(r.Context(), issueIDs)
	if prs == nil {
		prs = []PRSummary{}
	}
	writeJSON(w, http.StatusOK, FeatureIssuesResponse{ReadyNow: readyNow, Blocked: blocked, PullRequests: prs})
}

// loadFeaturePRs returns all GitHub PRs linked to any issue in issueIDs.
func (h *Handler) loadFeaturePRs(ctx context.Context, issueIDs []string) []PRSummary {
	if len(issueIDs) == 0 || h.DB == nil {
		return nil
	}
	rows, err := h.DB.Query(ctx, `
		SELECT DISTINCT gpr.pr_number, gpr.html_url, gpr.state, gpr.title, gpr.repo_id::text
		FROM issue_pull_request ipr
		JOIN github_pull_request gpr ON ipr.pull_request_id = gpr.id
		WHERE ipr.issue_id = ANY($1::uuid[])
		ORDER BY gpr.pr_number
	`, issueIDs)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []PRSummary
	for rows.Next() {
		var s PRSummary
		var repoID *string
		if err := rows.Scan(&s.Number, &s.HtmlURL, &s.State, &s.Title, &repoID); err == nil {
			s.RepoID = repoID
			result = append(result, s)
		}
	}
	return result
}

// loadBlockedByMap returns, for each issueID in the input set that has at least
// one unsatisfied blocking dependency, a slice of blocker identifiers.
// Issues with only 'related' dependencies are never included (non-gating).
func (h *Handler) loadBlockedByMap(ctx context.Context, issueIDs []string) map[string][]string {
	if len(issueIDs) == 0 || h.DB == nil {
		return nil
	}
	rows, err := h.DB.Query(ctx, `
		SELECT d.issue_id::text, w.issue_prefix || '-' || b.number::text AS blocker_identifier
		FROM issue_dependency d
		JOIN issue b ON d.depends_on_issue_id = b.id
		JOIN workspace w ON b.workspace_id = w.id
		WHERE d.issue_id = ANY($1::uuid[])
		  AND d.type IN ('blocks', 'blocked_by')
		  AND b.status != 'done'
	`, issueIDs)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := make(map[string][]string)
	for rows.Next() {
		var issID, blockerIdentifier string
		if err := rows.Scan(&issID, &blockerIdentifier); err == nil {
			result[issID] = append(result[issID], blockerIdentifier)
		}
	}
	return result
}

// loadIssueRepoMap returns a map from issue ID to {repoID, repoName} for issues that have a repo.
func (h *Handler) loadIssueRepoMap(ctx context.Context, issueIDs []string) map[string][2]string {
	if len(issueIDs) == 0 || h.DB == nil {
		return nil
	}
	rows, err := h.DB.Query(ctx, `
		SELECT i.id::text, r.id::text, r.name
		FROM issue i
		JOIN repo r ON r.id = i.repo_id
		WHERE i.id = ANY($1::uuid[])
		  AND i.repo_id IS NOT NULL
	`, issueIDs)
	if err != nil {
		return nil
	}
	defer rows.Close()
	result := make(map[string][2]string)
	for rows.Next() {
		var issID, repoID, repoName string
		if err := rows.Scan(&issID, &repoID, &repoName); err == nil {
			result[issID] = [2]string{repoID, repoName}
		}
	}
	return result
}
