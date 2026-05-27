package handler

import (
	"net/http"
	"os"
)

type AppConfig struct {
	CdnDomain string `json:"cdn_domain"`
	// Public auth config consumed by the web app at runtime so self-hosted
	// deployments do not need to rebuild the frontend image when operators
	// toggle signup or wire Google OAuth.
	AllowSignup    bool   `json:"allow_signup"`
	GoogleClientID string `json:"google_client_id,omitempty"`
}

// GetConfig is mounted on the public (unauthenticated) route group because
// the web app calls it before login to decide whether to render the Google
// sign-in button and signup UI. Only add fields here that are safe to expose
// to anonymous callers — never user- or tenant-scoped data.
func (h *Handler) GetConfig(w http.ResponseWriter, r *http.Request) {
	config := AppConfig{
		AllowSignup:    os.Getenv("ALLOW_SIGNUP") != "false",
		GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"),
	}
	if h.Storage != nil {
		config.CdnDomain = h.Storage.CdnDomain()
	}

	writeJSON(w, http.StatusOK, config)
}
