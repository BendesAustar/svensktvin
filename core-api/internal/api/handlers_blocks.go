package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

// listBlocks handles GET /api/vineyards/:id/blocks
func (r *Router) listBlocks(w http.ResponseWriter, req *http.Request) {
	user, _ := getUserFromContext(req.Context())
	vineyardID, _, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	// Verify access
	var role string
	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT role FROM vineyard_members
		WHERE vineyard_id = $1 AND user_id = $2
	`, vineyardID, user.ID).Scan(&role)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeForbidden(w, "access denied")
		} else {
			writeInternalError(w, "verification failed")
		}
		return
	}

	rows, err := r.store.Pool.Query(req.Context(), `
		SELECT b.id, b.block_name, v.name AS variety_name, v.color, v.status,
			   b.area_ha, b.vine_count, b.planting_year,
			   b.is_active
		FROM blocks b
		JOIN varieties v ON v.id = b.variety_id
		WHERE b.vineyard_id = $1 AND b.deleted_at IS NULL
		ORDER BY b.block_name
	`, vineyardID)

	if err != nil {
		slog.Error("blocks: list error", "err", err)
		writeInternalError(w, "failed to list blocks")
		return
	}
	defer rows.Close()

	type BlockSummary struct {
		ID           int64   `json:"id"`
		BlockName    string  `json:"block_name"`
		VarietyName  string  `json:"variety_name"`
		VarietyColor string  `json:"variety_color"`
		Status       string  `json:"variety_status"`
		AreaHA       float64 `json:"area_ha"`
		VineCount    *int    `json:"vine_count,omitempty"`
		PlantingYear *int    `json:"planting_year,omitempty"`
		IsActive     bool    `json:"is_active"`
	}

	var blocks []BlockSummary
	for rows.Next() {
		var b BlockSummary
		if err := rows.Scan(&b.ID, &b.BlockName, &b.VarietyName, &b.VarietyColor,
			&b.Status, &b.AreaHA, &b.VineCount, &b.PlantingYear, &b.IsActive); err != nil {
			continue
		}
		blocks = append(blocks, b)
	}

	writeJSON(w, http.StatusOK, blocks)
}

// createBlockRequest is the JSON payload for creating a block.
type createBlockRequest struct {
	BlockName     string  `json:"block_name"`
	VarietyID     *int64  `json:"variety_id"`
	VarietyName   *string `json:"variety_name"`
	AreaHA        *float64 `json:"area_ha"`
	VineCount     *int    `json:"vine_count"`
	PlantingYear  *int    `json:"planting_year"`
	TrainingSystem *string `json:"training_system"`
	Aspect        *string `json:"aspect"`
	SlopeDegrees  *float64 `json:"slope_degrees"`
	ElevationM    *int    `json:"elevation_m"`
}

// createBlock handles POST /api/vineyards/:id/blocks
func (r *Router) createBlock(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())
	vineyardID, _, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	// Verify editor+ access
	var role string
	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT role FROM vineyard_members
		WHERE vineyard_id = $1 AND user_id = $2
	`, vineyardID, user.ID).Scan(&role)

	if err != nil || role == "" {
		writeForbidden(w, "access denied")
		return
	}

	var body createBlockRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid request body")
		return
	}

	// Validate required fields
	var details []ValidationError
	if body.BlockName == "" {
		details = append(details, ValidationError{Field: "block_name", Issue: "required"})
	}
	if body.AreaHA == nil || *body.AreaHA <= 0 {
		details = append(details, ValidationError{Field: "area_ha", Issue: "required_and_positive"})
	}
	if body.VarietyID == nil && body.VarietyName == nil {
		details = append(details, ValidationError{Field: "variety", Issue: "variety_id or variety_name required"})
	}

	if len(details) > 0 {
		writeValidationError(w, "validation failed", details...)
		return
	}

	var varietyID int64

	// If variety_name provided, create or find variety
	if body.VarietyName != nil && *body.VarietyName != "" {
		// Try to find existing
		err = r.store.Pool.QueryRow(req.Context(), `
			SELECT id FROM varieties WHERE LOWER(name) = LOWER($1)
		`, *body.VarietyName).Scan(&varietyID)

		if err != nil {
			if err == pgx.ErrNoRows {
				// Create new with review_needed status
				err = r.store.Pool.QueryRow(req.Context(), `
					INSERT INTO varieties (name, piwi, color, status, submitted_by_vineyard_id)
					VALUES ($1, false, 'other', 'review_needed', $2)
					RETURNING id
				`, *body.VarietyName, vineyardID).Scan(&varietyID)
			}
		}

		if err != nil {
			slog.Error("blocks: variety lookup/insert error", "err", err)
			writeInternalError(w, "failed to resolve variety")
			return
		}
	} else if body.VarietyID != nil {
		varietyID = *body.VarietyID
	}

	if varietyID == 0 {
		writeValidationError(w, "could not resolve variety")
		return
	}

	// Create block
	var blockID int64
	err = r.store.Pool.QueryRow(req.Context(), `
		INSERT INTO blocks (
			vineyard_id, variety_id, block_name, area_ha,
			vine_count, planting_year, training_system, aspect,
			slope_degrees, elevation_m
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, vineyardID, varietyID, body.BlockName, *body.AreaHA,
		body.VineCount, body.PlantingYear, body.TrainingSystem, body.Aspect,
		body.SlopeDegrees, body.ElevationM).Scan(&blockID)

	if err != nil {
		if err.Error() == `pq: duplicate key value violates unique constraint` {
			writeConflict(w, "a block with this name already exists in this vineyard")
		} else {
			slog.Error("blocks: create error", "err", err)
			writeInternalError(w, "failed to create block")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":            blockID,
		"block_name":    body.BlockName,
		"variety_id":    varietyID,
		"area_ha":       *body.AreaHA,
		"is_active":     true,
	})
}

// updateBlock handles PUT /api/vineyards/:id/blocks/:blockId
func (r *Router) updateBlock(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())
	vineyardID, rest, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	// Parse block ID from remaining path
	blockID := 0
	n, _ := fmt.Sscanf(rest, "%d", &blockID)
	if n == 0 {
		writeNotFound(w, "block not found")
		return
	}

	// Verify access
	var role string
	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT role FROM vineyard_members
		WHERE vineyard_id = $1 AND user_id = $2
	`, vineyardID, user.ID).Scan(&role)

	if err != nil || role == "" {
		writeForbidden(w, "access denied")
		return
	}

	var body createBlockRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid request body")
		return
	}

	if body.BlockName == "" || body.AreaHA == nil || *body.AreaHA <= 0 {
		writeValidationError(w, "block_name and area_ha are required and must be positive")
		return
	}

	_, err = r.store.Pool.Exec(req.Context(), `
		UPDATE blocks SET
			block_name = $1,
			variety_id = $2,
			area_ha = $3,
			vine_count = $4,
			planting_year = $5,
			training_system = $6,
			aspect = $7,
			slope_degrees = $8,
			elevation_m = $9,
			updated_at = now()
		WHERE id = $10 AND vineyard_id = $11 AND deleted_at IS NULL
	`, body.BlockName, body.VarietyID, *body.AreaHA,
		body.VineCount, body.PlantingYear, body.TrainingSystem, body.Aspect,
		body.SlopeDegrees, body.ElevationM,
		int64(blockID), vineyardID)

	if err != nil {
		slog.Error("blocks: update error", "err", err)
		writeInternalError(w, "failed to update block")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"id": int64(blockID)})
}

