package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

// createHarvestRequest is the JSON payload for creating a harvest record.
type createHarvestRequest struct {
	BlockID            int64   `json:"block_id"`
	HarvestYear        *int    `json:"harvest_year"`
	HarvestDate        *string `json:"harvest_date"`
	YieldKG            *float64 `json:"yield_kg"`
	Brix               *float64 `json:"brix"`
	AcidGL             *float64 `json:"acid_g_l"`
	VineHealthRating   *int    `json:"vine_health_rating"`
	Notes              *string `json:"notes"`
	StillWineL         *float64 `json:"still_wine_l"`
	SparklingL         *float64 `json:"sparkling_l"`
	JuiceL             *float64 `json:"juice_l"`
	SoldKG             *float64 `json:"sold_kg"`
	DiscardedKG        *float64 `json:"discarded_kg"`
}

// createHarvest handles POST /api/vineyards/:id/harvests
func (r *Router) createHarvest(w http.ResponseWriter, req *http.Request) {
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

	if err != nil || role == "" {
		writeForbidden(w, "access denied")
		return
	}

	var body createHarvestRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid request body")
		return
	}

	var details []ValidationError
	if body.BlockID == 0 {
		details = append(details, ValidationError{Field: "block_id", Issue: "required"})
	}
	if body.HarvestYear == nil || *body.HarvestYear <= 0 {
		details = append(details, ValidationError{Field: "harvest_year", Issue: "required"})
	}
	if body.YieldKG == nil || *body.YieldKG <= 0 {
		details = append(details, ValidationError{Field: "yield_kg", Issue: "required_and_positive"})
	}
	currentYear := 2026
	if body.HarvestYear != nil && *body.HarvestYear > currentYear {
		details = append(details, ValidationError{Field: "harvest_year", Issue: "future_date"})
	}
	if body.VineHealthRating != nil && (*body.VineHealthRating < 1 || *body.VineHealthRating > 5) {
		details = append(details, ValidationError{Field: "vine_health_rating", Issue: "range_1_to_5"})
	}

	if len(details) > 0 {
		writeValidationError(w, "validation failed", details...)
		return
	}

	// Verify block belongs to vineyard
	var blockName string
	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT block_name FROM blocks
		WHERE id = $1 AND vineyard_id = $2 AND deleted_at IS NULL
	`, body.BlockID, vineyardID).Scan(&blockName)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeNotFound(w, "block not found in this vineyard")
		} else {
			writeInternalError(w, "block verification failed")
		}
		return
	}

	var existingID *int64
	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT id FROM harvest_records
		WHERE block_id = $1 AND harvest_year = $2 AND deleted_at IS NULL
	`, body.BlockID, *body.HarvestYear).Scan(&existingID)

	if err == nil && existingID != nil {
		writeConflict(w, "a harvest record for this year already exists")
		return
	}

	var harvestID int64
	err = r.store.Pool.QueryRow(req.Context(), `
		INSERT INTO harvest_records (
			block_id, harvest_year, harvest_date, yield_kg,
			brix, acid_g_l, vine_health_rating, notes,
			still_wine_l, sparkling_l, juice_l,
			sold_kg, discarded_kg
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`, body.BlockID, *body.HarvestYear, body.HarvestDate, *body.YieldKG,
		body.Brix, body.AcidGL, body.VineHealthRating, body.Notes,
		body.StillWineL, body.SparklingL, body.JuiceL,
		body.SoldKG, body.DiscardedKG).Scan(&harvestID)

	if err != nil {
		slog.Error("harvests: create error", "err", err)
		writeInternalError(w, "failed to create harvest record")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":                  harvestID,
		"block_id":            body.BlockID,
		"harvest_year":        *body.HarvestYear,
		"yield_kg":            *body.YieldKG,
		"vine_health_rating":  body.VineHealthRating,
	})
}

