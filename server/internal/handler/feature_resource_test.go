package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFeatureResourceLifecycle(t *testing.T) {
	// Create a feature to attach resources to.
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/features?workspace_id="+testWorkspaceID, map[string]any{
		"title": "Resource lifecycle project",
	})
	testHandler.CreateFeature(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateProject: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var feature FeatureResponse
	if err := json.NewDecoder(w.Body).Decode(&feature); err != nil {
		t.Fatalf("decode CreateProject: %v", err)
	}
	defer func() {
		req := newRequest("DELETE", "/api/features/"+feature.ID, nil)
		req = withURLParam(req, "id", feature.ID)
		testHandler.DeleteFeature(httptest.NewRecorder(), req)
	}()

	// Attach a github_repo resource.
	w = httptest.NewRecorder()
	req = newRequest("POST", "/api/features/"+feature.ID+"/resources", map[string]any{
		"resource_type": "github_repo",
		"resource_ref":  map[string]any{"url": "https://github.com/multica-ai/multica"},
	})
	req = withURLParam(req, "id", feature.ID)
	testHandler.CreateFeatureResource(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateProjectResource: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created FeatureResourceResponse
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("decode CreateProjectResource: %v", err)
	}
	if created.ResourceType != "github_repo" {
		t.Errorf("created.ResourceType = %q, want github_repo", created.ResourceType)
	}
	var ref struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(created.ResourceRef, &ref); err != nil {
		t.Fatalf("decode resource_ref: %v", err)
	}
	if ref.URL != "https://github.com/multica-ai/multica" {
		t.Errorf("created.ResourceRef.url = %q", ref.URL)
	}

	// Listing must include the new resource.
	w = httptest.NewRecorder()
	req = newRequest("GET", "/api/features/"+feature.ID+"/resources", nil)
	req = withURLParam(req, "id", feature.ID)
	testHandler.ListFeatureResources(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListProjectResources: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var listResp struct {
		Resources []FeatureResourceResponse `json:"resources"`
		Total     int                       `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if listResp.Total != 1 || len(listResp.Resources) != 1 {
		t.Fatalf("list returned %d resources, want 1", listResp.Total)
	}
	if listResp.Resources[0].ID != created.ID {
		t.Errorf("list[0].ID = %q, want %q", listResp.Resources[0].ID, created.ID)
	}

	// Duplicate attach must conflict (UNIQUE on feature_id + type + ref).
	w = httptest.NewRecorder()
	req = newRequest("POST", "/api/features/"+feature.ID+"/resources", map[string]any{
		"resource_type": "github_repo",
		"resource_ref":  map[string]any{"url": "https://github.com/multica-ai/multica"},
	})
	req = withURLParam(req, "id", feature.ID)
	testHandler.CreateFeatureResource(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("duplicate CreateProjectResource: expected 409, got %d: %s", w.Code, w.Body.String())
	}

	// Invalid URL must reject at the validator level.
	w = httptest.NewRecorder()
	req = newRequest("POST", "/api/features/"+feature.ID+"/resources", map[string]any{
		"resource_type": "github_repo",
		"resource_ref":  map[string]any{"url": "not-a-url"},
	})
	req = withURLParam(req, "id", feature.ID)
	testHandler.CreateFeatureResource(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid URL: expected 400, got %d: %s", w.Code, w.Body.String())
	}

	// Unknown resource_type must reject.
	w = httptest.NewRecorder()
	req = newRequest("POST", "/api/features/"+feature.ID+"/resources", map[string]any{
		"resource_type": "unknown_type",
		"resource_ref":  map[string]any{"foo": "bar"},
	})
	req = withURLParam(req, "id", feature.ID)
	testHandler.CreateFeatureResource(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("unknown type: expected 400, got %d: %s", w.Code, w.Body.String())
	}

	// Delete the resource.
	w = httptest.NewRecorder()
	req = newRequest("DELETE", "/api/features/"+feature.ID+"/resources/"+created.ID, nil)
	req = withURLParams(req, "id", feature.ID, "resourceId", created.ID)
	testHandler.DeleteFeatureResource(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("DeleteProjectResource: expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// After deletion the list should be empty.
	w = httptest.NewRecorder()
	req = newRequest("GET", "/api/features/"+feature.ID+"/resources", nil)
	req = withURLParam(req, "id", feature.ID)
	testHandler.ListFeatureResources(w, req)
	if err := json.NewDecoder(w.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode post-delete list: %v", err)
	}
	if listResp.Total != 0 {
		t.Errorf("post-delete list: total = %d, want 0", listResp.Total)
	}
}

// TestProjectResourceAcceptsSSHRepoURLs covers GitHub issue #2484: SSH and
// scp-like git URLs must be accepted alongside https URLs, because workspace
// repos configured with an SSH remote previously got rejected when attached
// to a feature.
func TestProjectResourceAcceptsSSHRepoURLs(t *testing.T) {
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/features?workspace_id="+testWorkspaceID, map[string]any{
		"title": "SSH repo URL acceptance",
	})
	testHandler.CreateFeature(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateProject: %d %s", w.Code, w.Body.String())
	}
	var feature FeatureResponse
	if err := json.NewDecoder(w.Body).Decode(&feature); err != nil {
		t.Fatalf("decode CreateProject: %v", err)
	}
	defer func() {
		r := newRequest("DELETE", "/api/features/"+feature.ID, nil)
		r = withURLParam(r, "id", feature.ID)
		testHandler.DeleteFeature(httptest.NewRecorder(), r)
	}()

	cases := []struct {
		name string
		url  string
	}{
		{"scp-like", "git@github.com:multica-ai/multica.git"},
		{"ssh-scheme", "ssh://git@github.com/multica-ai/multica.git"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := newRequest("POST", "/api/features/"+feature.ID+"/resources", map[string]any{
				"resource_type": "github_repo",
				"resource_ref":  map[string]any{"url": tc.url},
			})
			req = withURLParam(req, "id", feature.ID)
			testHandler.CreateFeatureResource(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("CreateProjectResource(%s): expected 201, got %d: %s", tc.url, w.Code, w.Body.String())
			}
			var created FeatureResourceResponse
			if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
				t.Fatalf("decode: %v", err)
			}
			var ref struct {
				URL string `json:"url"`
			}
			if err := json.Unmarshal(created.ResourceRef, &ref); err != nil {
				t.Fatalf("decode resource_ref: %v", err)
			}
			if ref.URL != tc.url {
				t.Errorf("ref.url = %q, want %q", ref.URL, tc.url)
			}
		})
	}
}

func TestIsValidGitRepoURL(t *testing.T) {
	good := []string{
		"https://github.com/multica-ai/multica",
		"https://github.com/multica-ai/multica.git",
		"http://github.example.com/x/y",
		"ssh://git@github.com/multica-ai/multica.git",
		"ssh://git@github.com:22/multica-ai/multica.git",
		"git@github.com:multica-ai/multica.git",
		"git@gitlab.example.com:group/sub/repo.git",
	}
	bad := []string{
		"",
		"not-a-url",
		"github.com/multica-ai/multica", // no scheme, no scp-style colon
		"https://",                      // empty host
		"git@github.com",                // missing :path
		"git@:foo/bar",                  // missing host
		"git@github.com:",               // missing path
		"ftp://example.com/repo",        // unsupported scheme
		"file:///tmp/repo",              // unsupported scheme
		"some random text with spaces",
		"github.com:org/repo@branch",    // '@' after ':' belongs to the path, not user
		"foo:bar@baz",                   // '@' after ':' with no scheme
		":foo/bar",                      // leading ':' with no host
	}
	for _, s := range good {
		if !isValidGitRepoURL(s) {
			t.Errorf("isValidGitRepoURL(%q) = false, want true", s)
		}
	}
	for _, s := range bad {
		if isValidGitRepoURL(s) {
			t.Errorf("isValidGitRepoURL(%q) = true, want false", s)
		}
	}
}

func TestCreateProjectAttachesResources(t *testing.T) {
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/features?workspace_id="+testWorkspaceID, map[string]any{
		"title": "Project with bundled resources",
		"resources": []map[string]any{
			{
				"resource_type": "github_repo",
				"resource_ref":  map[string]any{"url": "https://github.com/multica-ai/multica"},
			},
		},
	})
	testHandler.CreateFeature(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateProject with resources: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		ID        string                    `json:"id"`
		Resources []FeatureResourceResponse `json:"resources"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	defer func() {
		r := newRequest("DELETE", "/api/features/"+resp.ID, nil)
		r = withURLParam(r, "id", resp.ID)
		testHandler.DeleteFeature(httptest.NewRecorder(), r)
	}()

	if len(resp.Resources) != 1 || resp.Resources[0].ResourceType != "github_repo" {
		t.Fatalf("response resources mismatch: %+v", resp.Resources)
	}
}

// TestProjectResourceCountBreadcrumb asserts the resource_count breadcrumb
// surfaces on GetFeature and ListFeatures so agents know to call
// /api/features/{id}/resources without inlining the sub-collection.
func TestProjectResourceCountBreadcrumb(t *testing.T) {
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/features?workspace_id="+testWorkspaceID, map[string]any{
		"title": "Resource count breadcrumb",
	})
	testHandler.CreateFeature(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateProject: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var feature FeatureResponse
	if err := json.NewDecoder(w.Body).Decode(&feature); err != nil {
		t.Fatalf("decode CreateProject: %v", err)
	}
	defer func() {
		r := newRequest("DELETE", "/api/features/"+feature.ID, nil)
		r = withURLParam(r, "id", feature.ID)
		testHandler.DeleteFeature(httptest.NewRecorder(), r)
	}()

	getCount := func() int64 {
		w := httptest.NewRecorder()
		req := newRequest("GET", "/api/features/"+feature.ID, nil)
		req = withURLParam(req, "id", feature.ID)
		testHandler.GetFeature(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GetFeature: %d %s", w.Code, w.Body.String())
		}
		var resp FeatureResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode GetFeature: %v", err)
		}
		return resp.ResourceCount
	}
	if got := getCount(); got != 0 {
		t.Errorf("initial GetFeature ResourceCount = %d, want 0", got)
	}

	w = httptest.NewRecorder()
	req = newRequest("POST", "/api/features/"+feature.ID+"/resources", map[string]any{
		"resource_type": "github_repo",
		"resource_ref":  map[string]any{"url": "https://github.com/multica-ai/breadcrumb"},
	})
	req = withURLParam(req, "id", feature.ID)
	testHandler.CreateFeatureResource(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateProjectResource: %d %s", w.Code, w.Body.String())
	}

	if got := getCount(); got != 1 {
		t.Errorf("after attach GetFeature ResourceCount = %d, want 1", got)
	}

	w = httptest.NewRecorder()
	req = newRequest("GET", "/api/features?workspace_id="+testWorkspaceID, nil)
	testHandler.ListFeatures(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListFeatures: %d %s", w.Code, w.Body.String())
	}
	var list struct {
		Features []FeatureResponse `json:"features"`
	}
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
		t.Fatalf("decode ListFeatures: %v", err)
	}
	found := false
	for _, p := range list.Features {
		if p.ID == feature.ID {
			found = true
			if p.ResourceCount != 1 {
				t.Errorf("ListFeatures[%s].ResourceCount = %d, want 1", p.ID, p.ResourceCount)
			}
			break
		}
	}
	if !found {
		t.Fatalf("feature %s not found in ListFeatures response", feature.ID)
	}

	// UpdateProject must preserve the breadcrumb. A title-only PUT used to
	// reset resource_count to 0 because UpdateProject didn't reload the count.
	w = httptest.NewRecorder()
	req = newRequest("PUT", "/api/features/"+feature.ID, map[string]any{
		"title": "Resource count breadcrumb (updated)",
	})
	req = withURLParam(req, "id", feature.ID)
	testHandler.UpdateFeature(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("UpdateFeature: %d %s", w.Code, w.Body.String())
	}
	var updated FeatureResponse
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatalf("decode UpdateFeature: %v", err)
	}
	if updated.ResourceCount != 1 {
		t.Errorf("UpdateProject ResourceCount = %d, want 1", updated.ResourceCount)
	}
}

