package processor

import (
	"fmt"
	"regexp"
	"strconv"
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
// Tries each date pattern. For each match, extracts capture groups,
// builds a canonical date string, then tries Go time.Parse with each layout.
func ExtractDateFromString(input string) (time.Time, string, error) {
	// First try: look for long digit blocks (16+ digits = microseconds Unix time)
	if t, remaining, err := tryUnixTimestamp(input); err == nil {
		return t, remaining, nil
	}

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
		isSep := false
		for _, idx := range sepIdxs {
			if idx == i {
				isSep = true
				break
			}
		}
		if isSep {
			sb.WriteString(groups[i])
		} else {
			sb.WriteString(groups[i])
		}
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
		name: "YYYYMMDD",
		re:   regexp.MustCompile(`(\d{4})(\d{2})(\d{2})`),
		layouts: []string{"20060102"},
	},
	{
		name: "DDMMYYYY",
		re:   regexp.MustCompile(`(\d{2})(\d{2})(\d{4})`),
		layouts: []string{"02012006"},
	},
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
}

// tryUnixTimestamp looks for a 16+ digit block and converts it via ParseFlexibleUnixString.
func tryUnixTimestamp(input string) (time.Time, string, error) {
	re := regexp.MustCompile(`\d+`)
	digitStr := re.FindString(input)
	if len(digitStr) < 16 {
		return time.Time{}, input, fmt.Errorf("no 16+ digit block found")
	}
	t, err := ParseFlexibleUnixString(input)
	if err != nil {
		return time.Time{}, input, err
	}
	// Remove the digit block from input
	remaining := strings.Replace(input, digitStr, "", 1)
	return t, remaining, nil
}

// ParseFlexibleUnixString takes a filename/string, extracts the digits,
// determines the precision based on digit length, and returns a time.Time object.
func ParseFlexibleUnixString(input string) (time.Time, error) {
	re := regexp.MustCompile(`\d+`)
	digitStr := re.FindString(input)
	if digitStr == "" {
		return time.Time{}, fmt.Errorf("no digits found in input string")
	}
	length := len(digitStr)
	if length > 19 {
		digitStr = digitStr[:19]
		length = 19
	}
	val, err := strconv.ParseInt(digitStr, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse digits to integer: %w", err)
	}
	switch {
	case length >= 19: // Nanoseconds
		return time.Unix(0, val), nil
	case length >= 16: // Microseconds
		seconds := val / 1_000_000
		microsRemainder := val % 1_000_000
		return time.Unix(seconds, microsRemainder*1_000), nil
	case length >= 13: // Milliseconds
		seconds := val / 1_000
		millisRemainder := val % 1_000
		return time.Unix(seconds, millisRemainder*1_000_000), nil
	case length >= 10: // Seconds
		if length > 10 {
			digitStr = digitStr[:10]
			val, _ = strconv.ParseInt(digitStr, 10, 64)
		}
		return time.Unix(val, 0), nil
	default:
		return time.Time{}, fmt.Errorf("digit string length (%d) is below the minimum 10-digit threshold for Unix seconds", length)
	}
}
