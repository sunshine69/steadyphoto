// Package processor provides video and image metadata extraction utilities.
//
// QuickTime atom parser for MP4/MOV container files.
// This parser reads the binary structure of QuickTime containers to extract
// phone-specific metadata that ffprobe cannot access.
package processor

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// QuickTime atom errors
var (
	ErrAtomNotFound       = errors.New("atom not found in container")
	ErrInvalidAtomSize    = errors.New("invalid atom size")
	ErrTruncatedAtom      = errors.New("truncated atom data")
	ErrUnsupportedVersion = errors.New("unsupported atom version")
)

// QuickTimeAtom represents a parsed QuickTime atom/box.
type QuickTimeAtom struct {
	Size       int64             // Total size including header (8 bytes)
	Type       string            // 4-byte type code
	Version    byte              // Version field (if present)
	Flags      uint32            // Flags field (if present)
	HeaderSize int               // Size of header (8 or 16 bytes)
	Data       []byte            // Raw data after header
	Children   []*QuickTimeAtom  // Nested child atoms
}

// QuickTimeFile represents a parsed MP4/MOV container.
type QuickTimeFile struct {
	FTyp *FTypBox // File type box
	Moov *MoovBox // Movie metadata box
}

// FTypBox represents the file type box.
type FTypBox struct {
	MajorBrand     string
	MinorVersion   int
	CompatibleBrands []string
}

// MoovBox represents the movie metadata container.
type MoovBox struct {
	Mvhd    *MovieHeaderAtom
	Traks   []*TrackBox
	Meta    *MetaBox // Newer format metadata
}

// MovieHeaderAtom represents the mvhd atom.
type MovieHeaderAtom struct {
	Version      byte
	CreationTime uint64
	ModificationTime uint64
	TimeScale    uint32
	Duration     uint64
}

// TrackBox represents a track container.
type TrackBox struct {
	Tkhd *TrackHeaderAtom
	Mdia *MediaBox
}

// TrackHeaderAtom represents the tkhd atom.
type TrackHeaderAtom struct {
	Version  byte
	Flags    uint32 // 3-byte flags (lower 24 bits)
	Enabled  bool
	Width    int32  // Fixed-point 16.16
	Height   int32  // Fixed-point 16.16
	Matrix   []byte // 36 bytes transformation matrix
}

// MediaBox represents the mdia container.
type MediaBox struct {
	Mdhd *MediaHeaderAtom
	Hdlr *HandlerBox
	Minf *MediaInformationBox
}

// MediaHeaderAtom represents the mdhd atom.
type MediaHeaderAtom struct {
	Version      byte
	CreationTime uint64
	ModificationTime uint64
	TimeScale    uint32
	Duration     uint64
	Language     string // ISO-639-2/T language code
}

// HandlerBox represents the hdlr atom.
type HandlerBox struct {
	HandlerType string
	Description  string
}

// MediaInformationBox represents the minf container.
type MediaInformationBox struct {
	Stbl *SampleTableBox
}

// SampleTableBox represents the stbl container.
type SampleTableBox struct {
	Stsd *SampleDescriptionBox
}

// SampleDescriptionBox represents the stsd atom.
type SampleDescriptionBox struct {
	DescriptionType string
	Data           []byte
	Codecs         []CodecConfig
}

// CodecConfig represents codec-specific configuration data.
type CodecConfig struct {
	Type     string
	Data     []byte
	Width    int
	Height   int
	Depth    int
}

// MetaBox represents the meta atom (newer metadata format).
type MetaBox struct {
	Mdta *MdtoBox
}

// MdtoBox represents the mdta atom (Apple metadata).
type MdtoBox struct {
	Items map[string]string // key -> value
}

// PhoneMetadata represents extracted phone-specific metadata.
type PhoneMetadata struct {
	DeviceMake       string
	DeviceModel      string
	DeviceSoftware   string
	CreationDate     string
	GPSLatitude      float64
	GPSLongitude     float64
	GPSAltitude      float64
	HasGPS           bool
	Stabilization    bool
	VideoEncoder     string
	VideoLevel       string
	AudioCodec       string
	AudioSampleRate  int
	AudioChannels    int
	AudioBitDepth    int
	AudioBitRate     int
	ColorPrimaries   string
	TransferFunction string
	IsHDR            bool
	Caption          string
	Title            string
}

