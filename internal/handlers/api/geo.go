// Package api provides HTTP handlers for JSON API endpoints.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// GeoReverseRequest represents the JSON request body for geo reverse lookup.
type GeoReverseRequest struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// GeoReverseResponse represents the JSON response for geo reverse lookup.
type GeoReverseResponse struct {
	Name        string `json:"name"`
	County      string `json:"county"`
	Municipality string `json:"municipality"`
}

// GeoReverseHandler handles reverse geocoding API requests.
type GeoReverseHandler struct{}

// NewGeoReverseHandler creates a new geo reverse handler.
func NewGeoReverseHandler() *GeoReverseHandler {
	return &GeoReverseHandler{}
}

// HandlePOST handles POST /api/geo/reverse
func (h *GeoReverseHandler) HandlePOST(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse request body
	var req GeoReverseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(GeoReverseResponse{
			Name:        "",
			County:      "",
			Municipality: "",
		})
		return
	}

	// Validate coordinates (rough check: lat -90 to 90, lon -180 to 180)
	if req.Lat < -90 || req.Lat > 90 || req.Lon < -180 || req.Lon > 180 {
		json.NewEncoder(w).Encode(GeoReverseResponse{
			Name:        "",
			County:      "",
			Municipality: "",
		})
		return
	}

	// Call Nominatim reverse geocoding API
	result, err := h.reverseGeocode(r.Context(), req.Lat, req.Lon)
	if err != nil {
		slog.Error("geo: reverse geocode", "err", err, "lat", req.Lat, "lon", req.Lon)
		// Graceful failure — return empty results
		json.NewEncoder(w).Encode(GeoReverseResponse{
			Name:        "",
			County:      "",
			Municipality: "",
		})
		return
	}

	json.NewEncoder(w).Encode(result)
}

// reverseGeocode calls Nominatim API and parses the response.
func (h *GeoReverseHandler) reverseGeocode(ctx context.Context, lat, lon float64) (GeoReverseResponse, error) {
	// Build Nominatim request with Swedish language
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?lat=%.6f&lon=%.6f&format=json&accept-language=sv", lat, lon)

	// Create HTTP client with timeout (Nominatim is external, can be slow)
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Create request with required User-Agent header (Nominatim policy)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return GeoReverseResponse{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "SvensktVin/1.0")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return GeoReverseResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return GeoReverseResponse{}, fmt.Errorf("nominatim returned status %d", resp.StatusCode)
	}

	// Parse response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GeoReverseResponse{}, fmt.Errorf("read response: %w", err)
	}

	// Nominatim response structure (simplified)
	var nominatimResp struct {
		Address struct {
			City         string `json:"city"`
			Town         string `json:"town"`
			Municipality string `json:"municipality"`
			County       string `json:"county"`
			Country      string `json:"country"`
			CountryCode  string `json:"country_code"`
		} `json:"address"`
		Name string `json:"name"`
	}

	if err := json.Unmarshal(body, &nominatimResp); err != nil {
		return GeoReverseResponse{}, fmt.Errorf("parse nominatim response: %w", err)
	}

	// Extract location name
	name := nominatimResp.Name

	// Extract county (län in Swedish)
	county := nominatimResp.Address.County
	if county == "" {
		county = nominatimResp.Address.City
	}
	if county == "" {
		county = nominatimResp.Address.Town
	}

	// Extract municipality
	municipality := nominatimResp.Address.Municipality
	if municipality == "" {
		municipality = nominatimResp.Address.Town
	}

	return GeoReverseResponse{
		Name:        name,
		County:      county,
		Municipality: municipality,
	}, nil
}
