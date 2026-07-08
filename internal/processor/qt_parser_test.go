package processor

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"os"
	"testing"
)

// buildAtom builds a QuickTime atom with the given type and data, returns *QuickTimeAtom.
func buildAtom(atomType string, data []byte) (*QuickTimeAtom, error) {
	var raw bytes.Buffer
	raw.Write([]byte{0, 0, 0, 0}) // Size placeholder
	raw.WriteString(atomType)
	raw.Write(data)
	size := raw.Len()
	binary.BigEndian.PutUint32(raw.Bytes()[:4], uint32(size))

	return &QuickTimeAtom{
		Size:       int64(size),
		Type:       atomType,
		HeaderSize: 8,
		Data:       data,
	}, nil
}

// buildAtomBytes builds a QuickTime atom and returns raw bytes.
func buildAtomBytes(atomType string, data []byte) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0, 0, 0, 0}) // Size placeholder
	buf.WriteString(atomType)
	buf.Write(data)
	size := buf.Len()
	binary.BigEndian.PutUint32(buf.Bytes()[:4], uint32(size))
	return buf.Bytes()
}

// buildMvhdV0 builds a version-0 movie header atom data.
func buildMvhdV0() []byte {
	data := make([]byte, 100)
	data[0] = 0 // Version 0
	binary.BigEndian.PutUint32(data[4:8], 631152000)
	binary.BigEndian.PutUint32(data[8:12], 631152000)
	binary.BigEndian.PutUint32(data[12:16], 1000)
	binary.BigEndian.PutUint32(data[16:20], 5000)
	return data
}

// buildTkhdV0 builds a version-0 track header atom data.
func buildTkhdV0(width int32, height int32) []byte {
	data := make([]byte, 84)
	data[0] = 0 // Version 0
	binary.BigEndian.PutUint32(data[4:8], 631152000)
	binary.BigEndian.PutUint32(data[8:12], 631152000)
	binary.BigEndian.PutUint32(data[12:16], 1)
	binary.BigEndian.PutUint32(data[16:20], 5000)
	binary.BigEndian.PutUint16(data[28:30], 0)
	binary.BigEndian.PutUint16(data[30:32], 0)
	binary.BigEndian.PutUint16(data[32:34], 256)
	// Matrix (36 bytes) identity
	for i := 36; i < 72; i++ {
		if (i-36)%4 == 0 {
			data[i] = 0x00
			data[i+1] = 0x01
			data[i+2] = 0x00
			data[i+3] = 0x00
		}
	}
	binary.BigEndian.PutUint32(data[68:72], uint32(width)<<16)
	binary.BigEndian.PutUint32(data[72:76], uint32(height)<<16)
	return data
}

// buildMdhdV0 builds a version-0 media header atom data.
func buildMdhdV0(timeScale uint32, duration uint32, lang byte) []byte {
	data := make([]byte, 28)
	data[0] = 0
	binary.BigEndian.PutUint32(data[4:8], 631152000)
	binary.BigEndian.PutUint32(data[8:12], 631152000)
	binary.BigEndian.PutUint32(data[12:16], timeScale)
	binary.BigEndian.PutUint32(data[16:20], duration)
	data[24] = lang
	data[25] = lang
	data[26] = lang
	return data
}

// buildHdlr builds a handler atom data.
func buildHdlr(handlerType string) []byte {
	data := make([]byte, 32)
	data[0] = 0
	copy(data[8:12], handlerType)
	data[27] = 0
	return data
}

