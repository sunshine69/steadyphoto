package processor

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ParseDateFromFilename is a legacy wrapper for ExtractDateFromString.
// Returns the parsed time only, discarding the remaining string and error.
func ParseDateFromFilename(filename string) time.Time {
	if t, _, err := ExtractDateFromString(filename); err == nil && !t.IsZero() {
		return t
	}
	return time.Time{}
}

// ExtractDateFromString extracts a date from input.
// Returns parsed time, remaining string (with date removed), and error.
// Tries each date pattern in order. For each match, extracts capture groups,
// builds a canonical date string, then tries Go time.Parse with each layout.
// If a date parses successfully but the year is outside the valid range,
// it continues trying other patterns instead of returning the invalid date.
func ExtractDateFromString(input string) (time.Time, string, error) {
	for _, dp := range datePatterns {
		loc := dp.re.FindStringIndex(input)
		if loc == nil {
			continue
		}
		matched := input[loc[0]:loc[1]]
		groups := dp.re.FindStringSubmatch(matched)
		if groups == nil {
			continue
		}
		// Build canonical date string from groups
		canonical := buildCanonical(groups, dp.canonSepIdxs)
		for _, layout := range dp.layouts {
			t, err := time.Parse(layout, canonical)
			if err == nil {
				// Validate the year is in a reasonable range to avoid false positives
				// like treating a long numeric ID (e.g., 936410131344710) as YYYYMMDD.
				if !isValidYear(t.Year()) {
					continue // Try the next layout or pattern
				}
				remaining := input[:loc[0]] + input[loc[1]:]
				return t, remaining, nil
			}
		}
	}
	return time.Time{}, input, fmt.Errorf("no date found in: %s", input)
}

// isValidYear checks if a year falls within a reasonable range for media captures.
// This prevents false positives from numeric IDs or timestamps being misinterpreted
// as dates (e.g., year 9364 from "received_936410131344710.jpeg").
func isValidYear(year int) bool {
	return year >= 1970 && year <= 2100
}

// buildCanonical builds a canonical date string from regex capture groups.
// sepIdxs: 1-based group indices that are separators (e.g. [2,4] means groups[2] and groups[4] are separators).
// If sepIdxs is empty, groups are concatenated directly.
func buildCanonical(groups []string, sepIdxs []int) string {
	var sb strings.Builder
	for i := 1; i < len(groups); i++ {
		// In the current patterns, all groups are date parts (year/month/day), not separators.
		// The separator groups are handled by being concatenated into the canonical string
		// as part of the date string (e.g., "2006-01-02" expects hyphens at those positions).
		sb.WriteString(groups[i])
	}
	return sb.String()
}

// datePattern holds a regex, layout strings, and separator group indices.
type datePattern struct {
	name         string
	re           *regexp.Regexp
	layouts      []string
	canonSepIdxs []int // 1-based group indices that are separators
}

// datePatterns: try each in order.
var datePatterns = []datePattern{
	{
		name: "YYYY sep XX sep YY (YYYY-MM-DD or YYYY-DD-MM)",
		re: regexp.MustCompile(`(\d{4})([-_./\s:;.])(\d{1,2})([-_./\s:;.])(\d{1,2})`),
		layouts: []string{
			"2006-01-02",
			"2006-02-01",
			"2006/01/02",
			"2006/02/01",
			"2006.01.02",
			"2006.02.01",
		},
		canonSepIdxs: []int{2, 4},
	},
	{
		name: "XX sep YY sep YYYY (DD-MM-YYYY or MM-DD-YYYY)",
		re: regexp.MustCompile(`(\d{1,2})([-_./\s:;.])(\d{1,2})([-_./\s:;.])(\d{4})`),
		layouts: []string{
			"02-01-2006",
			"01-02-2006",
			"02/01/2006",
			"01/02/2006",
			"02.01.2006",
			"01.02.2006",
		},
		canonSepIdxs: []int{2, 4},
	},
	{
		name: "YYYYMMDD",
		re:   regexp.MustCompile(`(\d{4})(\d{2})(\d{2})`),
		layouts: []string{"20060102"},
	},
	{
		name: "DDMMYYYY",
		re:   regexp.MustCompile(`(\d{2})(\d{2})(\d{4})`),
		layouts: []string{"02012006"},
	},
}
