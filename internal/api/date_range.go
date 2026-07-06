package api

import (
	"fmt"
	"strings"
	"time"
)

// parseDateRange parses a date range string in various formats and returns parsed start/end times.
// Supported formats:
//   - "dd/mm/yyyy" (e.g., "01/06/2024")
//   - "yyyy/mm/dd" (e.g., "2024/06/01")
//   - "dd/mm/yyyy hh:mm:ss" (e.g., "01/06/2024 12:30:45")
//   - "yyyy/mm/dd hh:mm:ss" (e.g., "2024/06/01 12:30:45")
//   - "dd-mm-yyyy" (e.g., "01-06-2024")
//   - "yyyy-mm-dd" (e.g., "2024-06-01")
//   - "dd.mm.yyyy" (e.g., "01.06.2024")
//   - "yyyy.mm.dd" (e.g., "2024.06.01")
//   - "yyyy" (e.g., "2024" - entire year)
//   - "yyyy/mm" (e.g., "2024/06" - entire month)
//   - "dd/mm/yyyy - dd/mm/yyyy" (range with "-" separator)
//   - "yyyy/mm/dd - yyyy/mm/dd" (range with "-" separator)
//   - "yyyy-mm-dd - yyyy-mm-dd" (range with "-" separator)
//
// If the string contains a "-" separator (but not in date components), it's treated as a range.
// Returns the start and end times. If only one date is given, start and end will be the same
// (for single date search, we'll search that exact day).
// If startDate > endDate, they are swapped.
func parseDateRange(dateRange string) (start time.Time, end time.Time, err error) {
	if dateRange == "" {
		return time.Time{}, time.Time{}, nil
	}

	// Check if this is a range with "-" separator (but not in date components)
	// We need to distinguish between date separators (-) and range separator (-)
	// A range will have the format: "date1 - date2" or "date1- date2" or "date1 -date2"
	
	// Try to detect if this is a range by looking for a standalone "-"
	rangeParts := strings.Split(dateRange, " - ")
	if len(rangeParts) != 2 {
		// Try with just "-" (but not if it's part of a date like "2024-06-01")
		if strings.Contains(dateRange, " - ") || strings.Contains(dateRange, " -") || strings.Contains(dateRange, "- ") {
			// It's likely a range, but the split didn't work perfectly
			// Try to split by the first occurrence of " - " or "- " or " -"
			if idx := strings.Index(dateRange, " - "); idx != -1 {
				rangeParts = []string{strings.TrimSpace(dateRange[:idx]), strings.TrimSpace(dateRange[idx+3:])}
			} else if idx := strings.Index(dateRange, "- "); idx != -1 && idx > 0 {
				// Check if this is a date separator or range separator
				// If there's more than one "-", it might be a range
				if strings.Count(dateRange, "-") > 1 {
					rangeParts = []string{strings.TrimSpace(dateRange[:idx]), strings.TrimSpace(dateRange[idx+2:])}
				}
			} else if idx := strings.Index(dateRange, " -"); idx != -1 && idx > 0 {
				if strings.Count(dateRange, "-") > 1 {
					rangeParts = []string{strings.TrimSpace(dateRange[:idx]), strings.TrimSpace(dateRange[idx+2:])}
				}
			}
		}
	}

	if len(rangeParts) == 2 {
		// Parse as range
		start, err = parseSingleDate(rangeParts[0])
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end, err = parseSingleDate(rangeParts[1])
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	} else {
		// Parse as single date
		singleDate, err := parseSingleDate(dateRange)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		start = singleDate
		end = singleDate
	}

	// If start > end, swap them
	if start.After(end) {
		start, end = end, start
	}

	return start, end, nil
}

// parseSingleDate parses a single date string in various formats.
func parseSingleDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	
	if dateStr == "" {
		return time.Time{}, nil
	}

	// Try different formats
	formats := []string{
		"02/01/2006",      // dd/mm/yyyy (Go: day/month/year)
		"2006/01/02",      // yyyy/mm/dd
		"02/01/2006 15:04:05", // dd/mm/yyyy hh:mm:ss
		"2006/01/02 15:04:05", // yyyy/mm/dd hh:mm:ss
		"02-01-2006",      // dd-mm-yyyy
		"2006-01-02",      // yyyy-mm-dd (ISO format)
		"02.01.2006",      // dd.mm.yyyy
		"2006.01.02",      // yyyy.mm.dd
		"02/01/2006 15:04",    // dd/mm/yyyy hh:mm
		"2006/01/02 15:04",    // yyyy/mm/dd hh:mm
		"2006",            // yyyy (entire year)
		"2006/01",         // yyyy/mm (entire month)
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			// For year-only format, set to Jan 1 of that year
			if format == "2006" {
				t = time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
			}
			// For year/month format, set to first day of that month
			if format == "2006/01" {
				t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			}
			return t, nil
		}
	}

	// Try to handle formats with different separators
	// Replace "." with "/"
	if strings.Contains(dateStr, ".") {
		altStr := strings.ReplaceAll(dateStr, ".", "/")
		if t, err := time.Parse("01/02/2006", altStr); err == nil {
			return t, nil
		}
		if t, err := time.Parse("2006/01/02", altStr); err == nil {
			return t, nil
		}
	}

	// Try to handle formats with different separators
	// Replace "-" with "/"
	if strings.Contains(dateStr, "-") {
		altStr := strings.ReplaceAll(dateStr, "-", "/")
		if t, err := time.Parse("01/02/2006", altStr); err == nil {
			return t, nil
		}
		if t, err := time.Parse("2006/01/02", altStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}