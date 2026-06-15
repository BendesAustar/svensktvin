// Package api provides HTTP handlers for JSON API endpoints (varieties search, geo reverse, etc.).
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/svensktvin/svensktvin/internal/db"
)

// VarietySearchResult represents the JSON response for variety search.
type VarietySearchResult struct {
	Matches         []VarietyMatch `json:"matches"`
	HighConfidence  bool           `json:"high_confidence"`
}

// VarietyMatch represents a single variety search match.
type VarietyMatch struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Piwi  bool   `json:"piwi"`
}

// VarietySearchHandler handles variety search API requests.
type VarietySearchHandler struct {
	store *db.Store
}

// NewVarietySearchHandler creates a new variety search handler.
func NewVarietySearchHandler(store *db.Store) *VarietySearchHandler {
	return &VarietySearchHandler{
		store: store,
	}
}

// HandleGET handles GET /api/varieties/search?q=xxx
func (h *VarietySearchHandler) HandleGET(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get query parameter
	query := r.URL.Query().Get("q")
	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(VarietySearchResult{
			Matches:        []VarietyMatch{},
			HighConfidence: false,
		})
		return
	}

	// Search varieties
	matches, err := h.store.SearchVarieties(r.Context(), query)
	if err != nil {
		slog.Error("varieties: search", "err", err, "query", query)
		// Return empty results on error (don't leak errors to client)
		json.NewEncoder(w).Encode(VarietySearchResult{
			Matches:        []VarietyMatch{},
			HighConfidence: false,
		})
		return
	}

	// Convert to response format
	var resultVarieties []VarietyMatch
	var highConfidence bool
	for _, m := range matches {
		resultVarieties = append(resultVarieties, VarietyMatch{
			ID:    m.ID,
			Name:  m.Name,
			Color: m.Color,
			Piwi:  m.Piwi,
		})
		if m.Score >= 0.8 {
			highConfidence = true
		}
	}

	json.NewEncoder(w).Encode(VarietySearchResult{
		Matches:        resultVarieties,
		HighConfidence: highConfidence,
	})
}
