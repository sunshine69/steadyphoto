package processor

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestDebugFontHouse(t *testing.T) {
	filename := "FontHouse-20250224.jpeg"
	
	// Simulate ParseDateFromFilename logic
	base := filename
	if dotIdx := strings.LastIndex(base, "."); dotIdx > 0 {
		base = base[:dotIdx]
	}
	fmt.Printf("base after ext removal: %q\n", base)

	// Pattern 2: YYYYMMDD (date only)
	re := regexp.MustCompile(`\b(\d{4})(\d{2})(\d{2})\b`)
	m := re.FindStringSubmatch(base)
	fmt.Printf("Pattern 2 match: %v\n", m)
	
	// Check word boundaries manually
	for i, c := range base {
		fmt.Printf("pos %d: '%c' \n", i, c)
	}
}
