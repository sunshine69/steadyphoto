package processor

import (
	"encoding/json"
	"fmt"
	"io"
	"github.com/jbrodriguez/mlog"
	"net/http"
	"strings"
	"time"
)

// NominatimResponse represents the response from OSM Nominatim reverse geocoding API.
type NominatimResponse struct {
	Address struct {
		City        string `json:"city"`
		Town        string `json:"town"`
		Village     string `json:"village"`
		County      string `json:"county"`
		State       string `json:"state"`
		Country     string `json:"country"`
		CountryCode string `json:"country_code"`
		Suburb      string `json:"suburb"`
		Region      string `json:"region"`
	} `json:"address"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

// ReverseGeocode resolves GPS coordinates to a human-readable location string
// using OpenStreetMap Nominatim API (free, no API key required).
func ReverseGeocode(latitude, longitude float64) (string, error) {
	if latitude == 0 && longitude == 0 {
		return "", fmt.Errorf("invalid coordinates: 0, 0")
	}

	apiURL := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?format=json&lat=%f&lon=%f&zoom=10&addressdetails=1", latitude, longitude)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		mlog.Info("[GEOCODE] Failed to call Nominatim API: %v", err)
		return "", fmt.Errorf("geocoding API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Nominatim API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result NominatimResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse Nominatim response: %w", err)
	}

	// Build a human-readable location string from most specific to least specific
	parts := []string{}
	if result.Address.City != "" {
		parts = append(parts, result.Address.City)
	} else if result.Address.Town != "" {
		parts = append(parts, result.Address.Town)
	} else if result.Address.Village != "" {
		parts = append(parts, result.Address.Village)
	} else if result.Address.Suburb != "" {
		parts = append(parts, result.Address.Suburb)
	}
	if result.Address.County != "" {
		parts = append(parts, result.Address.County)
	}
	if result.Address.State != "" {
		parts = append(parts, result.Address.State)
	}
	if result.Address.Country != "" {
		parts = append(parts, result.Address.Country)
	}

	location := strings.Join(parts, ", ")
	if location == "" {
		// Fallback to display_name
		location = result.DisplayName
	}

	return location, nil
}