// updateHarvest handles PUT /api/vineyards/:id/harvests/:recordId
func (r *Router) updateHarvest(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())
	vineyardID, rest, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	parts := strings.Split(strings.Trim(rest, "/"), "/")
	recordID := 0
	n, _ := fmt.Sscanf(parts[0], "%d", &recordID)
	if n == 0 {
		writeNotFound(w, "harvest record not found")
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

	var body createHarvestRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid request body")
		return
	}

	if body.BlockID == 0 || body.HarvestYear == nil || *body.HarvestYear <= 0 ||
		body.YieldKG == nil || *body.YieldKG <= 0 {
		writeValidationError(w, "block_id, harvest_year, and yield_kg are required")
		return
	}

	_, err = r.store.Pool.Exec(req.Context(), `
		UPDATE harvest_records SET
			block_id = $1, harvest_year = $2, harvest_date = $3, yield_kg = $4,
			brix = $5, acid_g_l = $6, vine_health_rating = $7, notes = $8,
			still_wine_l = $9, sparkling_l = $10, juice_l = $11,
			sold_kg = $12, discarded_kg = $13, updated_at = now()
		WHERE id = $14 AND deleted_at IS NULL
	`, body.BlockID, *body.HarvestYear, body.HarvestDate, *body.YieldKG,
		body.Brix, body.AcidGL, body.VineHealthRating, body.Notes,
		body.StillWineL, body.SparklingL, body.JuiceL,
		body.SoldKG, body.DiscardedKG, int64(recordID))

	if err != nil {
		slog.Error("harvests: update error", "err", err)
		writeInternalError(w, "failed to update harvest record")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"id": int64(recordID)})
}

// deleteHarvest handles DELETE /api/vineyards/:id/harvests/:recordId
func (r *Router) deleteHarvest(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())
	vineyardID, rest, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	parts := strings.Split(strings.Trim(rest, "/"), "/")
	recordID := 0
	n, _ := fmt.Sscanf(parts[0], "%d", &recordID)
	if n == 0 {
		writeNotFound(w, "harvest record not found")
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
		UPDATE harvest_records SET deleted_at = now()
		WHERE id = $1 AND vineyard_id = $2 AND deleted_at IS NULL
	`, int64(recordID), vineyardID)

	if err != nil {
		writeInternalError(w, "failed to delete harvest record")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"id": int64(recordID)})
}

// getBenchmarks handles GET /api/vineyards/:id/benchmarks
func (r *Router) getBenchmarks(w http.ResponseWriter, req *http.Request) {
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

	if err != nil || role == "" {
		writeForbidden(w, "access denied")
		return
	}

	var vineyardName, county string
	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT name, county FROM vineyards WHERE id = $1 AND deleted_at IS NULL
	`, vineyardID).Scan(&vineyardName, &county)

	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	// User yields
	rows, err := r.store.Pool.Query(req.Context(), `
		SELECT
			v.name, hr.harvest_year,
			round(avg(hr.yield_kg / b.area_ha), 0)::int,
			round(avg(hr.brix::numeric), 1),
			count(*)
		FROM harvest_records hr
		JOIN blocks b ON b.id = hr.block_id
		JOIN varieties v ON v.id = b.variety_id
		WHERE b.vineyard_id = $1
			AND hr.harvest_year IS NOT NULL
		GROUP BY v.name, hr.harvest_year
		ORDER BY hr.harvest_year DESC, v.name
	`, vineyardID)

	if err != nil {
		slog.Error("benchmarks: user yields error", "err", err)
		writeInternalError(w, "failed to load benchmarks")
		return
	}
	defer rows.Close()

	type UserYield struct {
		VarietyName  string  `json:"variety_name"`
		HarvestYear  int     `json:"harvest_year"`
		AvgYieldKgHa int     `json:"avg_yield_kg_ha"`
		AvgBrix      float64 `json:"avg_brix"`
		HarvestCount int     `json:"harvest_count"`
	}

	var userYields []UserYield
	for rows.Next() {
		var y UserYield
		if err := rows.Scan(&y.VarietyName, &y.HarvestYear, &y.AvgYieldKgHa,
			&y.AvgBrix, &y.HarvestCount); err != nil {
			continue
		}
		userYields = append(userYields, y)
	}

	// Regional benchmarks
	rows, err = r.store.Pool.Query(req.Context(), `
		SELECT
			var.name, hr.harvest_year,
			round(avg(hr.yield_kg / b.area_ha), 0)::int,
			round(avg(hr.brix::numeric), 1),
			count(DISTINCT vi.id)
		FROM harvest_records hr
		JOIN blocks b ON b.id = hr.block_id
		JOIN varieties var ON var.id = b.variety_id
		JOIN vineyards vi ON vi.id = b.vineyard_id
		WHERE vi.county = $1
			AND var.status = 'approved'
			AND hr.harvest_year IS NOT NULL
		GROUP BY var.name, hr.harvest_year
		HAVING count(DISTINCT vi.id) >= 3
		ORDER BY hr.harvest_year DESC, var.name
	`, county)

	if err != nil {
		slog.Error("benchmarks: regional error", "err", err)
		writeInternalError(w, "failed to load benchmarks")
		return
	}
	defer rows.Close()

	type RegionalBenchmark struct {
		VarietyName     string  `json:"variety_name"`
		HarvestYear     int     `json:"harvest_year"`
		CountyAvgKgHa   int     `json:"county_avg_kg_ha"`
		CountyAvgBrix   float64 `json:"county_avg_brix"`
		VineyardCount   int     `json:"vineyard_count"`
	}

	var regionalBenchmarks []RegionalBenchmark
	for rows.Next() {
		var b RegionalBenchmark
		if err := rows.Scan(&b.VarietyName, &b.HarvestYear, &b.CountyAvgKgHa,
			&b.CountyAvgBrix, &b.VineyardCount); err != nil {
			continue
		}
		regionalBenchmarks = append(regionalBenchmarks, b)
	}

	// Timeline
	rows, err = r.store.Pool.Query(req.Context(), `
		SELECT hr.harvest_year, hr.harvest_date, b.block_name,
			   v.name, hr.yield_kg, hr.brix, hr.vine_health_rating
		FROM harvest_records hr
		JOIN blocks b ON b.id = hr.block_id
		JOIN varieties v ON v.id = b.variety_id
		WHERE b.vineyard_id = $1
			AND hr.harvest_year IS NOT NULL
		ORDER BY hr.harvest_year DESC, hr.harvest_date DESC
	`, vineyardID)

	if err != nil {
		slog.Error("benchmarks: timeline error", "err", err)
		writeInternalError(w, "failed to load benchmarks")
		return
	}
	defer rows.Close()

	type TimelineEntry struct {
		HarvestYear      int     `json:"harvest_year"`
		HarvestDate      *string `json:"harvest_date"`
		BlockName        string  `json:"block_name"`
		VarietyName      string  `json:"variety_name"`
		YieldKG          float64 `json:"yield_kg"`
		Brix             *float64 `json:"brix"`
		VineHealthRating *int    `json:"vine_health_rating"`
	}

	var timeline []TimelineEntry
	for rows.Next() {
		var t TimelineEntry
		if err := rows.Scan(&t.HarvestYear, &t.HarvestDate, &t.BlockName,
			&t.VarietyName, &t.YieldKG, &t.Brix, &t.VineHealthRating); err != nil {
			continue
		}
		timeline = append(timeline, t)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"vineyard": map[string]any{
			"id":     vineyardID,
			"name":   vineyardName,
			"county": county,
		},
		"user_yields":         userYields,
		"regional_benchmarks": regionalBenchmarks,
		"timeline":            timeline,
	})
}