func TestParseQuickTimeFile_SimpleMP4(t *testing.T) {
	ftypData := make([]byte, 12)
	copy(ftypData[4:8], "isom")
	binary.BigEndian.PutUint32(ftypData[8:12], 512)

	moovData := buildMvhdV0()

	var mp4 bytes.Buffer
	mp4.Write(buildAtomBytes(AtomFTyp, ftypData))
	mp4.Write(buildAtomBytes(AtomMoov, moovData))

	qtf, err := ParseQuickTimeFromReader(&mp4)
	if err != nil {
		t.Fatalf("ParseQuickTimeFromReader failed: %v", err)
	}

	if qtf.FTyp == nil {
		t.Fatal("expected ftyp box")
	}
	if qtf.FTyp.MajorBrand != "isom" {
		t.Errorf("expected major brand 'isom', got %q", qtf.FTyp.MajorBrand)
	}

	if qtf.Moov == nil {
		t.Fatal("expected moov box")
	}
	if qtf.Moov.Mvhd == nil {
		t.Fatal("expected mvhd atom")
	}
	if qtf.Moov.Mvhd.TimeScale != 1000 {
		t.Errorf("expected time scale 1000, got %d", qtf.Moov.Mvhd.TimeScale)
	}
	if qtf.Moov.Mvhd.Duration != 5000 {
		t.Errorf("expected duration 5000, got %d", qtf.Moov.Mvhd.Duration)
	}
}

func TestParseQuickTimeFile_TrackWithMedia(t *testing.T) {
	ftypData := make([]byte, 12)
	copy(ftypData[4:8], "isom")

	// Build track
	tkhdData := buildTkhdV0(1920, 1080)
	tkhdAtom := buildAtomBytes(AtomTkhd, tkhdData)

	mdhdData := buildMdhdV0(30000, 150000, 'e')
	mdhdAtom := buildAtomBytes(AtomMdhd, mdhdData)

	hdlrData := buildHdlr("vide")
	hdlrAtom := buildAtomBytes(AtomHdlr, hdlrData)

	stblAtom := buildAtomBytes(AtomStbl, nil)
	minfAtom := buildAtomBytes(AtomMinf, stblAtom)

	mdiaBuf := bytes.NewBuffer(nil)
	mdiaBuf.Write(mdhdAtom)
	mdiaBuf.Write(hdlrAtom)
	mdiaBuf.Write(minfAtom)
	mdiaAtom := buildAtomBytes(AtomMdia, mdiaBuf.Bytes())

	trakBuf := bytes.NewBuffer(nil)
	trakBuf.Write(tkhdAtom)
	trakBuf.Write(mdiaAtom)
	trakAtom := buildAtomBytes(AtomTrak, trakBuf.Bytes())

	moovBuf := bytes.NewBuffer(nil)
	moovBuf.Write(buildAtomBytes(AtomMvhd, buildMvhdV0()))
	moovBuf.Write(trakAtom)

	var mp4 bytes.Buffer
	mp4.Write(buildAtomBytes(AtomFTyp, ftypData))
	mp4.Write(buildAtomBytes(AtomMoov, moovBuf.Bytes()))

	qtf, err := ParseQuickTimeFromReader(&mp4)
	if err != nil {
		t.Fatalf("ParseQuickTimeFromReader failed: %v", err)
	}

	if qtf.Moov == nil {
		t.Fatal("expected moov box")
	}
	if len(qtf.Moov.Traks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(qtf.Moov.Traks))
	}

	trk := qtf.Moov.Traks[0]
	if trk.Tkhd == nil {
		t.Fatal("expected tkhd in track")
	}

	width := float64(trk.Tkhd.Width) / 65536.0
	height := float64(trk.Tkhd.Height) / 65536.0
	if width != 1920 {
		t.Errorf("expected width 1920, got %f", width)
	}
	if height != 1080 {
		t.Errorf("expected height 1080, got %f", height)
	}
}

func TestParseQuickTimeFile_64BitAtom(t *testing.T) {
	data := make([]byte, 1000)
	copy(data, "hello")

	// Build 64-bit atom manually
	var buf bytes.Buffer
	buf.Write([]byte{0, 0, 0, 1}) // size=1 (extended)
	buf.WriteString(AtomFree)
	binary.Write(&buf, binary.BigEndian, int64(8+8+1000)) // extended size
	buf.Write(data)

	atoms, err := parseAtoms(&buf, 0)
	if err != nil {
		t.Fatalf("parseAtoms failed: %v", err)
	}

	if len(atoms) != 1 {
		t.Fatalf("expected 1 atom, got %d", len(atoms))
	}

	if atoms[0].Type != AtomFree {
		t.Errorf("expected type 'free', got %q", atoms[0].Type)
	}
	if atoms[0].Size != 8+8+1000 {
		t.Errorf("expected size %d, got %d", 8+8+1000, atoms[0].Size)
	}
}

