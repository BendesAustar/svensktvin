// Package pages provides HTTP handlers for page rendering (auth, vineyard, blocks, etc.).
package pages

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/svensktvin/svensktvin/internal/auth"
	"github.com/svensktvin/svensktvin/internal/config"
	"github.com/svensktvin/svensktvin/internal/db"
)

// BlockHandler holds dependencies for block page handlers.
type BlockHandler struct {
	store      *db.Store
	sessionMgr *auth.SessionManager
	cookieCfg  config.CookieConfig
}

// NewBlockHandler creates a new block handler.
func NewBlockHandler(store *db.Store, sessionMgr *auth.SessionManager, cookieCfg config.CookieConfig) *BlockHandler {
	return &BlockHandler{
		store:      store,
		sessionMgr: sessionMgr,
		cookieCfg:  cookieCfg,
	}
}

// routeBlockRequest is a single catch-all handler for block routes. It parses the path
// and delegates to the appropriate handler.
func (h *BlockHandler) routeBlockRequest(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// New block form: GET /vineyard/{id}/blocks/new
		if r.Method == http.MethodGet && strings.HasSuffix(path, "/blocks/new") {
			h.handleBlockNewGET(tmpl).ServeHTTP(w, r)
			return
		}

		// Create block: POST /vineyard/{id}/blocks/new
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/blocks/new") {
			h.handleBlockNewPOST(tmpl).ServeHTTP(w, r)
			return
		}

		// Edit block form: GET /vineyard/{id}/blocks/{blockId}/edit
		if r.Method == http.MethodGet && strings.HasSuffix(path, "/edit") {
			h.handleBlockEditGET(tmpl).ServeHTTP(w, r)
			return
		}

		// Update block: POST /vineyard/{id}/blocks/{blockId}/edit
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/edit") {
			h.handleBlockEditPOST(tmpl).ServeHTTP(w, r)
			return
		}

		// Not a block route — fall through
		http.NotFound(w, r)
	}
}

