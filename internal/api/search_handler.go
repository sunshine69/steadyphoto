package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// handleSearchMedia searches for media by text across filename, tags, and metadata.
// Supports scopes: name, tags, location, place, all
// For "place" scope, geocodes the place name and searches within its bounding box.
// For "location" scope, can search by coordinates or coordinate patterns.
func (s *Server) handleSearchMedia(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, ok := GetUserIDFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("query")
	scope := r.URL.Query().Get("scope")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	dateRange := r.URL.Query().Get("dateRange")

	// Parse limit and offset with defaults
	limit := parseLimit(limitStr, 20)
	offset := parseOffset(offsetStr, 0)

	// Parse date range
	startDate, endDate, err := parseDateRangeFromQuery(dateRange)
	if err != nil {
		log.Printf("[ERROR] handleSearchMedia - dateRange: %v", err)
		http.Error(w, "Invalid date range format", http.StatusBadRequest)
		return
	}

	log.Printf("[DEBUG] ===== SEARCH REQUEST =====")
	log.Printf("[DEBUG] query='%s' scope='%s' limit=%d offset=%d", query, scope, limit, offset)
	log.Printf("[DEBUG] userID='%s'", userID)

	// Handle geocoding for place/location/all scopes - geocode place names to coordinates/bounding box
	shouldGeocode := false
	if query != "" {
		// Skip geocoding if query looks like coordinates (contains comma with numbers)
		if strings.Contains(query, ",") {
			parts := strings.SplitN(query, ",", 2)
			if len(parts) == 2 {
				if _, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err1 == nil {
					if _, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err2 == nil {
						// It's actual coordinates like "lat,lon" - don't geocode, use as-is
						log.Printf("[DEBUG] handleSearchMedia: query '%s' looks like coordinates, using directly", query)
						shouldGeocode = false
					}
				}
			}
		}
		
		if !shouldGeocode && (scope == "place" || scope == "location") {
			shouldGeocode = true
		}
	}

	if shouldGeocode {
		log.Printf("[DEBUG] handleSearchMedia: geocoding '%s' (scope=%s)", query, scope)
		geocodeResult, err := ForwardGeocode(query)
		if err != nil {
			log.Printf("[ERROR] handleSearchMedia - geocode FAILED: %v", err)
			log.Printf("[ERROR] handleSearchMedia: falling back to text search with query='%s' scope=%s", query, scope)
			// Fall through to text search if geocoding fails
		} else {
			log.Printf("[DEBUG] Geocode SUCCESS: Lat='%s' Lon='%s'", geocodeResult.Lat, geocodeResult.Lon)
			log.Printf("[DEBUG] Geocode BoundingBox (raw)=%v (len=%d)", geocodeResult.BoundingBox, len(geocodeResult.BoundingBox))
			
			scope = "location"
			if len(geocodeResult.BoundingBox) >= 4 {
				// BoundingBox is []string - parse each element to float64
				south, err1 := strconv.ParseFloat(geocodeResult.BoundingBox[0], 64)
				north, err2 := strconv.ParseFloat(geocodeResult.BoundingBox[1], 64)
				west, err3 := strconv.ParseFloat(geocodeResult.BoundingBox[2], 64)
				east, err4 := strconv.ParseFloat(geocodeResult.BoundingBox[3], 64)
				
				log.Printf("[DEBUG] BoundingBox parsed: south=%.6f north=%.6f west=%.6f east=%.6f", south, north, west, east)
				log.Printf("[DEBUG] parse errors: err1=%v err2=%v err3=%v err4=%v", err1, err2, err3, err4)
				
				if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
					query = fmt.Sprintf("bounding_box:%.6f,%.6f,%.6f,%.6f", south, north, west, east)
					log.Printf("[DEBUG] Using bounding box query: %s", query)
				} else {
					log.Printf("[ERROR] handleSearchMedia: failed to parse bounding box coordinates")
					centerLat, _ := strconv.ParseFloat(geocodeResult.Lat, 64)
					centerLon, _ := strconv.ParseFloat(geocodeResult.Lon, 64)
					query = fmt.Sprintf("%.6f,%.6f", centerLat, centerLon)
					log.Printf("[DEBUG] Fallback to center coordinates: %s", query)
				}
			} else {
				centerLat, _ := strconv.ParseFloat(geocodeResult.Lat, 64)
				centerLon, _ := strconv.ParseFloat(geocodeResult.Lon, 64)
				query = fmt.Sprintf("%.6f,%.6f", centerLat, centerLon)
				log.Printf("[DEBUG] No bounding box (len=%d), using center coordinates: %s", len(geocodeResult.BoundingBox), query)
			}
		}
	} else {
		log.Printf("[DEBUG] handleSearchMedia: scope=%s query=%s - no geocoding needed", scope, query)
	}

	log.Printf("[DEBUG] ===== CALLING mediaRepo.Search =====")
	log.Printf("[DEBUG]   query='%s' scope='%s' limit=%d offset=%d", query, scope, limit, offset)
	log.Printf("[DEBUG]   userID='%s' startDate='%s' endDate='%s'", userID, startDate, endDate)

	matchingMedia, total, err := s.mediaRepo.Search(ctx, query, scope, limit, offset, &userID, startDate, endDate)
	if err != nil {
		log.Printf("[ERROR] handleSearchMedia - search DB error: %v", err)
		http.Error(w, "Failed to search media: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[DEBUG] ===== SEARCH RESULT =====")
	log.Printf("[DEBUG] total=%d results_count=%d", total, len(matchingMedia))
	if len(matchingMedia) > 0 {
		for i, m := range matchingMedia {
			log.Printf("[DEBUG]   [%d] id='%s' filename='%s' lat='%s' lon='%s'", i, m.ID, m.Filename, m.Metadata["gps_latitude"], m.Metadata["gps_longitude"])
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": matchingMedia,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// parseLimit parses limit parameter with default fallback
func parseLimit(limitStr string, defaultLimit int) int {
	if limitStr == "" {
		return defaultLimit
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		return defaultLimit
	}
	return limit
}

// parseOffset parses offset parameter with default fallback
func parseOffset(offsetStr string, defaultOffset int) int {
	if offsetStr == "" {
		return defaultOffset
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		return defaultOffset
	}
	return offset
}

// parseDateRangeFromQuery parses date range from query parameter
func parseDateRangeFromQuery(dateRange string) (string, string, error) {
	if dateRange == "" {
		return "", "", nil
	}
	start, end, err := parseDateRange(dateRange)
	if err != nil {
		return "", "", err
	}
	return start.Format("2006/01/02"), end.Format("2006/01/02"), nil
}
