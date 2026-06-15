// Package pages provides HTTP handlers for page rendering (auth, vineyard, blocks, harvest, etc.).
package pages

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/svensktvin/svensktvin/internal/auth"
	"github.com/svensktvin/svensktvin/internal/db"
)

// HarvestHandler holds dependencies for harvest page handlers.
type HarvestHandler struct {
	store      *db.Store
	sessionMgr *auth.SessionManager
}

// NewHarvestHandler creates a new harvest handler.
func NewHarvestHandler(store *db.Store, sessionMgr *auth.SessionManager) *HarvestHandler {
	return &HarvestHandler{
		store:      store,
		sessionMgr: sessionMgr,
	}
}

// HarvestLockHandler holds dependencies for block lock handlers.
type HarvestLockHandler struct {
	store      *db.Store
	sessionMgr *auth.SessionManager
}

// NewHarvestLockHandler creates a new block lock handler.
func NewHarvestLockHandler(store *db.Store, sessionMgr *auth.SessionManager) *HarvestLockHandler {
	return &HarvestLockHandler{
		store:      store,
		sessionMgr: sessionMgr,
	}
}

// routeHarvestRequest is a catch-all handler for harvest routes.
func (h *HarvestHandler) routeHarvestRequest(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// New harvest form: GET /vineyard/{id}/harvest/new
		if r.Method == http.MethodGet && strings.HasSuffix(path, "/harvest/new") {
			h.handleHarvestNewGET(tmpl).ServeHTTP(w, r)
			return
		}

		// Create harvest: POST /vineyard/{id}/harvest/new
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/harvest/new") {
			h.handleHarvestNewPOST(tmpl).ServeHTTP(w, r)
			return
		}

		// Edit harvest form: GET /vineyard/{id}/harvest/{id}/edit
		if r.Method == http.MethodGet && strings.HasSuffix(path, "/edit") {
			h.handleHarvestEditGET(tmpl).ServeHTTP(w, r)
			return
		}

		// Update harvest: POST /vineyard/{id}/harvest/{id}/edit
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/edit") {
			h.handleHarvestEditPOST(tmpl).ServeHTTP(w, r)
			return
		}

		// Not a harvest route — fall through
		http.NotFound(w, r)
	}
}

// harvestBlockLock holds block lock information for the harvest form.
type harvestBlockLock struct {
	ID        int64     `json:"id"`
	BlockID   int64     `json:"block_id"`
	UserID    int64     `json:"user_id"`
	CreatedAt string    `json:"created_at"`
	ExpiresAt string    `json:"expires_at"`
}

// handleHarvestNewGET renders the new harvest form.
func (h *HarvestHandler) handleHarvestNewGET(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth check
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extract vineyard ID from path: /vineyard/{id}/harvest/new
		path := r.URL.Path
		var vineyardID int64
		_, err := fmt.Sscanf(path, "/vineyard/%d/harvest/new", &vineyardID)
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
			slog.Error("harvest: get vineyard", "err", err, "vineyard_id", vineyardID)
			http.NotFound(w, r)
			return
		}

		// Load blocks for selector
		blocks, err := h.store.ListBlocks(r.Context(), vineyardID)
		if err != nil {
			slog.Error("harvest: list blocks", "err", err, "vineyard_id", vineyardID)
			blocks = nil
		}

		// Check block lock status
		var lock *harvestBlockLock
		blockIDStr := r.URL.Query().Get("block_id")
		if blockIDStr != "" {
			blockID, _ := strconv.ParseInt(blockIDStr, 10, 64)
			if blockID > 0 {
				blocksLock, err := h.store.GetBlockLock(r.Context(), blockID)
				if err != nil {
					slog.Error("harvest: get block lock", "err", err, "block_id", blockID)
				} else if blocksLock != nil {
					lock = &harvestBlockLock{
						ID:        blocksLock.ID,
						BlockID:   blocksLock.BlockID,
						UserID:    blocksLock.UserID,
						CreatedAt: blocksLock.CreatedAt.Format(time.RFC3339),
						ExpiresAt: blocksLock.ExpiresAt.Format(time.RFC3339),
					}
				}
			}
		}

		csrfToken := generateCSRFToken()
		setCSRFCookie(w, csrfToken)

		var lockJSON string
		if lock != nil {
			lj, _ := json.Marshal(lock)
			lockJSON = string(lj)
		}

		data := map[string]any{
			"User":        user,
			"Role":        role,
			"Vineyard":    *vineyard,
			"Blocks":      blocks,
			"Lock":        lock,
			"LockJSON":    lockJSON,
			"CSRFToken":   csrfToken,
			"Title":       fmt.Sprintf("Ny skörd — %s — Svenskt Vin", vineyard.Name),
			"BlockID":     blockIDStr,
		}
		renderTemplate(w, tmpl, "vineyard/harvest/new.html", data)
	}
}

