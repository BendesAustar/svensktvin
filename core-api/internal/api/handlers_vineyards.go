package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// listVineyards handles GET /api/vineyards
func (r *Router) listVineyards(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())

	rows, err := r.store.Pool.Query(req.Context(), `
		SELECT v.id, v.name, v.county, v.municipality,
			   v.organic, v.biodynamic, v.total_area_ha,
			   vm.role
		FROM vineyards v
		JOIN vineyard_members vm ON vm.vineyard_id = v.id
		WHERE vm.user_id = $1 AND v.deleted_at IS NULL
		ORDER BY v.name
	`, user.ID)

	if err != nil {
		slog.Error("vineyards: list error", "err", err)
		writeInternalError(w, "failed to list vineyards")
		return
	}
	defer rows.Close()

	type VineyardSummary struct {
		ID            int64   `json:"id"`
		Name          string  `json:"name"`
		County        string  `json:"county"`
		Municipality  string  `json:"municipality"`
		Organic       bool    `json:"organic"`
		Biodynamic    bool    `json:"biodynamic"`
		TotalAreaHA   float64 `json:"total_area_ha,omitempty"`
		Role          string  `json:"role"`
	}

	var vineyards []VineyardSummary
	for rows.Next() {
		var v VineyardSummary
		if err := rows.Scan(&v.ID, &v.Name, &v.County, &v.Municipality,
			&v.Organic, &v.Biodynamic, &v.TotalAreaHA, &v.Role); err != nil {
			slog.Error("vineyards: scan error", "err", err)
			continue
		}
		vineyards = append(vineyards, v)
	}

	writeJSON(w, http.StatusOK, vineyards)
}

// createVineyardRequest is the JSON payload for creating a vineyard.
type createVineyardRequest struct {
	Name          string   `json:"name"`
	County        string   `json:"county"`
	Municipality  string   `json:"municipality"`
	Lat           float64  `json:"lat"`
	Lon           float64  `json:"lon"`
	EstablishedYear *int   `json:"established_year"`
	TotalAreaHA   *float64 `json:"total_area_ha"`
	Organic       bool     `json:"organic"`
	Biodynamic    bool     `json:"biodynamic"`
	LegalID       *string  `json:"legal_id"`
	LegalIDType   *string  `json:"legal_id_type"`
	LegalName     *string  `json:"legal_name"`
}

// createVineyard handles POST /api/vineyards
func (r *Router) createVineyard(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())

	var body createVineyardRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid request body")
		return
	}

	// Validate required fields
	var details []ValidationError
	if body.Name == "" {
		details = append(details, ValidationError{Field: "name", Issue: "required"})
	}
	if body.County == "" {
		details = append(details, ValidationError{Field: "county", Issue: "required"})
	}
	if body.Municipality == "" {
		details = append(details, ValidationError{Field: "municipality", Issue: "required"})
	}
	if body.Lat == 0 && body.Lon == 0 {
		details = append(details, ValidationError{Field: "location", Issue: "lat/lon required"})
	}

	if len(details) > 0 {
		writeValidationError(w, "validation failed", details...)
		return
	}

	// Validate name length
	if len(body.Name) > 100 {
		writeValidationError(w, "name too long (max 100 chars)",
			ValidationError{Field: "name", Issue: "max_length"})
		return
	}

	// Validate established year range
	if body.EstablishedYear != nil && (*body.EstablishedYear < 1800 || *body.EstablishedYear > 2030) {
		writeValidationError(w, "established_year must be between 1800 and 2030",
			ValidationError{Field: "established_year", Issue: "range"})
		return
	}

	var establishedYear *int
	if body.EstablishedYear != nil {
		establishedYear = body.EstablishedYear
	}
	var totalAreaHA *float64
	if body.TotalAreaHA != nil && *body.TotalAreaHA > 0 {
		totalAreaHA = body.TotalAreaHA
	}

	// Insert vineyard
	var vineyardID int64
	err := r.store.Pool.QueryRow(req.Context(), `
		INSERT INTO vineyards (
			name, county, municipality, lat, lon,
			established_year, total_area_ha,
			organic, biodynamic,
			legal_id, legal_id_type, legal_name
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`, body.Name, body.County, body.Municipality, body.Lat, body.Lon,
		establishedYear, totalAreaHA,
		body.Organic, body.Biodynamic,
		body.LegalID, body.LegalIDType, body.LegalName).Scan(&vineyardID)

	if err != nil {
		slog.Error("vineyards: create error", "err", err)
		writeInternalError(w, "failed to create vineyard")
		return
	}

	// Auto-assign creator as owner
	_, err = r.store.Pool.Exec(req.Context(), `
		INSERT INTO vineyard_members (vineyard_id, user_id, role)
		VALUES ($1, $2, 'owner')
	`, vineyardID, user.ID)

	if err != nil {
		slog.Error("vineyards: member insert error", "err", err)
		writeInternalError(w, "failed to create vineyard")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":          vineyardID,
		"name":        body.Name,
		"county":      body.County,
		"municipality": body.Municipality,
		"organic":     body.Organic,
		"biodynamic":  body.Biodynamic,
		"role":        "owner",
	})
}