func TestParseQuickTimeFile_EmptyMoov(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(buildAtomBytes(AtomFTyp, make([]byte, 12)))
	buf.Write(buildAtomBytes(AtomMdat, nil))

	_, err := ParseQuickTimeFromReader(&buf)
	if err == nil {
		t.Fatal("expected error for missing moov")
	}
}

func TestParseQuickTimeFile_DeepNesting(t *testing.T) {
	// Build atoms nested 25 levels deep
	var nested bytes.Buffer
	for i := 0; i < 25; i++ {
		inner := buildAtomBytes(AtomMoov, nil)
		nested = *bytes.NewBuffer(inner)
	}

	_, err := parseAtoms(&nested, 0)
	if err == nil {
		t.Fatal("expected error for deep nesting")
	}
}

func TestParseMvhdVersion0(t *testing.T) {
	data := buildMvhdV0()
	atom := &QuickTimeAtom{Type: AtomMvhd, Data: data, HeaderSize: 8, Size: int64(8 + len(data))}

	mvhd := parseMovieHeaderAtom(atom)
	if mvhd == nil {
		t.Fatal("expected mvhd")
	}
	if mvhd.Version != 0 {
		t.Errorf("expected version 0, got %d", mvhd.Version)
	}
	if mvhd.TimeScale != 1000 {
		t.Errorf("expected time scale 1000, got %d", mvhd.TimeScale)
	}
	if mvhd.Duration != 5000 {
		t.Errorf("expected duration 5000, got %d", mvhd.Duration)
	}
}

func TestParseMvhdVersion1(t *testing.T) {
	data := make([]byte, 32)
	data[0] = 1 // Version 1
	binary.BigEndian.PutUint64(data[4:12], 631152000)
	binary.BigEndian.PutUint64(data[12:20], 631152000)
	binary.BigEndian.PutUint32(data[20:24], 1000)
	binary.BigEndian.PutUint64(data[24:32], 10000)

	atom := &QuickTimeAtom{Type: AtomMvhd, Data: data, HeaderSize: 8, Size: int64(8 + len(data))}
	mvhd := parseMovieHeaderAtom(atom)

	if mvhd.Version != 1 {
		t.Errorf("expected version 1, got %d", mvhd.Version)
	}
	if mvhd.TimeScale != 1000 {
		t.Errorf("expected time scale 1000, got %d", mvhd.TimeScale)
	}
	if mvhd.Duration != 10000 {
		t.Errorf("expected duration 10000, got %d", mvhd.Duration)
	}
}

func TestParseTkhdVersion0(t *testing.T) {
	data := buildTkhdV0(1920, 1080)
	atom := &QuickTimeAtom{Type: AtomTkhd, Data: data, HeaderSize: 8, Size: int64(8 + len(data))}

	tkhd := parseTrackHeaderAtom(atom)
	if tkhd == nil {
		t.Fatal("expected tkhd")
	}
	if !tkhd.Enabled {
		t.Error("expected track to be enabled")
	}

	width := float64(tkhd.Width) / 65536.0
	height := float64(tkhd.Height) / 65536.0
	if width != 1920 {
		t.Errorf("expected width 1920, got %f", width)
	}
	if height != 1080 {
		t.Errorf("expected height 1080, got %f", height)
	}
}

func TestParseMdhdVersion0(t *testing.T) {
	data := buildMdhdV0(30000, 150000, 'e')
	atom := &QuickTimeAtom{Type: AtomMdhd, Data: data, HeaderSize: 8, Size: int64(8 + len(data))}

	mdhd := parseMediaHeaderAtom(atom)
	if mdhd == nil {
		t.Fatal("expected mdhd")
	}
	if mdhd.TimeScale != 30000 {
		t.Errorf("expected time scale 30000, got %d", mdhd.TimeScale)
	}
	if mdhd.Duration != 150000 {
		t.Errorf("expected duration 150000, got %d", mdhd.Duration)
	}
}

