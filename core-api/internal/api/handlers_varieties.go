package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

// listVarieties handles GET /api/varieties
func (r *Router) listVarieties(w http.ResponseWriter, req *http.Request) {
	rows, err := r.store.Pool.Query(req.Context(), `
		SELECT id, name, piwi, color, status
		FROM varieties
		WHERE status = 'approved'
		ORDER BY name
	`)

	if err != nil {
		slog.Error("varieties: list error", "err", err)
		writeInternalError(w, "failed to list varieties")
		return
	}
	defer rows.Close()

	type Variety struct {
		ID     int64  `json:"id"`
		Name   string `json:"name"`
		Piwi   bool   `json:"piwi"`
		Color  string `json:"color"`
		Status string `json:"status"`
	}

	var varieties []Variety
	for rows.Next() {
		var v Variety
		if err := rows.Scan(&v.ID, &v.Name, &v.Piwi, &v.Color, &v.Status); err != nil {
			continue
		}
		varieties = append(varieties, v)
	}

	writeJSON(w, http.StatusOK, varieties)
}

// submitVarietyRequest is the JSON payload for submitting a new variety.
type submitVarietyRequest struct {
	Name    string `json:"name"`
	Piwi    bool   `json:"piwi"`
	Color   string `json:"color"`
}

// submitVariety handles POST /api/varieties/submit
func (r *Router) submitVariety(w http.ResponseWriter, req *http.Request) {
	var err error
	vineyardID := int64(0)

	// Check if vineyard context is in request URL
	path := req.URL.Path
	if strings.HasPrefix(path, "/api/vineyards/") {
		var eid int64
		eid, _, err = extractID(path)
		if err == nil {
			vineyardID = eid
		}
	}

	if err != nil {
		vineyardID = 0
	}

	var body submitVarietyRequest
	if decodeErr := json.NewDecoder(req.Body).Decode(&body); decodeErr != nil {
		writeValidationError(w, "invalid request body")
		return
	}

	if body.Name == "" {
		writeValidationError(w, "variety name is required",
			ValidationError{Field: "name", Issue: "required"})
		return
	}

	var id int64
	err = r.store.Pool.QueryRow(req.Context(), `
		INSERT INTO varieties (name, piwi, color, status, submitted_by_vineyard_id)
		VALUES ($1, $2, $3, 'review_needed', $4)
		ON CONFLICT (LOWER(name)) DO NOTHING
		RETURNING id
	`, body.Name, body.Piwi, body.Color, vineyardID).Scan(&id)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeConflict(w, "a variety with this name already exists")
		} else {
			slog.Error("varieties: submit error", "err", err)
			writeInternalError(w, "failed to submit variety")
		}
		return
	}

	slog.Info("varieties: submitted", "id", id, "name", body.Name)
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":     id,
		"name":   body.Name,
		"status": "review_needed",
	})
}
