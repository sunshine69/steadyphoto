package processor

import (
	"testing"
	"time"
)

func TestExtractDateFromString(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        time.Time
		wantNil     bool
		wantRemain  string
	}{
		// --- Contiguous YYYYMMDD ---
		{
			name: "YYYYMMDD basic",
			input: "DSC_20230514_103000.JPG",
			want: time.Date(2023, 5, 14, 0, 0, 0, 0, time.UTC),
			wantRemain: "DSC__103000.JPG",
		},
		{
			name: "YYYYMMDD plain",
			input: "20230514.jpg",
			want: time.Date(2023, 5, 14, 0, 0, 0, 0, time.UTC),
			wantRemain: ".jpg",
		},
		{
			name: "YYYYMMDD with prefix",
			input: "IMG_20230514_103000.mp4",
			want: time.Date(2023, 5, 14, 0, 0, 0, 0, time.UTC),
			wantRemain: "IMG__103000.mp4",
		},
		{
			name: "YYYYMMDD DDMMYYYY ambiguity fallback",
			input: "14052023.jpg",
			want: time.Date(2023, 5, 14, 0, 0, 0, 0, time.UTC),
			wantRemain: ".jpg",
		},
		{
			name: "YYYYMMDD invalid month falls back to DDMMYYYY",
			input: "20230014.jpg",
			wantNil: true,
		},
		{
			name: "YYYYMMDD invalid day falls back to DDMMYYYY",
			input: "20230532.jpg",
			wantNil: true,
		},

		// --- Separator: YYYY-MM-DD ---
		{
			name: "YYYY-MM-DD dash",
			input: "2023-05-14.png",
			want: time.Date(2023, 5, 14, 0, 0, 0, 0, time.UTC),
			wantRemain: ".png",
		},
		{
			name: "YYYY/MM/DD slash",
			input: "WhatsApp Image 2023/05/14 at 14.22.33.jpg",
			want: time.Date(2023, 5, 14, 0, 0, 0, 0, time.UTC),
			wantRemain: "WhatsApp Image  at 14.22.33.jpg",
		},
		{
			name: "YYYY_MM_DD underscore",
			input: "file_2023_05_14_backup.txt",
			want: time.Date(2023, 5, 14, 0, 0, 0, 0, time.UTC),
			wantRemain: "file__ backup.txt",
		},
		{
			name: "YYYY.MM.DD dot",
			input: "report.2023.05.14.final.pdf",
			want: time.Date(2023, 5, 14, 0, 0, 0, 0, time.UTC),
			wantRemain: "report. final.pdf",
		},

		// --- Separator: DD-MM-YYYY ---
		{
			name: "DD-MM-YYYY",
			input: "14-05-2023.jpg",
			want: time.Date(2023, 5, 14, 0, 0, 0, 0, time.UTC),
			wantRemain: ".jpg",
		},
		{
			name: "DD/MM/YYYY",
			input: "14/05/2023 15.45.30.jpg",
			want: time.Date(2023, 5, 14, 0, 0, 0, 0, time.UTC),
			wantRemain: " 15.45.30.jpg",
		},

		// --- Separator: YYYY-DD-MM ---
		{
			name: "YYYY-DD-MM (ambig, second layout matches)",
			input: "2023-14-05.txt",
			want: time.Date(2023, 5, 14, 0, 0, 0, 0, time.UTC),
			wantRemain: ".txt",
		},

		// --- Real-world filenames ---
		{
			name: "FontHouse format",
			input: "FontHouse-20250224.jpeg",
			want: time.Date(2025, 2, 24, 0, 0, 0, 0, time.UTC),
			wantRemain: "FontHouse-.jpeg",
		},
		{
			name: "Screenshot format",
			input: "Screenshot_20230514_103000.png",
			want: time.Date(2023, 5, 14, 0, 0, 0, 0, time.UTC),
			wantRemain: "Screenshot__103000.png",
		},
		{
			name: "DJI drone",
			input: "DJI_0001_20230514_103000.DNG",
			want: time.Date(2023, 5, 14, 0, 0, 0, 0, time.UTC),
			wantRemain: "DJI_0001__103000.DNG",
		},
		{
			name: "Poco PXL",
			input: "PXL_20230514_103000.jpg",
			want: time.Date(2023, 5, 14, 0, 0, 0, 0, time.UTC),
			wantRemain: "PXL__103000.jpg",
		},

		// --- No date ---
		{
			name: "No date",
			input: "IMG_0001.jpg",
			wantNil: true,
		},
		{
			name: "Random text",
			input: "random-notes.txt",
			wantNil: true,
		},
		{
			name: "Empty",
			input: "",
			wantNil: true,
		},

		// --- Edge cases ---
		{
			name: "Invalid month 13 contiguous",
			input: "DSC_20231314_103000.jpg",
			wantNil: true,
		},
		{
			name: "Invalid day 00",
			input: "DSC_20230500_103000.jpg",
			wantNil: true,
		},
		{
			name: "Year too small",
			input: "DSC_19000101_103000.jpg",
			wantNil: true,
		},
		{
			name: "Year too large",
			input: "DSC_99991231_235959.jpg",
			wantNil: true,
		},
		{
			name: "Only extension",
			input: ".jpg",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, remaining, err := ExtractDateFromString(tt.input)
			if tt.wantNil {
				if err == nil {
					t.Errorf("ExtractDateFromString(%q) = %v, expected error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("ExtractDateFromString(%q) error: %v", tt.input, err)
				return
			}
			if got.IsZero() {
				t.Errorf("ExtractDateFromString(%q) = zero time, expected %v", tt.input, tt.want)
				return
			}
			if !got.Equal(tt.want) {
				t.Errorf("ExtractDateFromString(%q) = %v, expected %v", tt.input, got, tt.want)
			}
			if remaining != tt.wantRemain {
				t.Errorf("ExtractDateFromString(%q) remaining = %q, expected %q", tt.input, remaining, tt.wantRemain)
			}
		})
	}
}