// updateVineyard handles PUT /api/vineyards/:id
func (r *Router) updateVineyard(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())

	vineyardID, _, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	// Verify ownership
	var role string
	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT role FROM vineyard_members
		WHERE vineyard_id = $1 AND user_id = $2 AND role = 'owner'
	`, vineyardID, user.ID).Scan(&role)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeForbidden(w, "only owners can update vineyard details")
		} else {
			writeInternalError(w, "verification failed")
		}
		return
	}

	var body createVineyardRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid request body")
		return
	}

	if body.Name == "" || body.County == "" || body.Municipality == "" {
		writeValidationError(w, "name, county, and municipality are required")
		return
	}

	_, err = r.store.Pool.Exec(req.Context(), `
		UPDATE vineyards SET
			name = COALESCE($1, name),
			county = COALESCE($2, county),
			municipality = COALESCE($3, municipality),
			organic = COALESCE($4, organic),
			biodynamic = COALESCE($5, biodynamic),
			legal_id = COALESCE($6, legal_id),
			legal_id_type = COALESCE($7, legal_id_type),
			legal_name = COALESCE($8, legal_name)
		WHERE id = $9 AND deleted_at IS NULL
	`, body.Name, body.County, body.Municipality,
		body.Organic, body.Biodynamic,
		body.LegalID, body.LegalIDType, body.LegalName,
		vineyardID)

	if err != nil {
		slog.Error("vineyards: update error", "err", err)
		writeInternalError(w, "failed to update vineyard")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"id": vineyardID})
}

// deleteVineyard handles DELETE /api/vineyards/:id
func (r *Router) deleteVineyard(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())

	vineyardID, _, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	// Verify ownership
	var role string
	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT role FROM vineyard_members
		WHERE vineyard_id = $1 AND user_id = $2 AND role = 'owner'
	`, vineyardID, user.ID).Scan(&role)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeForbidden(w, "only owners can delete vineyard")
		} else {
			writeInternalError(w, "verification failed")
		}
		return
	}

	// Soft delete
	_, err = r.store.Pool.Exec(req.Context(), `
		UPDATE vineyards SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL
	`, vineyardID)

	if err != nil {
		writeInternalError(w, "failed to delete vineyard")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"id": vineyardID})
}

// getVineyard handles GET /api/vineyards/:id
func (r *Router) getVineyard(w http.ResponseWriter, req *http.Request) {
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

	var vineyard struct {
		ID             int64    `json:"id"`
		Name           string   `json:"name"`
		County         string   `json:"county"`
		Municipality   string   `json:"municipality"`
		Lat            float64  `json:"lat"`
		Lon            float64  `json:"lon"`
		EstablishedYear *int    `json:"established_year"`
		TotalAreaHA    *float64 `json:"total_area_ha"`
		Organic        bool     `json:"organic"`
		Biodynamic     bool     `json:"biodynamic"`
		LegalID        *string  `json:"legal_id"`
		LegalIDType    *string  `json:"legal_id_type"`
		LegalName      *string  `json:"legal_name"`
		Role           string   `json:"role"`
	}

	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT id, name, county, municipality, lat, lon,
			   established_year, total_area_ha, organic, biodynamic,
			   legal_id, legal_id_type, legal_name
		FROM vineyards
		WHERE id = $1 AND deleted_at IS NULL
	`, vineyardID).Scan(
		&vineyard.ID, &vineyard.Name, &vineyard.County, &vineyard.Municipality,
		&vineyard.Lat, &vineyard.Lon,
		&vineyard.EstablishedYear, &vineyard.TotalAreaHA,
		&vineyard.Organic, &vineyard.Biodynamic,
		&vineyard.LegalID, &vineyard.LegalIDType, &vineyard.LegalName,
	)

	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}
	vineyard.Role = role

	writeJSON(w, http.StatusOK, vineyard)
}