// handleBlockNewGET renders the new block form.
func (h *BlockHandler) handleBlockNewGET(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth check
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extract vineyard ID from path: /vineyard/{id}/blocks/new
		path := r.URL.Path
		var vineyardID int64
		_, err := fmt.Sscanf(path, "/vineyard/%d/blocks/new", &vineyardID)
		if err != nil || vineyardID == 0 {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// Check membership (editor+ required)
		role, err := h.store.GetVineyardRole(r.Context(), vineyardID, user.ID)
		if err != nil || role == "" {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// Load vineyard details
		vineyard, err := h.store.GetVineyard(r.Context(), vineyardID)
		if err != nil {
			slog.Error("blocks: get vineyard", "err", err, "vineyard_id", vineyardID)
			http.NotFound(w, r)
			return
		}

		csrfToken := generateCSRFToken()
		setCSRFCookie(w, csrfToken, h.cookieCfg)

		data := map[string]any{
			"User": user,
			"IsAdmin": user.IsAdmin,
			"Role":     role,
			"Vineyard": *vineyard,
			"CSRFToken": csrfToken,
			"Title":    fmt.Sprintf("Nytt block — %s — Svenskt Vin", vineyard.Name),
		}
		renderTemplate(w, tmpl, "vineyard/blocks/new.html", data)
	}
}

// handleBlockNewPOST processes new block form submission.
func (h *BlockHandler) handleBlockNewPOST(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth check
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extract vineyard ID
		path := r.URL.Path
		var vineyardID int64
		_, err := fmt.Sscanf(path, "/vineyard/%d/blocks/new", &vineyardID)
		if err != nil || vineyardID == 0 {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// CSRF validation
		if !validateCSRFToken(r) {
			data := map[string]any{
				"Error": "Ogiltig begäran. Försök igen.",
				"VineyardID": vineyardID,
				"CSRFToken": "",
			}
			renderTemplate(w, tmpl, "vineyard/blocks/new.html", data)
			return
		}

		// Parse form values
		blockName := sanitizeInput(r.FormValue("block_name"))
		varietyIDStr := r.FormValue("variety_id")
		varietyName := sanitizeInput(r.FormValue("variety_name"))
		areaHA, err := strconv.ParseFloat(r.FormValue("area_ha"), 64)
		if err != nil || areaHA <= 0 {
			areaHA = 0
		}
		vineCount := parseOptionalInt(r.FormValue("vine_count"))
		plantingYear := parseOptionalIntRange(r.FormValue("planting_year"), 1800, 2030)
		trainingSystem := parseOptionalString(r.FormValue("training_system"))
		aspect := parseOptionalString(r.FormValue("aspect"))
		slopeDegrees := parseOptionalFloat(r.FormValue("slope_degrees"), 0, 90)
		elevationM := parseOptionalInt(r.FormValue("elevation_m"))

		// Validate required fields
		var fieldErrors []ValidationError
		if blockName == "" {
			fieldErrors = append(fieldErrors, ValidationError{Field: "block_name", Issue: "Blocknamn krävs."})
		}
		if areaHA <= 0 {
			fieldErrors = append(fieldErrors, ValidationError{Field: "area_ha", Issue: "Area krävs och måste vara större än 0."})
		}
		if varietyIDStr == "" && varietyName == "" {
			fieldErrors = append(fieldErrors, ValidationError{Field: "variety", Issue: "Sort krävs — sök och välj en sort eller ange eget namn."})
		}

		if len(fieldErrors) > 0 {
			data := map[string]any{
				"User": user,
			"IsAdmin": user.IsAdmin,
				"VineyardID":     vineyardID,
				"Error":          "Vänligen fyll i alla obligatoriska fält.",
				"FieldErrors":    fieldErrors,
				"CSRFToken":      generateCSRFToken(),
				"Title":          fmt.Sprintf("Nytt block — Svenskt Vin"),
				// Pre-fill form values
				"BlockName":      blockName,
				"VarietyName":    varietyName,
				"AreaHA":         areaHA,
				"VineCount":      vineCount,
				"PlantingYear":   plantingYear,
				"TrainingSystem": trainingSystem,
				"Aspect":         aspect,
				"SlopeDegrees":   slopeDegrees,
				"ElevationM":     elevationM,
			}
			renderTemplate(w, tmpl, "vineyard/blocks/new.html", data)
			return
		}

		// Resolve variety ID
		var varietyID int64
		if varietyIDStr != "" {
			// Alpine.js provided variety_id
			id, parseErr := strconv.ParseInt(varietyIDStr, 10, 64)
			if parseErr != nil || id <= 0 {
				fieldErrors = append(fieldErrors, ValidationError{Field: "variety", Issue: "Ogiltig sort-ID."})
				data := map[string]any{
					"User": user,
			"IsAdmin": user.IsAdmin, "VineyardID": vineyardID,
					"Error": "Ogiltig sort. Välj en giltig sort.",
					"FieldErrors": fieldErrors,
					"CSRFToken": generateCSRFToken(),
					"Title": "Nytt block — Svenskt Vin",
				}
				renderTemplate(w, tmpl, "vineyard/blocks/new.html", data)
				return
			}
			varietyID = id
		} else if varietyName != "" {
			// Custom variety name — create or lookup
			varietyID, err = h.store.CreateOrLookupVariety(r.Context(), varietyName, vineyardID, user.ID)
			if err != nil {
				slog.Error("blocks: variety create/lookup", "err", err)
				http.Error(w, "Ett internt fel uppstod vid sortsökning.", http.StatusInternalServerError)
				return
			}
		}

		// Check membership (editor+ required)
		role, err := h.store.GetVineyardRole(r.Context(), vineyardID, user.ID)
		if err != nil || role == "" {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// Create block
		_, err = h.store.CreateBlock(r.Context(), vineyardID, varietyID, blockName, areaHA,
			vineCount, plantingYear, trainingSystem, aspect, slopeDegrees, elevationM)
		if err != nil {
			if contains(err.Error(), "duplicate key") {
				data := map[string]any{
					"User": user,
			"IsAdmin": user.IsAdmin, "VineyardID": vineyardID,
					"Error": "Ett block med detta namn finns redan i vingården.",
					"CSRFToken": generateCSRFToken(),
					"Title": "Nytt block — Svenskt Vin",
				}
				renderTemplate(w, tmpl, "vineyard/blocks/new.html", data)
				return
			}
			slog.Error("blocks: create", "err", err, "vineyard_id", vineyardID)
			http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
			return
		}

		// Success — redirect to vineyard dashboard
		w.Header().Set("HX-Redirect", fmt.Sprintf("/vineyard/%d", vineyardID))
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleBlockEditGET renders the edit block form.
func (h *BlockHandler) handleBlockEditGET(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth check
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extract vineyard ID and block ID from path: /vineyard/{id}/blocks/{blockId}/edit
		path := r.URL.Path
		var vineyardID, blockID int64
		_, err := fmt.Sscanf(path, "/vineyard/%d/blocks/%d/edit", &vineyardID, &blockID)
		if err != nil || vineyardID == 0 || blockID == 0 {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// Check membership
		role, err := h.store.GetVineyardRole(r.Context(), vineyardID, user.ID)
		if err != nil || role == "" {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// Load block with variety info
		result, err := h.store.GetBlock(r.Context(), blockID, vineyardID)
		if err != nil {
			slog.Error("blocks: get block", "err", err, "block_id", blockID)
			http.NotFound(w, r)
			return
		}

		// Load vineyard details
		vineyard, err := h.store.GetVineyard(r.Context(), vineyardID)
		if err != nil {
			slog.Error("blocks: get vineyard", "err", err, "vineyard_id", vineyardID)
			http.NotFound(w, r)
			return
		}

		csrfToken := generateCSRFToken()
		setCSRFCookie(w, csrfToken, h.cookieCfg)

		data := map[string]any{
			"User": user,
			"IsAdmin": user.IsAdmin,
			"Role":           role,
			"Vineyard":       *vineyard,
			"Block":          result.Block,
			"VarietyName":    result.VarietyName,
			"VarietyColor":   result.VarietyColor,
			"VarietyStatus":  result.Block.VarietyID > 0,
			"CSRFToken":      csrfToken,
			"Title":          fmt.Sprintf("Redigera block — %s — Svenskt Vin", vineyard.Name),
		}
		renderTemplate(w, tmpl, "vineyard/blocks/edit.html", data)
	}
}

// handleBlockEditPOST processes edit block form submission.
func (h *BlockHandler) handleBlockEditPOST(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth check
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extract vineyard ID and block ID
		path := r.URL.Path
		var vineyardID, blockID int64
		_, err := fmt.Sscanf(path, "/vineyard/%d/blocks/%d/edit", &vineyardID, &blockID)
		if err != nil || vineyardID == 0 || blockID == 0 {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// CSRF validation
		if !validateCSRFToken(r) {
			// Pre-load block data for re-render
			result, _ := h.store.GetBlock(r.Context(), blockID, vineyardID)
			if result == nil {
				http.NotFound(w, r)
				return
			}
			data := map[string]any{
				"User": user,
			"IsAdmin": user.IsAdmin, "Vineyard": result, "Block": result.Block,
				"Error": "Ogiltig begäran. Försök igen.",
				"CSRFToken": "",
				"Title": "Redigera block — Svenskt Vin",
			}
			renderTemplate(w, tmpl, "vineyard/blocks/edit.html", data)
			return
		}

		// Parse form values
		blockName := sanitizeInput(r.FormValue("block_name"))
		varietyIDStr := r.FormValue("variety_id")
		varietyName := sanitizeInput(r.FormValue("variety_name"))
		areaHA, err := strconv.ParseFloat(r.FormValue("area_ha"), 64)
		if err != nil || areaHA <= 0 {
			areaHA = 0
		}
		vineCount := parseOptionalInt(r.FormValue("vine_count"))
		plantingYear := parseOptionalIntRange(r.FormValue("planting_year"), 1800, 2030)
		trainingSystem := parseOptionalString(r.FormValue("training_system"))
		aspect := parseOptionalString(r.FormValue("aspect"))
		slopeDegrees := parseOptionalFloat(r.FormValue("slope_degrees"), 0, 90)
		elevationM := parseOptionalInt(r.FormValue("elevation_m"))

		// Validate required fields
		var fieldErrors []ValidationError
		if blockName == "" {
			fieldErrors = append(fieldErrors, ValidationError{Field: "block_name", Issue: "Blocknamn krävs."})
		}
		if areaHA <= 0 {
			fieldErrors = append(fieldErrors, ValidationError{Field: "area_ha", Issue: "Area krävs och måste vara större än 0."})
		}
		if varietyIDStr == "" && varietyName == "" {
			fieldErrors = append(fieldErrors, ValidationError{Field: "variety", Issue: "Sort krävs."})
		}

		if len(fieldErrors) > 0 {
			// Pre-load block data for re-render
			result, _ := h.store.GetBlock(r.Context(), blockID, vineyardID)
			if result == nil {
				http.NotFound(w, r)
				return
			}
			data := map[string]any{
				"User": user,
			"IsAdmin": user.IsAdmin, "Vineyard": result, "Block": result.Block,
				"Error": "Vänligen fyll i alla obligatoriska fält.",
				"FieldErrors": fieldErrors,
				"CSRFToken": generateCSRFToken(),
				"Title": "Redigera block — Svenskt Vin",
				// Pre-fill form values
				"BlockName":      blockName,
				"VarietyName":    varietyName,
				"AreaHA":         areaHA,
				"VineCount":      vineCount,
				"PlantingYear":   plantingYear,
				"TrainingSystem": trainingSystem,
				"Aspect":         aspect,
				"SlopeDegrees":   slopeDegrees,
				"ElevationM":     elevationM,
			}
			renderTemplate(w, tmpl, "vineyard/blocks/edit.html", data)
			return
		}

		// Resolve variety ID
		var varietyID int64
		if varietyIDStr != "" {
			id, parseErr := strconv.ParseInt(varietyIDStr, 10, 64)
			if parseErr != nil || id <= 0 {
				fieldErrors = append(fieldErrors, ValidationError{Field: "variety", Issue: "Ogiltig sort-ID."})
				result, _ := h.store.GetBlock(r.Context(), blockID, vineyardID)
				if result == nil {
					http.NotFound(w, r)
					return
				}
				data := map[string]any{
					"User": user,
			"IsAdmin": user.IsAdmin, "Vineyard": result, "Block": result.Block,
					"Error": "Ogiltig sort. Välj en giltig sort.",
					"FieldErrors": fieldErrors,
					"CSRFToken": generateCSRFToken(),
					"Title": "Redigera block — Svenskt Vin",
				}
				renderTemplate(w, tmpl, "vineyard/blocks/edit.html", data)
				return
			}
			varietyID = id
		} else if varietyName != "" {
			// Custom variety name — create or lookup
			varietyID, err = h.store.CreateOrLookupVariety(r.Context(), varietyName, vineyardID, user.ID)
			if err != nil {
				slog.Error("blocks: variety create/lookup", "err", err)
				http.Error(w, "Ett internt fel uppstod vid sortsökning.", http.StatusInternalServerError)
				return
			}
		}

		// Check membership (editor+ required)
		role, err := h.store.GetVineyardRole(r.Context(), vineyardID, user.ID)
		if err != nil || role == "" {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// Update block
		err = h.store.UpdateBlock(r.Context(), blockID, vineyardID, blockName, varietyID, areaHA,
			vineCount, plantingYear, trainingSystem, aspect, slopeDegrees, elevationM)
		if err != nil {
			if contains(err.Error(), "duplicate key") {
				result, _ := h.store.GetBlock(r.Context(), blockID, vineyardID)
				if result == nil {
					http.NotFound(w, r)
					return
				}
				data := map[string]any{
					"User": user,
			"IsAdmin": user.IsAdmin, "Vineyard": result, "Block": result.Block,
					"Error": "Ett block med detta namn finns redan i vingården.",
					"CSRFToken": generateCSRFToken(),
					"Title": "Redigera block — Svenskt Vin",
				}
				renderTemplate(w, tmpl, "vineyard/blocks/edit.html", data)
				return
			}
			slog.Error("blocks: update", "err", err, "block_id", blockID)
			http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
			return
		}

		// Success — redirect to vineyard dashboard
		w.Header().Set("HX-Redirect", fmt.Sprintf("/vineyard/%d", vineyardID))
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- Helper functions for parsing optional fields ---

func parseOptionalInt(s string) *int {
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return nil
	}
	return &n
}

func parseOptionalIntRange(s string, min, max int) *int {
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < min || n > max {
		return nil
	}
	return &n
}

func parseOptionalString(s string) *string {
	s = sanitizeInput(s)
	if s == "" {
		return nil
	}
	return &s
}

func parseOptionalFloat(s string, min, max float64) *float64 {
	if s == "" {
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < min || f > max {
		return nil
	}
	return &f
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}

// ValidationError represents a single field validation error.
type ValidationError struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}
