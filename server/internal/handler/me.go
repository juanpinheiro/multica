package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

// supportedLanguages mirrors `SUPPORTED_LOCALES` in packages/core/i18n/types.ts.
// Keep both lists in sync when adding a locale.
var supportedLanguages = map[string]struct{}{
	"en":      {},
	"zh-Hans": {},
}

// UserResponse is the JSON representation of a user returned by GET /api/me and PATCH /api/me.
type UserResponse struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Email              string  `json:"email"`
	AvatarURL          *string `json:"avatar_url"`
	Language           *string `json:"language"`
	// Pinned IANA tz; nil = no preference (use browser-detected tz).
	Timezone           *string `json:"timezone"`
	ProfileDescription string  `json:"profile_description"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
}

// MaxProfileDescriptionLen caps the user-supplied profile_description body.
const MaxProfileDescriptionLen = 2000

func userToResponse(u db.User) UserResponse {
	return UserResponse{
		ID:                 uuidToString(u.ID),
		Name:               u.Name,
		Email:              u.Email,
		AvatarURL:          textToPtr(u.AvatarUrl),
		Language:           textToPtr(u.Language),
		Timezone:           textToPtr(u.Timezone),
		ProfileDescription: u.ProfileDescription,
		CreatedAt:          timestampToString(u.CreatedAt),
		UpdatedAt:          timestampToString(u.UpdatedAt),
	}
}

// UpdateMeRequest is the JSON body for PATCH /api/me.
type UpdateMeRequest struct {
	Name               *string `json:"name"`
	AvatarURL          *string `json:"avatar_url"`
	Language           *string `json:"language"`
	ProfileDescription *string `json:"profile_description"`
	// IANA tz to pin; "" clears back to NULL; nil leaves untouched.
	Timezone *string `json:"timezone"`
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	user, err := h.Queries.GetUser(r.Context(), parseUUID(userID))
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, userToResponse(user))
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(w, r)
	if !ok {
		return
	}

	var req UpdateMeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	currentUser, err := h.Queries.GetUser(r.Context(), parseUUID(userID))
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	name := currentUser.Name
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
		if name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}
	}

	params := db.UpdateUserParams{
		ID:   currentUser.ID,
		Name: name,
	}
	if req.AvatarURL != nil {
		params.AvatarUrl = pgtype.Text{String: strings.TrimSpace(*req.AvatarURL), Valid: true}
	}
	if req.Language != nil {
		lang := strings.TrimSpace(*req.Language)
		if _, ok := supportedLanguages[lang]; !ok {
			writeError(w, http.StatusBadRequest, "unsupported language")
			return
		}
		params.Language = pgtype.Text{String: lang, Valid: true}
	}
	if req.ProfileDescription != nil {
		// Count runes, not bytes: 2000 chars of Chinese must not be rejected
		// as ~6000 bytes. utf8.RuneCountInString handles invalid UTF-8 by
		// counting each bad byte as one rune, which still bounds the column.
		desc := strings.TrimSpace(*req.ProfileDescription)
		if utf8.RuneCountInString(desc) > MaxProfileDescriptionLen {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("profile_description exceeds %d characters", MaxProfileDescriptionLen))
			return
		}
		params.ProfileDescription = pgtype.Text{String: desc, Valid: true}
	}

	if req.Timezone != nil {
		// Valid=false → column untouched; Valid=true + "" → clear to
		// NULL; Valid=true + IANA → set. Three-way semantics enforced
		// in the UpdateUser SQL CASE.
		tz := strings.TrimSpace(*req.Timezone)
		if tz != "" {
			if loc, err := time.LoadLocation(tz); err != nil || loc == nil {
				writeError(w, http.StatusBadRequest, "invalid timezone")
				return
			}
		}
		params.Timezone = pgtype.Text{String: tz, Valid: true}
	}

	updatedUser, err := h.Queries.UpdateUser(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	writeJSON(w, http.StatusOK, userToResponse(updatedUser))
}