// ParseQuickTimeFile parses a QuickTime container file.
func ParseQuickTimeFile(filePath string) (*QuickTimeFile, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	return ParseQuickTimeFromReader(f)
}

// ParseQuickTimeFromReader parses a QuickTime container from an io.Reader.
func ParseQuickTimeFromReader(r io.Reader) (*QuickTimeFile, error) {
	qtf := &QuickTimeFile{}

	// Parse atoms at root level
	children, err := parseAtoms(r, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse root atoms: %w", err)
	}

	// Find ftyp
	for _, atom := range children {
		if atom.Type == AtomFTyp {
			qtf.FTyp = parseFTypBox(atom)
		} else if atom.Type == AtomMoov {
			qtf.Moov = parseMoovBox(atom)
		}
	}

	if qtf.Moov == nil {
		return nil, ErrAtomNotFound
	}

	return qtf, nil
}

// parseAtoms reads atoms from a reader at the given nesting level.
func parseAtoms(r io.Reader, depth int) ([]*QuickTimeAtom, error) {
	if depth > 20 {
		return nil, fmt.Errorf("atom nesting too deep (max 20)")
	}

	var atoms []*QuickTimeAtom

	for {
		atom, err := parseAtom(r, depth)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return atoms, err
		}
		atoms = append(atoms, atom)
	}

	return atoms, nil
}

// parseAtom reads a single atom from the reader.
func parseAtom(r io.Reader, depth int) (*QuickTimeAtom, error) {
	// Read size and type (8 bytes)
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}

	size := int64(binary.BigEndian.Uint32(header[0:4]))
	atomType := string(header[4:8])

	// Calculate header size
	headerSize := 8
	if size == 1 {
		// 64-bit extended size
		headerSize = 16
	}

	// Read extended size if present
	var extendedSize int64
	if size == 1 {
		var extBuf [8]byte
		if _, err := io.ReadFull(r, extBuf[:]); err != nil {
			return nil, err
		}
		extendedSize = int64(binary.BigEndian.Uint64(extBuf[:]))
		if extendedSize <= 0 {
			return nil, ErrInvalidAtomSize
		}
		size = extendedSize
	}

	// Handle special case: size 0 means "until EOF"
	if size == 0 {
		// Read rest of file (we'll handle this differently for root atoms)
		// For now, seek to end to determine size
		return nil, fmt.Errorf("size 0 not supported for root atoms")
	}

	// Read version and flags (4 bytes) for most atoms
	dataStart := headerSize
	dataSize := size - int64(dataStart)
	if dataSize <= 0 {
		return &QuickTimeAtom{
			Size:       size,
			Type:       atomType,
			Version:    0,
			Flags:      0,
			HeaderSize: dataStart,
			Data:       nil,
		}, nil
	}

	data := make([]byte, dataSize)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("failed to read atom data: %w", err)
	}

	// Parse version and flags (first 4 bytes of data for most atoms)
	version := byte(0)
	flags := uint32(0)
	payloadStart := 0

	if dataSize >= 4 && !isContainerAtom(atomType) {
		version = data[0]
		flags = binary.BigEndian.Uint32(data[1:4]) & 0x00FFFFFF
		payloadStart = 4
	}

	atom := &QuickTimeAtom{
		Size:       size,
		Type:       atomType,
		Version:    version,
		Flags:      flags,
		HeaderSize: dataStart,
		Data:       data,
	}

	// Parse child atoms for container types
	if isContainerAtom(atomType) {
		childReader := &bytesReader{data: data[payloadStart:]}
		children, err := parseAtoms(childReader, depth+1)
		if err != nil {
			// Don't fail on nested parsing errors, just skip children
			return atom, nil
		}
		atom.Children = children
	}

	return atom, nil
}