// deleteBlock handles DELETE /api/vineyards/:id/blocks/:blockId
func (r *Router) deleteBlock(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())
	vineyardID, rest, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	blockID := 0
	n, _ := fmt.Sscanf(rest, "%d", &blockID)
	if n == 0 {
		writeNotFound(w, "block not found")
		return
	}

	var role string
	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT role FROM vineyard_members
		WHERE vineyard_id = $1 AND user_id = $2
	`, vineyardID, user.ID).Scan(&role)

	if err != nil || role == "" {
		writeForbidden(w, "access denied")
		return
	}

	_, err = r.store.Pool.Exec(req.Context(), `
		UPDATE blocks SET deleted_at = now(), is_active = false
		WHERE id = $1 AND vineyard_id = $2 AND deleted_at IS NULL
	`, int64(blockID), vineyardID)

	if err != nil {
		writeInternalError(w, "failed to delete block")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"id": int64(blockID)})
}

// getBlock handles GET /api/vineyards/:id/blocks/:blockId
func (r *Router) getBlock(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())
	vineyardID, _, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	blockID := 0
	n, _ := fmt.Sscanf(strings.TrimPrefix(req.URL.Path, fmt.Sprintf("/api/vineyards/%d/blocks/", vineyardID)), "%d", &blockID)
	if n == 0 {
		writeNotFound(w, "block not found")
		return
	}

	var role string
	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT role FROM vineyard_members
		WHERE vineyard_id = $1 AND user_id = $2
	`, vineyardID, user.ID).Scan(&role)

	if err != nil || role == "" {
		writeForbidden(w, "access denied")
		return
	}

	var block struct {
		ID             int64    `json:"id"`
		BlockName      string   `json:"block_name"`
		VarietyID      int64    `json:"variety_id"`
		VarietyName    string   `json:"variety_name"`
		VarietyColor   string   `json:"variety_color"`
		AreaHA         float64  `json:"area_ha"`
		VineCount      *int     `json:"vine_count,omitempty"`
		PlantingYear   *int     `json:"planting_year,omitempty"`
		TrainingSystem *string  `json:"training_system"`
		Aspect         *string  `json:"aspect"`
		SlopeDegrees   *float64 `json:"slope_degrees"`
		ElevationM     *int     `json:"elevation_m"`
		IsActive       bool     `json:"is_active"`
	}

	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT b.id, b.block_name, b.variety_id, v.name, v.color,
			   b.area_ha, b.vine_count, b.planting_year,
			   b.training_system, b.aspect, b.slope_degrees, b.elevation_m,
			   b.is_active
		FROM blocks b
		JOIN varieties v ON v.id = b.variety_id
		WHERE b.id = $1 AND b.vineyard_id = $2 AND b.deleted_at IS NULL
	`, int64(blockID), vineyardID).Scan(
		&block.ID, &block.BlockName, &block.VarietyID, &block.VarietyName, &block.VarietyColor,
		&block.AreaHA, &block.VineCount, &block.PlantingYear,
		&block.TrainingSystem, &block.Aspect, &block.SlopeDegrees, &block.ElevationM, &block.IsActive,
	)

	if err != nil {
		writeNotFound(w, "block not found")
		return
	}

	writeJSON(w, http.StatusOK, block)
}