// searchVarieties handles POST /api/blocks/search-varieties
func (r *Router) searchVarieties(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Query string `json:"query"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid request body")
		return
	}

	if body.Query == "" || len(body.Query) < 2 {
		writeJSON(w, http.StatusOK, map[string]any{
			"query":           body.Query,
			"matches":         []any{},
			"high_confidence": false,
		})
		return
	}

	rows, err := r.store.Pool.Query(req.Context(), `
		SELECT id, name, piwi, color,
			   round(similarity(name, $1)::numeric, 2) AS score
		FROM varieties
		WHERE similarity(name, $1) > 0.4
			AND status = 'approved'
		ORDER BY similarity(name, $1) DESC
		LIMIT 3
	`, body.Query)

	if err != nil {
		slog.Error("varieties: search error", "err", err)
		writeInternalError(w, "search failed")
		return
	}
	defer rows.Close()

	type VarietyMatch struct {
		ID    int64   `json:"id"`
		Name  string  `json:"name"`
		Score float64 `json:"score"`
		Piwi  bool    `json:"piwi"`
		Color string  `json:"color"`
	}

	var matches []VarietyMatch
	for rows.Next() {
		var m VarietyMatch
		if err := rows.Scan(&m.ID, &m.Name, &m.Piwi, &m.Color, &m.Score); err != nil {
			continue
		}
		matches = append(matches, m)
	}

	highConfidence := false
	if len(matches) > 0 && matches[0].Score >= 0.8 {
		highConfidence = true
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"query":           body.Query,
		"matches":         matches,
		"high_confidence": highConfidence,
	})
}

// listMembers handles GET /api/vineyards/:id/members
func (r *Router) listMembers(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())
	vineyardID, _, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT 1 FROM vineyard_members
		WHERE vineyard_id = $1 AND user_id = $2
	`, vineyardID, user.ID).Scan(nil)

	if err != nil {
		writeForbidden(w, "access denied")
		return
	}

	rows, err := r.store.Pool.Query(req.Context(), `
		SELECT um.id, um.role, u.email, u.name
		FROM vineyard_members um
		JOIN users u ON u.id = um.user_id
		WHERE um.vineyard_id = $1
		ORDER BY um.role DESC, u.name
	`, vineyardID)

	if err != nil {
		slog.Error("members: list error", "err", err)
		writeInternalError(w, "failed to list members")
		return
	}
	defer rows.Close()

	type Member struct {
		ID    int64  `json:"id"`
		Role  string `json:"role"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.Role, &m.Email, &m.Name); err != nil {
			continue
		}
		members = append(members, m)
	}

	writeJSON(w, http.StatusOK, members)
}

// addMemberRequest is the JSON payload for adding a member.
type addMemberRequest struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

// addMember handles POST /api/vineyards/:id/members
func (r *Router) addMember(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())
	vineyardID, _, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT 1 FROM vineyard_members
		WHERE vineyard_id = $1 AND user_id = $2 AND role = 'owner'
	`, vineyardID, user.ID).Scan(nil)

	if err != nil {
		writeForbidden(w, "only owners can manage members")
		return
	}

	var body addMemberRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid request body")
		return
	}

	if body.UserID == 0 {
		writeValidationError(w, "user_id is required", ValidationError{Field: "user_id", Issue: "required"})
		return
	}
	if body.Role != "owner" && body.Role != "editor" {
		writeValidationError(w, "role must be 'owner' or 'editor'",
			ValidationError{Field: "role", Issue: "invalid_role"})
		return
	}

	_, err = r.store.Pool.Exec(req.Context(), `
		INSERT INTO vineyard_members (vineyard_id, user_id, role)
		VALUES ($1, $2, $3)
	`, vineyardID, body.UserID, body.Role)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			writeConflict(w, "user is already a member of this vineyard")
		} else {
			slog.Error("members: add error", "err", err)
			writeInternalError(w, "failed to add member")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"vineyard_id": vineyardID,
		"user_id":     body.UserID,
		"role":        body.Role,
	})
}