// isContainerAtom checks if an atom type is a container that can have children.
func isContainerAtom(atomType string) bool {
	containerAtoms := map[string]bool{
		AtomMoov:  true,
		AtomTrak:  true,
		AtomMdia:  true,
		AtomMinf:  true,
		AtomStbl:  true,
		AtomStsd:  true,
		AtomDref:  true,
		AtomEdts:  true,
		AtomMeta:  true,
		AtomIlst:  true,
		AtomIpco:  true,
		AtomMvex:  true,
	}
	return containerAtoms[atomType]
}

// parseFTypBox parses a file type box.
func parseFTypBox(atom *QuickTimeAtom) *FTypBox {
	ftyp := &FTypBox{}
	data := atom.Data

	// Skip version and flags (4 bytes)
	if len(data) < 8 {
		return ftyp
	}

	// Major brand (4 bytes)
	ftyp.MajorBrand = string(data[4:8])

	// Minor version (4 bytes)
	if len(data) >= 12 {
		ftyp.MinorVersion = int(binary.BigEndian.Uint32(data[8:12]))
	}

	// Compatible brands (4 bytes each)
	for i := 12; i+4 <= len(data); i += 4 {
		brand := string(data[i : i+4])
		// Skip null/empty brands
		if len(brand) == 4 && brand[0] == 0 && brand[1] == 0 && brand[2] == 0 && brand[3] == 0 {
			break
		}
		ftyp.CompatibleBrands = append(ftyp.CompatibleBrands, brand)
	}

	return ftyp
}

// parseMoovBox parses the movie metadata container.
func parseMoovBox(atom *QuickTimeAtom) *MoovBox {
	moov := &MoovBox{}

	for _, child := range atom.Children {
		switch child.Type {
		case AtomMvhd:
			moov.Mvhd = parseMovieHeaderAtom(child)
		case AtomTrak:
			trak := parseTrackBox(child)
			if trak != nil {
				moov.Traks = append(moov.Traks, trak)
			}
		case AtomMeta:
			moov.Meta = parseMetaBox(child)
		}
	}

	return moov
}

// parseMovieHeaderAtom parses the mvhd atom.
func parseMovieHeaderAtom(atom *QuickTimeAtom) *MovieHeaderAtom {
	mvhd := &MovieHeaderAtom{}
	data := atom.Data

	if len(data) < 20 {
		return mvhd
	}

	mvhd.Version = data[0]

	if mvhd.Version == 0 {
		// Version 0: 32-bit times
		if len(data) >= 24 {
			mvhd.CreationTime = uint64(binary.BigEndian.Uint32(data[4:8]))
			mvhd.ModificationTime = uint64(binary.BigEndian.Uint32(data[8:12]))
			mvhd.TimeScale = binary.BigEndian.Uint32(data[12:16])
			mvhd.Duration = uint64(binary.BigEndian.Uint32(data[16:20]))
		}
	} else {
		// Version 1: 64-bit times
		if len(data) >= 32 {
			mvhd.CreationTime = binary.BigEndian.Uint64(data[4:12])
			mvhd.ModificationTime = binary.BigEndian.Uint64(data[12:20])
			mvhd.TimeScale = binary.BigEndian.Uint32(data[20:24])
			mvhd.Duration = binary.BigEndian.Uint64(data[24:32])
		}
	}

	return mvhd
}

// parseTrackBox parses a track container.
func parseTrackBox(atom *QuickTimeAtom) *TrackBox {
	trak := &TrackBox{}

	for _, child := range atom.Children {
		switch child.Type {
		case AtomTkhd:
			trak.Tkhd = parseTrackHeaderAtom(child)
		case AtomMdia:
			trak.Mdia = parseMediaBox(child)
		}
	}

	return trak
}

