package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"steadyphoto/internal/processor"
)

func main() {
	filename := "FontHouse-20250224.jpeg"
	base := filename
	if dotIdx := strings.LastIndex(base, "."); dotIdx > 0 {
		base = base[:dotIdx]
	}

	// Pattern 2: YYYYMMDD (date only)
	re := regexp.MustCompile(`\b(\d{4})(\d{2})(\d{2})\b`)
	m := re.FindStringSubmatch(base)
	fmt.Printf("Base: %q, Match: %v\n", base, m)

	fnDate := processor.ParseDateFromFilename(filename)
	fmt.Printf("ParseDateFromFilename(%q) = %v, IsZero: %v\n", filename, fnDate, fnDate.IsZero())

	// Also check EXIF reader
	if len(os.Args) > 1 {
		f, err := os.Open(os.Args[1])
		if err != nil {
			fmt.Printf("Open error: %v\n", err)
			return
		}
		defer f.Close()
		reader := processor.NewExifReader()
		info, err := reader.ReadExif(f)
		if err != nil {
			fmt.Printf("ReadExif error: %v\n", err)
			return
		}
		if info == nil {
			fmt.Println("Exif: nil info")
			return
		}
		fmt.Printf("Exif: CapturedAt=%v, Orientation=%d\n", info.CapturedAt, info.Orientation)
		fmt.Printf("Exif tags: %d\n", len(info.Tags))
		for _, tag := range info.Tags {
			fmt.Printf("  tag: %s = %s\n", tag.Tag, tag.Value)
		}
	}
}