// handleHarvestNewPOST processes new harvest form submission.
func (h *HarvestHandler) handleHarvestNewPOST(tmpl *template.Template) http.HandlerFunc {
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
		_, err := fmt.Sscanf(path, "/vineyard/%d/harvest/new", &vineyardID)
		if err != nil || vineyardID == 0 {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// CSRF validation
		if !validateCSRFToken(r) {
			data := map[string]any{
				"Error":     "Ogiltig begäran. Försök igen.",
				"VineyardID": vineyardID,
				"CSRFToken": "",
			}
			renderTemplate(w, tmpl, "vineyard/harvest/new.html", data)
			return
		}

		// Parse form values
		blockIDStr := sanitizeInput(r.FormValue("block_id"))
		harvestDateStr := sanitizeInput(r.FormValue("harvest_date"))
		harvestYearStr := sanitizeInput(r.FormValue("harvest_year"))
		yieldKgStr := sanitizeInput(r.FormValue("yield_kg"))
		brixStr := sanitizeInput(r.FormValue("brix"))
		acidGStr := sanitizeInput(r.FormValue("acid_g_l"))
		vineHealthStr := sanitizeInput(r.FormValue("vine_health_rating"))
		notes := sanitizeInput(r.FormValue("notes"))
		stillWineStr := sanitizeInput(r.FormValue("still_wine_l"))
		sparklingStr := sanitizeInput(r.FormValue("sparkling_l"))
		juiceStr := sanitizeInput(r.FormValue("juice_l"))
		soldStr := sanitizeInput(r.FormValue("sold_kg"))
		discardedStr := sanitizeInput(r.FormValue("discarded_kg"))

		// Parse numeric fields
		var blockID int64
		if blockIDStr != "" {
			blockID, _ = strconv.ParseInt(blockIDStr, 10, 64)
		}
		var harvestYear int
		if harvestYearStr != "" {
			harvestYear, _ = strconv.Atoi(harvestYearStr)
		}
		var yieldKg float64
		if yieldKgStr != "" {
			yieldKg, _ = strconv.ParseFloat(yieldKgStr, 64)
		}
		var harvestDate *time.Time
		if harvestDateStr != "" {
			t, err := time.Parse("2006-01-02", harvestDateStr)
			if err == nil {
				harvestDate = &t
			}
		}
		brix := parseOptionalFloat(brixStr, 0, 100)
		acidGL := parseOptionalFloat(acidGStr, 0, 50)
		vineHealth := parseOptionalIntRange(vineHealthStr, 1, 5)
		stillWine := parseOptionalFloat(stillWineStr, 0, 100000)
		sparkling := parseOptionalFloat(sparklingStr, 0, 100000)
		juice := parseOptionalFloat(juiceStr, 0, 100000)
		sold := parseOptionalFloat(soldStr, 0, 100000)
		discarded := parseOptionalFloat(discardedStr, 0, 100000)

		// Validate required fields
		var fieldErrors []ValidationError
		if blockID <= 0 {
			fieldErrors = append(fieldErrors, ValidationError{Field: "block_id", Issue: "Block krävs."})
		}
		if harvestDate == nil {
			fieldErrors = append(fieldErrors, ValidationError{Field: "harvest_date", Issue: "Skördedatum krävs."})
		}
		if harvestYear <= 0 {
			fieldErrors = append(fieldErrors, ValidationError{Field: "harvest_year", Issue: "Skördeår krävs."})
		}
		if yieldKg <= 0 {
			fieldErrors = append(fieldErrors, ValidationError{Field: "yield_kg", Issue: "Skördeavkastning krävs och måste vara större än 0."})
		}

		// Check block belongs to vineyard
		if blockID > 0 {
			result, err := h.store.GetBlock(r.Context(), blockID, vineyardID)
			if err != nil || result == nil {
				fieldErrors = append(fieldErrors, ValidationError{Field: "block_id", Issue: "Ogiltigt block."})
				blockID = 0
			}
		}

		if len(fieldErrors) > 0 {
			data := map[string]any{
				"User":           user,
				"VineyardID":     vineyardID,
				"Error":          "Vänligen fyll i alla obligatoriska fält.",
				"FieldErrors":    fieldErrors,
				"CSRFToken":      generateCSRFToken(),
				"Title":          "Ny skörd — Svenskt Vin",
				"BlockID":        blockIDStr,
				"HarvestDate":    harvestDateStr,
				"HarvestYear":    harvestYearStr,
				"YieldKg":        yieldKgStr,
				"Brix":           brixStr,
				"AcidGL":         acidGStr,
				"VineHealth":     vineHealthStr,
				"Notes":          notes,
				"StillWine":      stillWineStr,
				"Sparkling":      sparklingStr,
				"Juice":          juiceStr,
				"Sold":           soldStr,
				"Discarded":      discardedStr,
			}
			renderTemplate(w, tmpl, "vineyard/harvest/new.html", data)
			return
		}

		// Check block lock status
		blocksLock, err := h.store.GetBlockLock(r.Context(), blockID)
		if err != nil {
			slog.Error("harvest: check lock", "err", err, "block_id", blockID)
		}
		if blocksLock != nil {
			// Lock is still active — allow creation
			_ = blocksLock
		}
		// If blocksLock is nil, no lock exists — also allow creation

		// Check for duplicate harvest for this block + year
		harvests, err := h.store.GetHarvestsByBlock(r.Context(), blockID)
		if err == nil {
			for _, existing := range harvests {
				if existing.HarvestYear != nil && *existing.HarvestYear == harvestYear {
					fieldErrors := []ValidationError{
						{Field: "harvest_year", Issue: fmt.Sprintf("En skörd från år %d finns redan för detta block.", harvestYear)},
					}
					data := map[string]any{
						"User": user,
						"Vineyard": vineyardID,
						"Error": "Vänligen fyll i alla obligatoriska fält.",
						"FieldErrors": fieldErrors,
						"CSRFToken": generateCSRFToken(),
						"Title": "Ny skörd — Svenskt Vin",
						"BlockID": blockIDStr,
						"HarvestDate": harvestDateStr,
						"HarvestYear": harvestYearStr,
						"YieldKg": yieldKgStr,
					}
					renderTemplate(w, tmpl, "vineyard/harvest/new.html", data)
					return
				}
			}
		}

		// Create harvest
		harvestInput := &db.HarvestCreateInput{
			BlockID:          blockID,
			VineyardID:       vineyardID,
			UserID:           user.ID,
			HarvestDate:      harvestDate,
			HarvestYear:      harvestYear,
			YieldKG:          yieldKg,
			Brix:             brix,
			AcidgL:           acidGL,
			VineHealthRating: vineHealth,
			Notes:            &notes,
			StillWineL:       stillWine,
			SparklingL:       sparkling,
			JuiceL:           juice,
			SoldKG:           sold,
			DiscardedKG:      discarded,
		}

		_, err = h.store.CreateHarvest(r.Context(), harvestInput)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				data := map[string]any{
					"User": user, "VineyardID": vineyardID,
					"Error": fmt.Sprintf("En skörd från år %d finns redan för detta block.", harvestYear),
					"CSRFToken": generateCSRFToken(),
					"Title": "Ny skörd — Svenskt Vin",
				}
				renderTemplate(w, tmpl, "vineyard/harvest/new.html", data)
				return
			}
			slog.Error("harvest: create", "err", err, "vineyard_id", vineyardID)
			http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
			return
		}

		// Clear block lock if it existed
		if blocksLock != nil {
			_ = h.store.DeleteBlockLock(r.Context(), blockID)
		}

		// Success — redirect to vineyard dashboard
		w.Header().Set("HX-Redirect", fmt.Sprintf("/vineyard/%d", vineyardID))
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleHarvestEditGET renders the edit harvest form.
func (h *HarvestHandler) handleHarvestEditGET(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth check
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extract vineyard ID and harvest ID from path: /vineyard/{id}/harvest/{harvestId}/edit
		path := r.URL.Path
		var vineyardID, harvestID int64
		_, err := fmt.Sscanf(path, "/vineyard/%d/harvest/%d/edit", &vineyardID, &harvestID)
		if err != nil || vineyardID == 0 || harvestID == 0 {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// Check membership
		role, err := h.store.GetVineyardRole(r.Context(), vineyardID, user.ID)
		if err != nil || role == "" {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// Load harvest record
		harvest, err := h.store.GetHarvest(r.Context(), harvestID, vineyardID)
		if err != nil {
			slog.Error("harvest: get harvest", "err", err, "harvest_id", harvestID)
			http.NotFound(w, r)
			return
		}

		// Load vineyard details
		vineyard, err := h.store.GetVineyard(r.Context(), vineyardID)
		if err != nil {
			slog.Error("harvest: get vineyard", "err", err, "vineyard_id", vineyardID)
			http.NotFound(w, r)
			return
		}

		// Load blocks for selector
		blocks, err := h.store.ListBlocks(r.Context(), vineyardID)
		if err != nil {
			slog.Error("harvest: list blocks", "err", err, "vineyard_id", vineyardID)
			blocks = nil
		}

		csrfToken := generateCSRFToken()
		setCSRFCookie(w, csrfToken)

		data := map[string]any{
			"User":        user,
			"Role":        role,
			"Vineyard":    *vineyard,
			"Harvest":     harvest,
			"Blocks":      blocks,
			"CSRFToken":   csrfToken,
			"Title":       fmt.Sprintf("Redigera skörd — %s — Svenskt Vin", vineyard.Name),
		}
		renderTemplate(w, tmpl, "vineyard/harvest/edit.html", data)
	}
}

// handleHarvestEditPOST processes edit harvest form submission.
func (h *HarvestHandler) handleHarvestEditPOST(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth check
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extract vineyard ID and harvest ID
		path := r.URL.Path
		var vineyardID, harvestID int64
		_, err := fmt.Sscanf(path, "/vineyard/%d/harvest/%d/edit", &vineyardID, &harvestID)
		if err != nil || vineyardID == 0 || harvestID == 0 {
			http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
			return
		}

		// CSRF validation
		if !validateCSRFToken(r) {
			harvest, _ := h.store.GetHarvest(r.Context(), harvestID, vineyardID)
			if harvest == nil {
				http.NotFound(w, r)
				return
			}
			data := map[string]any{
				"User": user, "Vineyard": harvest, "Harvest": harvest,
				"Error": "Ogiltig begäran. Försök igen.",
				"CSRFToken": "",
				"Title": "Redigera skörd — Svenskt Vin",
			}
			renderTemplate(w, tmpl, "vineyard/harvest/edit.html", data)
			return
		}

		// Parse form values
		blockIDStr := sanitizeInput(r.FormValue("block_id"))
		harvestDateStr := sanitizeInput(r.FormValue("harvest_date"))
		harvestYearStr := sanitizeInput(r.FormValue("harvest_year"))
		yieldKgStr := sanitizeInput(r.FormValue("yield_kg"))
		brixStr := sanitizeInput(r.FormValue("brix"))
		acidGStr := sanitizeInput(r.FormValue("acid_g_l"))
		vineHealthStr := sanitizeInput(r.FormValue("vine_health_rating"))
		notes := sanitizeInput(r.FormValue("notes"))
		stillWineStr := sanitizeInput(r.FormValue("still_wine_l"))
		sparklingStr := sanitizeInput(r.FormValue("sparkling_l"))
		juiceStr := sanitizeInput(r.FormValue("juice_l"))
		soldStr := sanitizeInput(r.FormValue("sold_kg"))
		discardedStr := sanitizeInput(r.FormValue("discarded_kg"))

		// Parse numeric fields
		var blockID int64
		if blockIDStr != "" {
			blockID, _ = strconv.ParseInt(blockIDStr, 10, 64)
		}
		var harvestYear int
		if harvestYearStr != "" {
			harvestYear, _ = strconv.Atoi(harvestYearStr)
		}
		var yieldKg float64
		if yieldKgStr != "" {
			yieldKg, _ = strconv.ParseFloat(yieldKgStr, 64)
		}
		var harvestDate *time.Time
		if harvestDateStr != "" {
			t, err := time.Parse("2006-01-02", harvestDateStr)
			if err == nil {
				harvestDate = &t
			}
		}
		brix := parseOptionalFloat(brixStr, 0, 100)
		acidGL := parseOptionalFloat(acidGStr, 0, 50)
		vineHealth := parseOptionalIntRange(vineHealthStr, 1, 5)
		stillWine := parseOptionalFloat(stillWineStr, 0, 100000)
		sparkling := parseOptionalFloat(sparklingStr, 0, 100000)
		juice := parseOptionalFloat(juiceStr, 0, 100000)
		sold := parseOptionalFloat(soldStr, 0, 100000)
		discarded := parseOptionalFloat(discardedStr, 0, 100000)

		// Validate required fields
		var fieldErrors []ValidationError
		if blockID <= 0 {
			fieldErrors = append(fieldErrors, ValidationError{Field: "block_id", Issue: "Block krävs."})
		}
		if harvestDate == nil {
			fieldErrors = append(fieldErrors, ValidationError{Field: "harvest_date", Issue: "Skördedatum krävs."})
		}
		if harvestYear <= 0 {
			fieldErrors = append(fieldErrors, ValidationError{Field: "harvest_year", Issue: "Skördeår krävs."})
		}
		if yieldKg <= 0 {
			fieldErrors = append(fieldErrors, ValidationError{Field: "yield_kg", Issue: "Skördeavkastning krävs och måste vara större än 0."})
		}

		// Check block belongs to vineyard
		if blockID > 0 {
			_, err := h.store.GetBlock(r.Context(), blockID, vineyardID)
			if err != nil {
				fieldErrors = append(fieldErrors, ValidationError{Field: "block_id", Issue: "Ogiltigt block."})
				blockID = 0
			}
		}

		if len(fieldErrors) > 0 {
			harvest, _ := h.store.GetHarvest(r.Context(), harvestID, vineyardID)
			if harvest == nil {
				http.NotFound(w, r)
				return
			}
			data := map[string]any{
				"User": user, "Vineyard": harvest, "Harvest": harvest,
				"Error": "Vänligen fyll i alla obligatoriska fält.",
				"FieldErrors": fieldErrors,
				"CSRFToken": generateCSRFToken(),
				"Title": "Redigera skörd — Svenskt Vin",
			}
			renderTemplate(w, tmpl, "vineyard/harvest/edit.html", data)
			return
		}

		// Load harvest for error context
		harvest, _ := h.store.GetHarvest(r.Context(), harvestID, vineyardID)

		// Check for duplicate harvest for this block + year (excluding current record)
		harvests, _ := h.store.GetHarvestsByBlock(r.Context(), blockID)
		for _, existing := range harvests {
			if existing.ID != harvestID && existing.HarvestYear != nil && *existing.HarvestYear == harvestYear {
				fieldErrors := []ValidationError{
					{Field: "harvest_year", Issue: fmt.Sprintf("En skörd från år %d finns redan för detta block.", harvestYear)},
				}
				ctxData := map[string]any{
					"User": user,
					"Error": "Vänligen fyll i alla obligatoriska fält.",
					"FieldErrors": fieldErrors,
					"CSRFToken": generateCSRFToken(),
					"Title": "Redigera skörd — Svenskt Vin",
				}
				if harvest != nil {
					ctxData["Vineyard"] = harvest
					ctxData["Harvest"] = harvest
				}
				renderTemplate(w, tmpl, "vineyard/harvest/edit.html", ctxData)
				return
			}
		}

		// Build update input (only non-zero/nil fields)
		updateInput := &db.HarvestUpdateInput{
			HarvestDate:      harvestDate,
			HarvestYear:      &harvestYear,
			YieldKG:          &yieldKg,
			Brix:             brix,
			AcidgL:           acidGL,
			VineHealthRating: vineHealth,
			Notes:            &notes,
			StillWineL:       stillWine,
			SparklingL:       sparkling,
			JuiceL:           juice,
			SoldKG:           sold,
			DiscardedKG:      discarded,
		}

		err = h.store.UpdateHarvest(r.Context(), harvestID, vineyardID, updateInput)
		if err != nil {
			slog.Error("harvest: update", "err", err, "harvest_id", harvestID)
			http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
			return
		}

		// Success — redirect to vineyard dashboard
		w.Header().Set("HX-Redirect", fmt.Sprintf("/vineyard/%d", vineyardID))
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleHarvestLockPOST acquires a lock on a block for harvest creation.
func (h *HarvestLockHandler) handleHarvestLockPOST(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth check
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extract vineyard ID and block ID from path: /vineyard/{id}/blocks/{blockId}/harvest/lock
		path := r.URL.Path
		var vineyardID, blockID int64
		_, err := fmt.Sscanf(path, "/vineyard/%d/blocks/%d/harvest/lock", &vineyardID, &blockID)
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

		// Check if block already locked
		existingLock, err := h.store.GetBlockLock(r.Context(), blockID)
		if err != nil {
			slog.Error("harvest: check lock", "err", err, "block_id", blockID)
		}
		if existingLock != nil {
			// Block already locked — redirect to harvest new with lock info
			w.Header().Set("HX-Redirect", fmt.Sprintf("/vineyard/%d/harvest/new?block_id=%d", vineyardID, blockID))
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Create lock
		lockInput := &db.BlockLockInput{
			BlockID: blockID,
			UserID:  user.ID,
		}
		ttlMinutes := 30 // Default TTL (configurable)

		err = h.store.CreateBlockLock(r.Context(), lockInput, ttlMinutes)
		if err != nil {
			slog.Error("harvest: create lock", "err", err, "block_id", blockID)
			http.Error(w, "Ett internt fel uppstod.", http.StatusInternalServerError)
			return
		}

		// Redirect to harvest new with block_id pre-filled
		w.Header().Set("HX-Redirect", fmt.Sprintf("/vineyard/%d/harvest/new?block_id=%d", vineyardID, blockID))
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleHarvestUnlockPOST releases a block lock.
func (h *HarvestLockHandler) handleHarvestUnlockPOST(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth check
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

	// Extract vineyard ID and block ID from path: /vineyard/{id}/blocks/{blockId}/harvest/lock
	path := r.URL.Path
	var vineyardID, blockID int64
	_, err := fmt.Sscanf(path, "/vineyard/%d/blocks/%d/harvest/lock", &vineyardID, &blockID)
	if err != nil || vineyardID == 0 || blockID == 0 {
		http.Redirect(w, r, "/vineyard", http.StatusSeeOther)
		return
	}

	// Delete lock
	err = h.store.DeleteBlockLock(r.Context(), blockID)
	if err != nil {
		slog.Error("harvest: delete lock", "err", err, "block_id", blockID)
	}

	// Redirect to vineyard dashboard
	w.Header().Set("HX-Redirect", fmt.Sprintf("/vineyard/%d", vineyardID))
	w.WriteHeader(http.StatusNoContent)
}
}

// routeHarvestLockRequest is a catch-all handler for harvest lock routes.
func (h *HarvestLockHandler) routeHarvestLockRequest(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Extend lock: GET/POST /vineyard/{id}/blocks/{id}/harvest/lock/extend
		if r.Method == http.MethodGet && strings.HasSuffix(path, "/harvest/lock/extend") {
			h.handleHarvestExtendPOST(tmpl).ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/harvest/lock/extend") {
			h.handleHarvestExtendPOST(tmpl).ServeHTTP(w, r)
			return
		}

		// Unlock: POST /vineyard/{id}/blocks/{id}/harvest/lock
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/harvest/lock") {
			h.handleHarvestUnlockPOST(tmpl).ServeHTTP(w, r)
			return
		}

		// Lock: POST /vineyard/{id}/blocks/{id}/harvest/lock
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/harvest/lock") {
			h.handleHarvestLockPOST(tmpl).ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	}
}

// handleHarvestExtendPOST extends a block lock TTL.
func (h *HarvestLockHandler) handleHarvestExtendPOST(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Auth check
		user := h.sessionMgr.SessionFromRequest(r)
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Extract vineyard ID and block ID from path: /vineyard/{id}/blocks/{blockId}/harvest/lock/extend
		path := r.URL.Path
		var vineyardID, blockID int64
		_, err := fmt.Sscanf(path, "/vineyard/%d/blocks/%d/harvest/lock/extend", &vineyardID, &blockID)
		if err != nil || vineyardID == 0 || blockID == 0 {
			http.NotFound(w, r)
			return
		}

		// Extend lock
		ttlMinutes := 30 // Default TTL
		err = h.store.ExtendBlockLock(r.Context(), blockID, ttlMinutes)
		if err != nil {
			slog.Error("harvest: extend lock", "err", err, "block_id", blockID)
			http.NotFound(w, r)
			return
		}

		// Return 200 with empty body and HX-Trigger header
		w.Header().Set("HX-Trigger", "lock-extended")
		w.WriteHeader(http.StatusOK)
	}
}