// parseTrackHeaderAtom parses the tkhd atom.
func parseTrackHeaderAtom(atom *QuickTimeAtom) *TrackHeaderAtom {
	tkhd := &TrackHeaderAtom{}
	data := atom.Data

	if len(data) < 84 {
		return tkhd
	}

	tkhd.Version = data[0]
	tkhd.Flags = binary.BigEndian.Uint32(data[1:4]) & 0x00FFFFFF
	tkhd.Enabled = tkhd.Flags&0x1 == 1

	if tkhd.Version == 0 {
		if len(data) >= 84 {
			tkhd.Width = int32(binary.BigEndian.Uint32(data[68:72]))
			tkhd.Height = int32(binary.BigEndian.Uint32(data[72:76]))
		}
	} else {
		if len(data) >= 92 {
			tkhd.Width = int32(binary.BigEndian.Uint32(data[80:84]))
			tkhd.Height = int32(binary.BigEndian.Uint32(data[84:88]))
		}
	}

	return tkhd
}

// parseMediaBox parses the mdia container.
func parseMediaBox(atom *QuickTimeAtom) *MediaBox {
	mdia := &MediaBox{}

	for _, child := range atom.Children {
		switch child.Type {
		case AtomMdhd:
			mdia.Mdhd = parseMediaHeaderAtom(child)
		case AtomHdlr:
			mdia.Hdlr = parseHandlerBox(child)
		case AtomMinf:
			mdia.Minf = parseMediaInformationBox(child)
		}
	}

	return mdia
}

// parseMediaHeaderAtom parses the mdhd atom.
func parseMediaHeaderAtom(atom *QuickTimeAtom) *MediaHeaderAtom {
	mdhd := &MediaHeaderAtom{}
	data := atom.Data

	if len(data) < 20 {
		return mdhd
	}

	mdhd.Version = data[0]

	if mdhd.Version == 0 {
		if len(data) >= 28 {
			mdhd.CreationTime = uint64(binary.BigEndian.Uint32(data[4:8]))
			mdhd.ModificationTime = uint64(binary.BigEndian.Uint32(data[8:12]))
			mdhd.TimeScale = binary.BigEndian.Uint32(data[12:16])
			mdhd.Duration = uint64(binary.BigEndian.Uint32(data[16:20]))
			// Language code (3 bytes) + padding (1 byte)
			if len(data) >= 28 {
				lang := data[24:27]
				mdhd.Language = fmt.Sprintf("%c%c%c", lang[0], lang[1], lang[2])
			}
		}
	} else {
		if len(data) >= 40 {
			mdhd.CreationTime = binary.BigEndian.Uint64(data[4:12])
			mdhd.ModificationTime = binary.BigEndian.Uint64(data[12:20])
			mdhd.TimeScale = binary.BigEndian.Uint32(data[20:24])
			mdhd.Duration = binary.BigEndian.Uint64(data[24:32])
			// Language code (3 bytes) + padding (1 byte)
			if len(data) >= 40 {
				lang := data[36:39]
				mdhd.Language = fmt.Sprintf("%c%c%c", lang[0], lang[1], lang[2])
			}
		}
	}

	return mdhd
}

// parseHandlerBox parses the hdlr atom.
func parseHandlerBox(atom *QuickTimeAtom) *HandlerBox {
	hdlr := &HandlerBox{}
	data := atom.Data

	if len(data) < 20 {
		return hdlr
	}

	// Skip version, flags, pre_defined (9 bytes)
	// Handler type (4 bytes)
	if len(data) >= 20 {
		hdlr.HandlerType = string(data[9:13])
		// Description (null-terminated string)
		for i := 13; i < len(data); i++ {
			if data[i] == 0 {
				hdlr.Description = string(data[13:i])
				break
			}
			if i == len(data)-1 {
				hdlr.Description = string(data[13:])
			}
		}
	}

	return hdlr
}

// parseMediaInformationBox parses the minf container.
func parseMediaInformationBox(atom *QuickTimeAtom) *MediaInformationBox {
	minf := &MediaInformationBox{}

	for _, child := range atom.Children {
		switch child.Type {
		case AtomStbl:
			minf.Stbl = parseSampleTableBox(child)
		}
	}

	return minf
}