func TestParseHdlr(t *testing.T) {
	data := buildHdlr("vide")
	atom := &QuickTimeAtom{Type: AtomHdlr, Data: data, HeaderSize: 8, Size: int64(8 + len(data))}

	hdlr := parseHandlerBox(atom)
	if hdlr == nil {
		t.Fatal("expected hdlr")
	}
	if hdlr.HandlerType != "vide" {
		t.Errorf("expected handler type 'vide', got %q", hdlr.HandlerType)
	}
}

func TestParseFTyp(t *testing.T) {
	data := make([]byte, 16)
	copy(data[4:8], "isom")
	binary.BigEndian.PutUint32(data[8:12], 512)
	copy(data[12:16], "isom")

	atom := &QuickTimeAtom{Type: AtomFTyp, Data: data, HeaderSize: 8, Size: int64(8 + len(data))}
	ftyp := parseFTypBox(atom)

	if ftyp.MajorBrand != "isom" {
		t.Errorf("expected major brand 'isom', got %q", ftyp.MajorBrand)
	}
	if ftyp.MinorVersion != 512 {
		t.Errorf("expected minor version 512, got %d", ftyp.MinorVersion)
	}
	if len(ftyp.CompatibleBrands) < 1 || ftyp.CompatibleBrands[0] != "isom" {
		t.Errorf("expected compatible brand 'isom', got %v", ftyp.CompatibleBrands)
	}
}

func TestParseFTyp_AppleBrand(t *testing.T) {
	data := make([]byte, 20)
	copy(data[4:8], "M4A ")
	binary.BigEndian.PutUint32(data[8:12], 0)
	copy(data[12:16], "M4A ")
	copy(data[16:20], "mp41")

	atom := &QuickTimeAtom{Type: AtomFTyp, Data: data, HeaderSize: 8, Size: int64(8 + len(data))}
	ftyp := parseFTypBox(atom)

	if ftyp.MajorBrand != "M4A " {
		t.Errorf("expected major brand 'M4A ', got %q", ftyp.MajorBrand)
	}
}

func TestExtractPhoneMetadata_MvhdCreationDate(t *testing.T) {
	data := buildMvhdV0()
	binary.BigEndian.PutUint32(data[4:8], 2714000000)

	moovBuf := bytes.NewBuffer(nil)
	moovBuf.Write(buildAtomBytes(AtomMvhd, data))

	var mp4 bytes.Buffer
	mp4.Write(buildAtomBytes(AtomFTyp, make([]byte, 12)))
	mp4.Write(buildAtomBytes(AtomMoov, moovBuf.Bytes()))

	qtf, err := ParseQuickTimeFromReader(&mp4)
	if err != nil {
		t.Fatalf("ParseQuickTimeFromReader failed: %v", err)
	}

	metadata := extractPhoneMetadataFromContainer(qtf)
	if metadata == nil {
		t.Fatal("expected non-nil metadata")
	}
	if metadata.CreationDate == "" {
		t.Error("expected creation date to be set")
	}
}

func TestParseGPSCoordinates_SanFrancisco(t *testing.T) {
	metadata := &PhoneMetadata{}
	parseGPSCoordinates("+37.7749-122.4194/+", metadata)

	if !metadata.HasGPS {
		t.Error("expected HasGPS to be true")
	}
	if metadata.GPSLatitude != 37.7749 {
		t.Errorf("expected latitude 37.7749, got %f", metadata.GPSLatitude)
	}
	if metadata.GPSLongitude != -122.4194 {
		t.Errorf("expected longitude -122.4194, got %f", metadata.GPSLongitude)
	}
}

func TestParseGPSCoordinates_WithAltitude(t *testing.T) {
	metadata := &PhoneMetadata{}
	parseGPSCoordinates("+37.7749-122.4194/+10.0", metadata)

	if !metadata.HasGPS {
		t.Error("expected HasGPS to be true")
	}
	if metadata.GPSLatitude != 37.7749 {
		t.Errorf("expected latitude 37.7749, got %f", metadata.GPSLatitude)
	}
	if metadata.GPSLongitude != -122.4194 {
		t.Errorf("expected longitude -122.4194, got %f", metadata.GPSLongitude)
	}
	if metadata.GPSAltitude != 10.0 {
		t.Errorf("expected altitude 10.0, got %f", metadata.GPSAltitude)
	}
}