// updateMember handles PUT /api/vineyards/:id/members/:userId
func (r *Router) updateMember(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())
	vineyardID, rest, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	userId := 0
	n, _ := fmt.Sscanf(rest, "%d", &userId)
	if n == 0 {
		writeNotFound(w, "member not found")
		return
	}

	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT 1 FROM vineyard_members
		WHERE vineyard_id = $1 AND user_id = $2 AND role = 'owner'
	`, vineyardID, user.ID).Scan(nil)

	if err != nil {
		writeForbidden(w, "only owners can manage members")
		return
	}

	var body addMemberRequest
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid request body")
		return
	}

	if body.Role != "owner" && body.Role != "editor" {
		writeValidationError(w, "role must be 'owner' or 'editor'")
		return
	}

	_, err = r.store.Pool.Exec(req.Context(), `
		UPDATE vineyard_members SET role = $1
		WHERE vineyard_id = $2 AND user_id = $3
	`, body.Role, vineyardID, int64(userId))

	if err != nil {
		writeInternalError(w, "failed to update member role")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"vineyard_id": vineyardID,
		"user_id":     int64(userId),
		"role":        body.Role,
	})
}

// removeMember handles DELETE /api/vineyards/:id/members/:userId
func (r *Router) removeMember(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())
	vineyardID, rest, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	userId := 0
	n, _ := fmt.Sscanf(rest, "%d", &userId)
	if n == 0 {
		writeNotFound(w, "member not found")
		return
	}

	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT 1 FROM vineyard_members
		WHERE vineyard_id = $1 AND user_id = $2 AND role = 'owner'
	`, vineyardID, user.ID).Scan(nil)

	if err != nil {
		writeForbidden(w, "only owners can remove members")
		return
	}

	if int64(userId) == user.ID {
		writeValidationError(w, "cannot remove yourself from vineyard")
		return
	}

	_, err = r.store.Pool.Exec(req.Context(), `
		DELETE FROM vineyard_members
		WHERE vineyard_id = $1 AND user_id = $2
	`, vineyardID, int64(userId))

	if err != nil {
		writeInternalError(w, "failed to remove member")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"vineyard_id": vineyardID,
		"user_id":     int64(userId),
		"removed":     true,
	})
}