// TestCreateProjectWithResourcesEchoesCount asserts the create-with-resources
// echo carries resource_count matching the attached resources, so the HTTP
// response and the published project:created event agree.
func TestCreateProjectWithResourcesEchoesCount(t *testing.T) {
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/features?workspace_id="+testWorkspaceID, map[string]any{
		"title": "Create echo with resource_count",
		"resources": []map[string]any{
			{
				"resource_type": "github_repo",
				"resource_ref":  map[string]any{"url": "https://github.com/multica-ai/echo-count"},
			},
		},
	})
	testHandler.CreateFeature(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreateProject with resources: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		ID            string                    `json:"id"`
		ResourceCount int64                     `json:"resource_count"`
		Resources     []FeatureResourceResponse `json:"resources"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode CreateProject: %v", err)
	}
	defer func() {
		r := newRequest("DELETE", "/api/features/"+resp.ID, nil)
		r = withURLParam(r, "id", resp.ID)
		testHandler.DeleteFeature(httptest.NewRecorder(), r)
	}()
	if resp.ResourceCount != 1 || len(resp.Resources) != 1 {
		t.Errorf("CreateProject echo: resource_count=%d resources=%d, want 1/1", resp.ResourceCount, len(resp.Resources))
	}
}

func TestCreateProjectRollsBackOnInvalidResource(t *testing.T) {
	w := httptest.NewRecorder()
	req := newRequest("POST", "/api/features?workspace_id="+testWorkspaceID, map[string]any{
		"title": "Project that should not exist",
		"resources": []map[string]any{
			{
				"resource_type": "github_repo",
				"resource_ref":  map[string]any{"url": "not-a-url"},
			},
		},
	})
	testHandler.CreateFeature(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("CreateProject with invalid resource: expected 400, got %d: %s", w.Code, w.Body.String())
	}

	// Confirm no feature survived (transactional rollback). Listing all projects
	// in the workspace and checking for the title is enough.
	w = httptest.NewRecorder()
	req = newRequest("GET", "/api/features?workspace_id="+testWorkspaceID, nil)
	testHandler.ListFeatures(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListFeatures: %d %s", w.Code, w.Body.String())
	}
	var list struct {
		Features []FeatureResponse `json:"features"`
	}
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	for _, p := range list.Features {
		if p.Title == "Project that should not exist" {
			t.Errorf("invalid resource should have rolled back feature create, but found %s", p.ID)
		}
	}
}