// parseSampleTableBox parses the stbl container.
func parseSampleTableBox(atom *QuickTimeAtom) *SampleTableBox {
	stbl := &SampleTableBox{}

	for _, child := range atom.Children {
		switch child.Type {
		case AtomStsd:
			stbl.Stsd = parseSampleDescriptionBox(child)
		}
	}

	return stbl
}

// parseSampleDescriptionBox parses the stsd atom.
func parseSampleDescriptionBox(atom *QuickTimeAtom) *SampleDescriptionBox {
	stsd := &SampleDescriptionBox{}
	data := atom.Data

	if len(data) < 8 {
		return stsd
	}

	// Skip version and flags (4 bytes)
	// Number of entries (4 bytes)
	numEntries := int(binary.BigEndian.Uint32(data[4:8]))
	if numEntries == 0 {
		return stsd
	}

	// Parse first entry (simplified)
	if len(data) >= 16 {
		stsd.DescriptionType = string(data[8:12])
		stsd.Data = data[16:]
		stsd.Codecs = parseCodecConfigs(data[16:], stsd.DescriptionType)
	}

	return stsd
}

// parseCodecConfigs parses codec configuration atoms from sample entries.
func parseCodecConfigs(data []byte, descType string) []CodecConfig {
	configs := []CodecConfig{}

	// Look for known codec atoms in the data
	atomTypes := []string{AtomAvcC, AtomHvcC, AtomAv1C, AtomEsds}
	for _, at := range atomTypes {
		idx := findAtomType(data, at)
		if idx >= 0 {
			configs = append(configs, CodecConfig{
				Type: at,
				Data: data[idx:],
			})
		}
	}

	return configs
}

// findAtomType finds the byte offset of an atom type within data.
func findAtomType(data []byte, atomType string) int {
	pattern := []byte(atomType)
	for i := 0; i+4 <= len(data); i++ {
		if data[i] == pattern[0] && data[i+1] == pattern[1] &&
			data[i+2] == pattern[2] && data[i+3] == pattern[3] {
			return i
		}
	}
	return -1
}

// parseMetaBox parses the meta atom (newer metadata format).
func parseMetaBox(atom *QuickTimeAtom) *MetaBox {
	meta := &MetaBox{}

	for _, child := range atom.Children {
		if child.Type == AtomMdta {
			meta.Mdta = parseMdtoBox(child)
		}
	}

	return meta
}

// parseMdtoBox parses the mdta atom (Apple metadata).
func parseMdtoBox(atom *QuickTimeAtom) *MdtoBox {
	mdta := &MdtoBox{
		Items: make(map[string]string),
	}

	for _, child := range atom.Children {
		if child.Type == AtomKeys {
			// Parse keys
			keys := parseKeysAtom(child)
			for _, key := range keys {
				mdta.Items[key] = ""
			}
		} else if child.Type == "data" {
			// Parse data entries
			values := parseDataAtom(child)
			for i, val := range values {
				if i < len(mdta.Items) {
					// This is a simplified parser - real implementation would match keys to data
					_ = val
				}
			}
		}
	}

	return mdta
}

// parseKeysAtom parses the keys atom to extract key identifiers.
func parseKeysAtom(atom *QuickTimeAtom) []string {
	keys := []string{}
	data := atom.Data

	// Simplified parser: look for 4-byte key identifiers
	// Real implementation would parse the proper keys structure
	if len(data) > 20 {
		// Skip version, flags, pre_defined (9 bytes)
		// Number of keys (4 bytes)
		numKeys := int(binary.BigEndian.Uint32(data[9:13]))

		if numKeys > 0 && len(data) >= 17+numKeys*4 {
			offset := 13
			for i := 0; i < numKeys && offset+4 <= len(data); i++ {
				key := string(data[offset : offset+4])
				keys = append(keys, key)
				offset += 4
			}
		}
	}

	return keys
}

// parseDataAtom parses the data atom entries.
func parseDataAtom(atom *QuickTimeAtom) []string {
	values := []string{}
	data := atom.Data

	// Simplified parser
	_ = data
	return values
}

