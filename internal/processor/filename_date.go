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

// isValidDate checks if a parsed date falls within a reasonable range for media captures.
// Rejects: dates before 1980, dates in the future, or dates outside valid month/day ranges.
func isValidDate(t time.Time) bool {
	year := t.Year()

	// Reject anything before 1980 — no realistic media predates this era
	if year < 1980 {
		return false
	}

	// Reject any date that is in the future (clock skew protection).
	nowUTC := time.Now().UTC()
	nowYMD := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)
	tYMD := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	if tYMD.After(nowYMD) {
		return false
	}

	return true
}

// ExtractDateFromString extracts a date from input.
// Returns parsed time, remaining string (with date removed), and error.
// Only tries very obvious date patterns. If the date parses but is outside valid range,
// it returns an error instead of a zero time (so the caller falls back to the next method).
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
				// Validate the date is in a reasonable range.
				// If invalid, return error so caller can fall back.
				if !isValidDate(t) {
					return time.Time{}, input, fmt.Errorf("date %s (from %s in %q) is outside valid range (1980 to now)", canonical, matched, input)
				}
				remaining := input[:loc[0]] + input[loc[1]:]
				return t, remaining, nil
			}
		}
	}
	return time.Time{}, input, fmt.Errorf("no date found in: %s", input)
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

// datePatterns: only VERY obvious date patterns, in order of strictness.
var datePatterns = []datePattern{
	{
		name: "YYYY sep XX sep YY (YYYY-MM-DD)",
		re: regexp.MustCompile(`(\d{4})([-_])(\d{2})([-_])(\d{2})`),
		layouts: []string{
			"2006-01-02",
			"2006_01_02",
		},
		canonSepIdxs: []int{2, 4},
	},
	{
		name: "YYYY/MM/DD",
		re: regexp.MustCompile(`(\d{4})/(\d{2})/(\d{2})`),
		layouts: []string{
			"2006/01/02",
		},
		canonSepIdxs: []int{2, 4},
	},
	{
		name: "YYYY.MM.DD",
		re: regexp.MustCompile(`(\d{4})\.(\d{2})\.(\d{2})`),
		layouts: []string{
			"2006.01.02",
		},
		canonSepIdxs: []int{2, 4},
	},
}