func TestParseGPSCoordinates_NewYork(t *testing.T) {
	metadata := &PhoneMetadata{}
	parseGPSCoordinates("+40.7128-74.0060/", metadata)

	if !metadata.HasGPS {
		t.Error("expected HasGPS to be true")
	}
	if metadata.GPSLatitude != 40.7128 {
		t.Errorf("expected latitude 40.7128, got %f", metadata.GPSLatitude)
	}
	if metadata.GPSLongitude != -74.0060 {
		t.Errorf("expected longitude -74.0060, got %f", metadata.GPSLongitude)
	}
}

func TestParseGPSCoordinates_London(t *testing.T) {
	metadata := &PhoneMetadata{}
	parseGPSCoordinates("+51.5074-0.1278/", metadata)

	if !metadata.HasGPS {
		t.Error("expected HasGPS to be true")
	}
	if math.Abs(metadata.GPSLatitude-51.5074) > 0.0001 {
		t.Errorf("expected latitude ~51.5074, got %f", metadata.GPSLatitude)
	}
	if math.Abs(metadata.GPSLongitude-(-0.1278)) > 0.0001 {
		t.Errorf("expected longitude ~-0.1278, got %f", metadata.GPSLongitude)
	}
}

func TestSplitLocation(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"+37.7749-122.4194/+", []string{"+37.7749", "-122.4194", "+"}},
		{"+37.7749-122.4194/", []string{"+37.7749", "-122.4194", ""}},
		{"+37.7749-122.4194", []string{"+37.7749", "-122.4194"}},
		{"+", []string{"+"}},
		{"+37.7749", []string{"+37.7749"}},
	}

	for _, tt := range tests {
		result := splitLocation(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("splitLocation(%q) returned %d parts, expected %d: %v", tt.input, len(result), len(tt.expected), result)
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("splitLocation(%q)[%d] = %q, expected %q", tt.input, i, v, tt.expected[i])
			}
		}
	}
}

func TestExtractFromMdtoBox(t *testing.T) {
	mdta := &MdtoBox{
		Items: map[string]string{
			"mdta.1001": "iPhone 15 Pro",
			"mdta.1002": "iOS 17.0",
			"mdta.3004": "+37.7749-122.4194/+10.0",
			"mdta.3007": "1",
			"mdta.3008": "HEVC",
			"mdta.4016": "AAC",
		},
	}

	metadata := &PhoneMetadata{}
	extractFromMdtoBox(mdta, metadata)

	if metadata.DeviceModel != "iPhone 15 Pro" {
		t.Errorf("expected device model 'iPhone 15 Pro', got %q", metadata.DeviceModel)
	}
	if metadata.DeviceSoftware != "iOS 17.0" {
		t.Errorf("expected device software 'iOS 17.0', got %q", metadata.DeviceSoftware)
	}
	if !metadata.HasGPS {
		t.Error("expected HasGPS to be true")
	}
	if metadata.GPSLatitude != 37.7749 {
		t.Errorf("expected latitude 37.7749, got %f", metadata.GPSLatitude)
	}
	if !metadata.Stabilization {
		t.Error("expected stabilization to be true")
	}
	if metadata.VideoEncoder != "HEVC" {
		t.Errorf("expected video encoder 'HEVC', got %q", metadata.VideoEncoder)
	}
	if metadata.AudioCodec != "AAC" {
		t.Errorf("expected audio codec 'AAC', got %q", metadata.AudioCodec)
	}
}

func TestExtractFromMdtoBox_StabilizationFalse(t *testing.T) {
	mdta := &MdtoBox{
		Items: map[string]string{
			"mdta.3007": "0",
		},
	}

	metadata := &PhoneMetadata{}
	extractFromMdtoBox(mdta, metadata)

	if metadata.Stabilization {
		t.Error("expected stabilization to be false")
	}
}

func TestExtractFromMdtoBox_TrueString(t *testing.T) {
	mdta := &MdtoBox{
		Items: map[string]string{
			"mdta.3007": "true",
		},
	}

	metadata := &PhoneMetadata{}
	extractFromMdtoBox(mdta, metadata)

	if !metadata.Stabilization {
		t.Error("expected stabilization to be true")
	}
}

