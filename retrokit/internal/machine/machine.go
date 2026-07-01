// Package machine defines the abstraction for retro computer cassette formats.
//
// Each platform (MSX, TRS-80, ZX81, Apple II, ...) implements the Machine
// interface, which encapsulates all cassette-format knowledge: audio
// modulation/demodulation, block structure, binary wrapping and format
// detection. The conversion commands delegate to a Machine looked up by name,
// so adding a new platform does not require changes to any command package.
package machine

// Block describes a logical block within a cassette image.
type Block struct {
	Type     string
	Offset   int
	Size     int
	LoadAddr uint16
	ExecAddr uint16
	Filename string
	Raw      []byte
}

// CASAnalysis is the result of parsing a cassette image's structure.
type CASAnalysis struct {
	Valid    bool
	Blocks   []Block
	DataSize int
}

// EncodeOpts controls audio encoding (CAS bytes to PCM samples).
type EncodeOpts struct {
	BaudRate int
	GapTime  int
}

// DecodeOpts controls audio decoding (PCM samples to CAS bytes).
type DecodeOpts struct {
	Threshold int
	Envelope  int
	Normalize bool
	Window    float64
}

// WrapOpts controls wrapping a raw binary payload into a CAS image.
type WrapOpts struct {
	Type string
	Addr uint16
	Exec uint16
}

// Machine encapsulates all cassette-format knowledge for a single platform.
type Machine interface {
	// Name returns the machine identifier (e.g. "msx", "trs80").
	Name() string

	// DisplayName returns a human-readable name (e.g. "MSX", "TRS-80").
	DisplayName() string

	// BaudRates returns the baud rates supported by this machine.
	BaudRates() []int

	// DefaultBaudRate returns the default baud rate for encoding.
	DefaultBaudRate() int

	// Detect reports whether the given CAS data looks like this machine's
	// cassette format.
	Detect(data []byte) bool

	// Parse analyzes the CAS binary structure and returns its blocks.
	Parse(data []byte) *CASAnalysis

	// Wrap wraps a raw binary payload into this machine's CAS format.
	Wrap(payload []byte, opts WrapOpts) []byte

	// Extract strips the CAS framing and returns the raw payload plus the
	// logical blocks found.
	Extract(data []byte) (payload []byte, blocks []Block)

	// Encode produces 8-bit unsigned mono PCM samples (at the package's WAV
	// sample rate) from CAS bytes.
	Encode(casData []byte, opts EncodeOpts) []byte

	// Decode produces CAS bytes from signed 8-bit mono samples.
	Decode(samples []int8, sampleRate uint32, opts DecodeOpts) []byte
}