// ExtractPhoneMetadata extracts phone-specific metadata from a QuickTime file.
func ExtractPhoneMetadata(filePath string) (*PhoneMetadata, error) {
	qtf, err := ParseQuickTimeFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse QuickTime file: %w", err)
	}

	return extractPhoneMetadataFromContainer(qtf), nil
}

// extractPhoneMetadataFromContainer extracts phone metadata from a parsed container.
func extractPhoneMetadataFromContainer(qtf *QuickTimeFile) *PhoneMetadata {
	metadata := &PhoneMetadata{}

	// Extract from moov/mvhd
	if qtf.Moov != nil && qtf.Moov.Mvhd != nil {
		if qtf.Moov.Mvhd.CreationTime > 0 {
			// Convert Unix timestamp to ISO 8601
			// QuickTime uses 1904 as epoch, Unix uses 1970
			unixTime := int64(qtf.Moov.Mvhd.CreationTime) - 2082844800
			if unixTime > 0 {
				metadata.CreationDate = fmt.Sprintf("%d", unixTime)
			}
		}
	}

	// Extract from meta/mdta if available
	if qtf.Moov != nil && qtf.Moov.Meta != nil && qtf.Moov.Meta.Mdta != nil {
		extractFromMdtoBox(qtf.Moov.Meta.Mdta, metadata)
	}

	return metadata
}

// extractFromMdtoBox extracts metadata from an mdta box.
func extractFromMdtoBox(mdta *MdtoBox, metadata *PhoneMetadata) {
	for key, value := range mdta.Items {
		switch key {
		case "mdta.1001", "mdta.2000":
			metadata.DeviceModel = value
		case "mdta.1002":
			metadata.DeviceSoftware = value
		case "mdta.1000":
			metadata.CreationDate = value
		case "mdta.3004":
			parseGPSCoordinates(value, metadata)
		case "mdta.3007":
			metadata.Stabilization = value == "1" || value == "true"
		case "mdta.3008":
			metadata.VideoEncoder = value
		case "mdta.4016":
			metadata.AudioCodec = value
		}
	}
}

// parseGPSCoordinates parses ISO 6709 location format.
func parseGPSCoordinates(location string, metadata *PhoneMetadata) {
	// Format: +DD.DDDD±DDD.DDDD/±DDD.DDDD
	// Example: +37.7749-122.4194/+10.0

	// Find the separator
	parts := splitLocation(location)
	if len(parts) < 2 {
		return
	}

	// Parse latitude and longitude
	lat, err := parseCoordinate(parts[0])
	if err != nil {
		return
	}

	lon, err := parseCoordinate(parts[1])
	if err != nil {
		return
	}

	metadata.GPSLatitude = lat
	metadata.GPSLongitude = lon
	metadata.HasGPS = true

	// Parse altitude if present
	if len(parts) > 2 {
		alt, err := parseCoordinate(parts[2])
		if err == nil {
			metadata.GPSAltitude = alt
		}
	}
}

// splitLocation splits an ISO 6709 location string.
func splitLocation(location string) []string {
	parts := []string{}
	current := ""

	for _, c := range location {
		if c == '+' || c == '-' {
			if current != "" {
				parts = append(parts, current)
			}
			current = string(c)
		} else if c == '/' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// parseCoordinate parses a coordinate string to float64.
func parseCoordinate(s string) (float64, error) {
	var result float64
	sign := 1.0

	for i, c := range s {
		if c == '-' {
			sign = -1.0
		} else if c >= '0' && c <= '9' {
			result = result*10 + float64(c-'0')
			if i > 0 {
				// Check if we're past the decimal point
				hasDecimal := false
				for _, prev := range s[:i] {
					if prev == '.' {
						hasDecimal = true
						break
					}
				}
				if !hasDecimal {
					// Integer part
				}
			}
		}
	}

	return result * sign, nil
}

// bytesReader wraps a byte slice to implement io.Reader.
type bytesReader struct {
	data []byte
	pos  int
}

func (br *bytesReader) Read(p []byte) (n int, err error) {
	if br.pos >= len(br.data) {
		return 0, io.EOF
	}
	n = copy(p, br.data[br.pos:])
	br.pos += n
	return n, nil
}