func TestParseAtom_EmptyData(t *testing.T) {
	data := []byte{0, 0, 0, 8, 'f', 'r', 'e', 'e'}
	var buf bytes.Buffer
	buf.Write(data)

	atom, err := parseAtom(&buf, 0)
	if err != nil {
		t.Fatalf("parseAtom failed: %v", err)
	}

	if atom.Type != "free" {
		t.Errorf("expected type 'free', got %q", atom.Type)
	}
	if atom.Size != 8 {
		t.Errorf("expected size 8, got %d", atom.Size)
	}
	if atom.Data != nil {
		t.Error("expected nil data for empty atom")
	}
}

func TestParseAtom_TruncatedData(t *testing.T) {
	data := make([]byte, 20)
	binary.BigEndian.PutUint32(data[0:4], 100)
	copy(data[4:8], "free")

	var buf bytes.Buffer
	buf.Write(data)

	_, err := parseAtom(&buf, 0)
	if err == nil {
		t.Fatal("expected error for truncated atom data")
	}
}

func TestBytesReader(t *testing.T) {
	br := &bytesReader{data: []byte("hello")}

	buf := make([]byte, 5)
	n, err := br.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes read, got %d", n)
	}
	if string(buf) != "hello" {
		t.Errorf("expected 'hello', got %q", string(buf))
	}

	buf2 := make([]byte, 1)
	_, err = br.Read(buf2)
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestIsContainerAtom(t *testing.T) {
	tests := []struct {
		atomType string
		expected bool
	}{
		{AtomMoov, true},
		{AtomTrak, true},
		{AtomMdia, true},
		{AtomMinf, true},
		{AtomStbl, true},
		{AtomStsd, true},
		{AtomDref, true},
		{AtomEdts, true},
		{AtomMeta, true},
		{AtomIlst, true},
		{AtomIpco, true},
		{AtomMvex, true},
		{AtomMvhd, false},
		{AtomTkhd, false},
		{AtomMdhd, false},
		{AtomFree, false},
		{AtomMdat, false},
		{AtomHdlr, false},
	}

	for _, tt := range tests {
		result := isContainerAtom(tt.atomType)
		if result != tt.expected {
			t.Errorf("isContainerAtom(%q) = %v, expected %v", tt.atomType, result, tt.expected)
		}
	}
}

func TestFindAtomType(t *testing.T) {
	data := make([]byte, 20)
	copy(data[10:14], "avcC")

	idx := findAtomType(data, "avcC")
	if idx != 10 {
		t.Errorf("expected offset 10, got %d", idx)
	}

	idx = findAtomType(data, "hvcC")
	if idx != -1 {
		t.Errorf("expected -1 for missing atom, got %d", idx)
	}
}

