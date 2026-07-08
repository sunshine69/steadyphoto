// Package processor provides video and image metadata extraction utilities.
//
// QuickTime atom type constants used in MP4/MOV containers.
// Reference: ISO 14496-12 (MP4 Container Specification)
// https://developer.apple.com/library/archive/documentation/QuickTime/QTFF/QTFFPreface/qtffPreface.html
package processor

// Atom type constants — these are the 4-byte type codes used in MP4/MOV containers.
const (
	AtomFTyp = "ftyp" // File type box (root)
	AtomMoov = "moov" // Movie metadata container (root)
	AtomMvhd = "mvhd" // Movie header box
	AtomTrak = "trak" // Track container
	AtomTkhd = "tkhd" // Track header box
	AtomMdia = "mdia" // Media information container
	AtomMdhd = "mdhd" // Media header box
	AtomMinf = "minf" // Media information container
	AtomStbl = "stbl" // Sample table box
	AtomStsd = "stsd" // Sample description box
	AtomMdat = "mdat" // Media data box
	AtomStsz = "stsz" // Sample size box
	AtomStts = "stts" // Time-to-sample box
	AtomStsc = "stsc" // Sample-to-chunk box
	AtomStco = "stco" // Chunk offset box
	AtomCo64 = "co64" // Chunk offset 64-bit box
	AtomCmov = "cmov" // Compressed movie (rare)
	AtomDref = "dref" // Data reference box
	AtomEdts = "edts" // Edit box
	AtomElst = "elst" // Edit list box
	AtomFree = "free" // Free space box
	AtomJp2h = "jp2h" // JPEG 2000 header
	AtomMeta = "meta" // Metadata container
	AtomIlst = "ilst" // Item list (older metadata format)
	AtomIinf = "iinf" // Item information
	AtomIloc = "iloc" // Item location
	AtomIpco = "ipco" // Item property container
	AtomIpma = "ipma" // Item property association
	AtomIprp = "iprp" // Item properties box
	AtomIspe = "ispe" // Image spatial extension
	AtomPixi = "pixi" // Pixel information
	AtomPitm = "pitm" // Primary item
	AtomUrii = "urii" // URI box
	AtomUrng = "urn:" // URN namespace
	AtomMvex = "mvex" // Movie extended box
	AtomTrex = "trex" // Track extended box
	AtomSidx = "sidx" // Segment index
	AtomPssh = "pssh" // Protection system-specific header
	AtomColr = "colr" // Color info
	AtomCicp = "cicp" // Content color information
	AtomNclx = "nclx" // NCLX color properties
	AtomAv1C = "av1C" // AV1 codec config
	AtomHvcC = "hvcC" // HEVC codec config
	AtomAvcC = "avcC" // AVC/H.264 codec config
	AtomEsds = "esds" // MPEG-4 ES descriptor
	AtomMp4a = "mp4a" // AAC audio codec
	AtomAlac = "alac" // Apple Lossless audio codec
	AtomWAV  = "WAV " // PCM audio codec
	AtomSinf = "sinf" // Scheme information
	AtomSchi = "schi" // Scheme choice
	AtomSchm = "schm" // Scheme type
	AtomTrpy = "trpy" // Track power state
	AtomTrgr = "trgr" // Track power state group
	AtomCslg = "cslg" // Composition time-to-sample
	AtomMdta = "mdta" // Metadata dictionary (key-value pairs) — KEY for phone metadata
	AtomKeys = "keys" // Metadata keys (inside ilst)
	AtomData = "data" // Metadata data (inside ilst)
	AtomHdlr = "hdlr" // Handler box
)

// Apple-specific QuickTime metadata keys (4-byte string identifiers).
const (
	AppleCreationDate         = "mdta.1000"
	AppleModel                = "mdta.1001"
	AppleSoftware             = "mdta.1002"
	AppleMake                 = "mdta.2000"
	AppleCamera               = "mdta.2002"
	AppleLocation             = "mdta.3004"
	AppleStabilization        = "mdta.3007"
	AppleVideoEncoder         = "mdta.3008"
	AppleVideoLevel           = "mdta.3009"
	AppleColorPrimaries       = "mdta.3010"
	AppleTransferFunc         = "mdta.3011"
	AppleVideoHDR             = "mdta.3012"
	AppleAudioChannelLayout   = "mdta.4001"
	AppleCaption              = "mdta.4010"
	AppleTitle                = "mdta.4012"
	AppleAudioCodec           = "mdta.4016"
	AppleAudioSampleRate      = "mdta.4017"
	AppleAudioChannels        = "mdta.4018"
	AppleAudioBitDepth        = "mdta.4019"
	AppleAudioBitrate         = "mdta.4020"
	AppleAudioFormat          = "mdta.4021"
	AppleAudioProfile         = "mdta.4022"

	// ID3 tags
	ID3Title     = "MDTA.0000"
	ID3Artist    = "MDTA.0001"
	ID3Album     = "MDTA.0002"
	ID3Genre     = "MDTA.0003"
	ID3Year      = "MDTA.0004"
	ID3Comment   = "MDTA.0005"
)

// Known Apple metadata keys and their human-readable descriptions.
var appleMetadataKeys = map[string]string{
	AppleCreationDate:         "com.apple.quicktime.creationdate",
	AppleModel:                "com.apple.quicktime.model",
	AppleSoftware:             "com.apple.quicktime.software",
	AppleMake:                 "com.apple.quicktime.make",
	AppleCamera:               "com.apple.quicktime.camera",
	AppleLocation:             "com.apple.quicktime.location.ISO6709",
	AppleStabilization:        "com.apple.quicktime.stabilization",
	AppleVideoEncoder:         "com.apple.quicktime.video.encoder",
	AppleVideoLevel:           "com.apple.quicktime.video.level",
	AppleColorPrimaries:       "com.apple.quicktime.color.primarys",
	AppleTransferFunc:         "com.apple.quicktime.transfer.function",
	AppleVideoHDR:             "com.apple.quicktime.video.hdr",
	AppleAudioChannelLayout:   "com.apple.quicktime.audio.channel_layout",
	AppleCaption:              "com.apple.iPhoto.caption",
	AppleTitle:                "org.id3.TIT2",
	AppleAudioCodec:           "com.apple.quicktime.audio.codec",
	AppleAudioSampleRate:      "com.apple.quicktime.audio.sample_rate",
	AppleAudioChannels:        "com.apple.quicktime.audio.channels",
	AppleAudioBitDepth:        "com.apple.quicktime.audio.bit_depth",
	AppleAudioBitrate:         "com.apple.quicktime.audio.bit_rate",
	AppleAudioFormat:          "com.apple.quicktime.audio.format",
	AppleAudioProfile:         "com.apple.quicktime.audio.profile",
}
