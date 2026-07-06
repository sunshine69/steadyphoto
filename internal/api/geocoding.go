package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

// GeocodeResult represents the response from Nominatim geocoding.
type GeocodeResult struct {
	PlaceID     int     `json:"place_id"`
	Displayname string  `json:"display_name"`
	Lat         string  `json:"lat"`
	Lon         string  `json:"lon"`
	Type        string  `json:"type"`
	PlaceRank   int     `json:"place_rank"`
	BoundingBox []string `json:"boundingbox"` // Nominatim returns strings!
}

// ReverseGeocodeResult represents the response from Nominatim reverse geocoding.
type ReverseGeocodeResult struct {
	DisplayName string `json:"display_name"`
	Latitude    string `json:"lat"`
	Longitude   string `json:"lon"`
	City        string `json:"city"`
	Suburb      string `json:"suburb"`
	County      string `json:"county"`
	State       string `json:"state"`
	Country     string `json:"country"`
}

// ReverseGeocode converts coordinates to a place name using Nominatim.
func ReverseGeocode(lat, lon float64) (*ReverseGeocodeResult, error) {
	apiURL := "https://nominatim.openstreetmap.org/reverse"
	params := url.Values{}
	params.Set("format", "json")
	params.Set("lat", fmt.Sprintf("%.6f", lat))
	params.Set("lon", fmt.Sprintf("%.6f", lon))
	params.Set("zoom", "10")
	params.Set("addressdetails", "1")

	fullURL := apiURL + "?" + params.Encode()

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to call Nominatim: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Nominatim returned status %d", resp.StatusCode)
	}

	var result ReverseGeocodeResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ForwardGeocode converts a place name to coordinates and bounding box using Nominatim.
func ForwardGeocode(placeName string) (*GeocodeResult, error) {
	apiURL := "https://nominatim.openstreetmap.org/search"
	params := url.Values{}
	params.Set("format", "json")
	params.Set("q", placeName)
	params.Set("limit", "1")
	params.Set("polygon", "1")

	fullURL := apiURL + "?" + params.Encode()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	// Nominatim requires a valid User-Agent (refuses requests without one)
	req.Header.Set("User-Agent", "SteadyPhoto/1.0 (https://steadyphoto.local)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Nominatim: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body := ""
		b := make([]byte, 200)
		n, _ := resp.Body.Read(b)
		if n > 0 {
			body = string(b[:n])
		}
		return nil, fmt.Errorf("Nominatim returned status %d: %s", resp.StatusCode, body)
	}

	var results []GeocodeResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for '%s'", placeName)
	}

	// Return the first (most relevant) result
	result := &results[0]

	// Parse bounding box if available
	if len(result.BoundingBox) >= 4 {
		log.Printf("[INFO] Geocoded '%s' to bounding box: [%s, %s, %s, %s]",
			placeName, result.BoundingBox[0], result.BoundingBox[1],
			result.BoundingBox[2], result.BoundingBox[3])
	}

	return result, nil
}

// ParseCoordinates converts string coordinates to float64.
func ParseCoordinates(latStr, lonStr string) (float64, float64, error) {
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse latitude: %w", err)
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse longitude: %w", err)
	}
	return lat, lon, nil
}

// ParseBoundingBox converts string coordinates to float64 bounding box.
func ParseBoundingBox(bbox []float64) (south, north, west, east float64, err error) {
	if len(bbox) < 4 {
		return 0, 0, 0, 0, fmt.Errorf("invalid bounding box format: %v", bbox)
	}

	south, err = strconv.ParseFloat(fmt.Sprintf("%f", bbox[0]), 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse south: %w", err)
	}
	north, err = strconv.ParseFloat(fmt.Sprintf("%f", bbox[1]), 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse north: %w", err)
	}
	west, err = strconv.ParseFloat(fmt.Sprintf("%f", bbox[2]), 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse west: %w", err)
	}
	east, err = strconv.ParseFloat(fmt.Sprintf("%f", bbox[3]), 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse east: %w", err)
	}

	return south, north, west, east, nil
}

// GeocodePlaceName searches for a place name and returns its coordinates.
// This is used by the "place" search scope to convert city/place names to coordinates.
func GeocodePlaceName(placeName string) (lat, lon float64, err error) {
	result, err := ForwardGeocode(placeName)
	if err != nil {
		return 0, 0, err
	}

	lat, lon, err = ParseCoordinates(result.Lat, result.Lon)
	if err != nil {
		return 0, 0, err
	}

	return lat, lon, nil
}