func TestParseQuickTimeFile_NonExistentFile(t *testing.T) {
	_, err := ParseQuickTimeFile("/nonexistent/file.mp4")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestParseQuickTimeFile_EmptyReader(t *testing.T) {
	var buf bytes.Buffer
	_, err := ParseQuickTimeFromReader(&buf)
	if err == nil {
		t.Fatal("expected error for empty reader")
	}
}

func TestParseQuickTimeFile_JustFtyp(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(buildAtomBytes(AtomFTyp, make([]byte, 12)))

	qtf, err := ParseQuickTimeFromReader(&buf)
	if err != nil {
		t.Fatalf("ParseQuickTimeFromReader failed: %v", err)
	}

	if qtf.FTyp == nil {
		t.Fatal("expected ftyp box")
	}
	if qtf.Moov != nil {
		t.Error("expected nil moov box")
	}
}

func TestExtractPhoneMetadata_EmptyContainer(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(buildAtomBytes(AtomFTyp, make([]byte, 12)))
	buf.Write(buildAtomBytes(AtomMoov, nil))

	qtf, err := ParseQuickTimeFromReader(&buf)
	if err != nil {
		t.Fatalf("ParseQuickTimeFromReader failed: %v", err)
	}

	metadata := extractPhoneMetadataFromContainer(qtf)
	if metadata == nil {
		t.Fatal("expected non-nil metadata")
	}
	if metadata.CreationDate != "" {
		t.Error("expected empty creation date")
	}
	if metadata.DeviceModel != "" {
		t.Error("expected empty device model")
	}
}

func TestExtractPhoneMetadata_WithMdtoBox(t *testing.T) {
	metaBuf := bytes.NewBuffer(nil)
	keysData := make([]byte, 20)
	keysData[0] = 0
	binary.BigEndian.PutUint32(keysData[9:13], 1)
	copy(keysData[13:17], "test")
	mdtoItem := buildAtomBytes(AtomKeys, keysData)
	metaBuf.Write(mdtoItem)

	meta := buildAtomBytes(AtomMeta, metaBuf.Bytes())

	moovBuf := bytes.NewBuffer(nil)
	moovBuf.Write(meta)

	var mp4 bytes.Buffer
	mp4.Write(buildAtomBytes(AtomFTyp, make([]byte, 12)))
	mp4.Write(buildAtomBytes(AtomMoov, moovBuf.Bytes()))

	qtf, err := ParseQuickTimeFromReader(&mp4)
	if err != nil {
		t.Fatalf("ParseQuickTimeFromReader failed: %v", err)
	}

	if qtf.Moov == nil {
		t.Fatal("expected moov")
	}
	if qtf.Moov.Meta == nil {
		t.Fatal("expected meta")
	}
	if qtf.Moov.Meta.Mdta == nil {
		t.Fatal("expected mdta")
	}

	metadata := extractPhoneMetadataFromContainer(qtf)
	if metadata == nil {
		t.Fatal("expected non-nil metadata")
	}
}

func TestParseQuickTimeFile_RealTestVideo(t *testing.T) {
	if _, err := os.Stat("testdata/ten_second.mp4"); os.IsNotExist(err) {
		t.Skip("testdata/ten_second.mp4 not found, skipping")
	}

	qtf, err := ParseQuickTimeFile("testdata/ten_second.mp4")
	if err != nil {
		t.Fatalf("ParseQuickTimeFile failed: %v", err)
	}

	if qtf.FTyp == nil {
		t.Fatal("expected ftyp box")
	}
	if qtf.Moov == nil {
		t.Fatal("expected moov box")
	}
	if qtf.Moov.Mvhd == nil {
		t.Fatal("expected mvhd in moov")
	}
	if qtf.Moov.Mvhd.TimeScale == 0 {
		t.Error("expected non-zero time scale")
	}
	if qtf.Moov.Mvhd.Duration == 0 {
		t.Error("expected non-zero duration")
	}
}

func TestParseQuickTimeFile_RealHDRTVideo(t *testing.T) {
	if _, err := os.Stat("testdata/hdr_like.mp4"); os.IsNotExist(err) {
		t.Skip("testdata/hdr_like.mp4 not found, skipping")
	}

	qtf, err := ParseQuickTimeFile("testdata/hdr_like.mp4")
	if err != nil {
		t.Fatalf("ParseQuickTimeFile failed: %v", err)
	}

	if qtf.FTyp == nil {
		t.Fatal("expected ftyp box")
	}
	if qtf.Moov == nil {
		t.Fatal("expected moov box")
	}
}

func TestParseQuickTimeFile_AllTestVideos(t *testing.T) {
	videos := []string{
		"testdata/ten_second.mp4",
		"testdata/thirty_second.mp4",
		"testdata/one_second.mp4",
		"testdata/hdr_like.mp4",
	}

	for _, video := range videos {
		t.Run(video, func(t *testing.T) {
			if _, err := os.Stat(video); os.IsNotExist(err) {
				t.Skipf("%s not found, skipping", video)
			}

			qtf, err := ParseQuickTimeFile(video)
			if err != nil {
				t.Fatalf("ParseQuickTimeFile(%s) failed: %v", video, err)
			}

			if qtf.FTyp == nil {
				t.Errorf("%s: expected ftyp box", video)
			}
			if qtf.Moov == nil {
				t.Errorf("%s: expected moov box", video)
			}
		})
	}
}