// listHarvests handles GET /api/vineyards/:id/harvests
func (r *Router) listHarvests(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())
	vineyardID, _, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
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

	rows, err := r.store.Pool.Query(req.Context(), `
		SELECT hr.id, hr.block_id, hr.harvest_year, hr.harvest_date,
			   hr.yield_kg, hr.brix, hr.vine_health_rating, b.block_name,
			   v.name AS variety_name
		FROM harvest_records hr
		JOIN blocks b ON b.id = hr.block_id
		JOIN varieties v ON v.id = b.variety_id
		WHERE b.vineyard_id = $1 AND hr.deleted_at IS NULL
		ORDER BY hr.harvest_year DESC, b.block_name
	`, vineyardID)

	if err != nil {
		slog.Error("harvests: list error", "err", err)
		writeInternalError(w, "failed to list harvests")
		return
	}
	defer rows.Close()

	type HarvestSummary struct {
		ID               int64    `json:"id"`
		BlockID          int64    `json:"block_id"`
		HarvestYear      *int     `json:"harvest_year"`
		HarvestDate      *string  `json:"harvest_date"`
		YieldKG          float64  `json:"yield_kg"`
		Brix             *float64 `json:"brix"`
		VineHealthRating *int     `json:"vine_health_rating"`
		BlockName        string   `json:"block_name"`
		VarietyName      string   `json:"variety_name"`
	}

	var harvests []HarvestSummary
	for rows.Next() {
		var h HarvestSummary
		if err := rows.Scan(&h.ID, &h.BlockID, &h.HarvestYear, &h.HarvestDate,
			&h.YieldKG, &h.Brix, &h.VineHealthRating, &h.BlockName, &h.VarietyName); err != nil {
			continue
		}
		harvests = append(harvests, h)
	}

	writeJSON(w, http.StatusOK, harvests)
}

// getHarvest handles GET /api/vineyards/:id/harvests/:recordId
func (r *Router) getHarvest(w http.ResponseWriter, req *http.Request) {
	 user, _ := getUserFromContext(req.Context())
	vineyardID, _, err := extractID(req.URL.Path)
	if err != nil {
		writeNotFound(w, "vineyard not found")
		return
	}

	recordID := 0
	rest := strings.TrimPrefix(req.URL.Path, fmt.Sprintf("/api/vineyards/%d/harvests/", vineyardID))
	n, _ := fmt.Sscanf(rest, "%d", &recordID)
	if n == 0 {
		writeNotFound(w, "harvest record not found")
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

	var harvest struct {
		ID               int64    `json:"id"`
		BlockID          int64    `json:"block_id"`
		HarvestYear      *int     `json:"harvest_year"`
		HarvestDate      *string  `json:"harvest_date"`
		YieldKG          float64  `json:"yield_kg"`
		Brix             *float64 `json:"brix"`
		AcidGL           *float64 `json:"acid_g_l"`
		VineHealthRating *int     `json:"vine_health_rating"`
		Notes            *string  `json:"notes"`
		StillWineL       *float64 `json:"still_wine_l"`
		SparklingL       *float64 `json:"sparkling_l"`
		JuiceL           *float64 `json:"juice_l"`
		SoldKG           *float64 `json:"sold_kg"`
		DiscardedKG      *float64 `json:"discarded_kg"`
	}

	err = r.store.Pool.QueryRow(req.Context(), `
		SELECT id, block_id, harvest_year, harvest_date, yield_kg,
			   brix, acid_g_l, vine_health_rating, notes,
			   still_wine_l, sparkling_l, juice_l,
			   sold_kg, discarded_kg
		FROM harvest_records
		WHERE id = $1 AND deleted_at IS NULL
	`, int64(recordID)).Scan(
		&harvest.ID, &harvest.BlockID, &harvest.HarvestYear, &harvest.HarvestDate,
		&harvest.YieldKG, &harvest.Brix, &harvest.AcidGL, &harvest.VineHealthRating, &harvest.Notes,
		&harvest.StillWineL, &harvest.SparklingL, &harvest.JuiceL,
		&harvest.SoldKG, &harvest.DiscardedKG,
	)

	if err != nil {
		writeNotFound(w, "harvest record not found")
		return
	}

	writeJSON(w, http.StatusOK, harvest)
}
