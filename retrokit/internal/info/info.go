// Package info inspects cassette images, CMD files and WAV recordings, reporting
// their structure and metadata. Cassette parsing is delegated to the provided
// machine.Machine.
package info

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
	"github.com/carlosrabelo/retrokit/retrokit/internal/machine/trs80"
	"github.com/carlosrabelo/retrokit/retrokit/internal/wav"
)

// BlockInfo describes a block within an inspected file.
type BlockInfo = machine.Block

// Result holds the outcome of inspecting a file.
type Result struct {
	Path     string
	Format   string // "cas", "cmd", "wav", or "unknown"
	Machine  string // machine name, or ""
	Valid    bool
	Blocks   []BlockInfo
	DataSize int

	WAVSampleRate    uint32
	WAVBitsPerSample uint16
	WAVNumChannels   uint16
	WAVDuration      float64
}

// Inspect detects the file type from its extension and reports its contents.
// The machine is used for CAS parsing; CMD files are always parsed as TRS-80.
func Inspect(path string, m machine.Machine) (*Result, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".cas":
		return inspectCAS(path, m)
	case ".cmd":
		return inspectCMD(path)
	case ".wav":
		return inspectWAV(path)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

func inspectCAS(path string, m machine.Machine) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	analysis := m.Parse(data)

	return &Result{
		Path:     path,
		Format:   "cas",
		Machine:  m.Name(),
		Valid:    analysis.Valid,
		Blocks:   analysis.Blocks,
		DataSize: analysis.DataSize,
	}, nil
}

func inspectCMD(path string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	analysis := trs80.ParseCMD(data)

	return &Result{
		Path:     path,
		Format:   "cmd",
		Machine:  "trs80",
		Valid:    analysis.Valid,
		Blocks:   analysis.Blocks,
		DataSize: analysis.DataSize,
	}, nil
}

func inspectWAV(path string) (*Result, error) {
	w, err := wav.Inspect(path)
	if err != nil {
		return nil, fmt.Errorf("reading WAV: %w", err)
	}

	return &Result{
		Path:             path,
		Format:           "wav",
		Valid:            true,
		WAVSampleRate:    w.SampleRate,
		WAVBitsPerSample: w.BitsPerSample,
		WAVNumChannels:   w.NumChannels,
		WAVDuration:      w.Duration,
		DataSize:         w.Frames,
	}, nil
}

// String formats the inspection result for display.
func (r *Result) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "File: %s\n", r.Path)

	switch r.Format {
	case "cas":
		fmt.Fprintf(&b, "Format: CAS (Cassette Tape Image)\n")
		r.writeMachine(&b)
		r.writeBlocks(&b)
	case "cmd":
		fmt.Fprintf(&b, "Format: CMD (TRS-80 Command File)\n")
		fmt.Fprintf(&b, "Machine: TRS-80\n")
		r.writeBlocks(&b)
	case "wav":
		fmt.Fprintf(&b, "Format: WAV (Waveform Audio)\n")
		fmt.Fprintf(&b, "Sample rate: %d Hz\n", r.WAVSampleRate)
		fmt.Fprintf(&b, "Bits per sample: %d\n", r.WAVBitsPerSample)
		fmt.Fprintf(&b, "Channels: %d\n", r.WAVNumChannels)
		fmt.Fprintf(&b, "Duration: %.2f seconds\n", r.WAVDuration)
		fmt.Fprintf(&b, "Data size: %d samples\n", r.DataSize)

		if r.WAVSampleRate == wav.DefaultSampleRate && r.WAVBitsPerSample == 8 && r.WAVNumChannels == 1 {
			fmt.Fprintf(&b, "Likely contains: CAS audio data (MSX standard)\n")
		}
	default:
		fmt.Fprintf(&b, "Format: unknown\n")
	}

	return b.String()
}

func (r *Result) writeMachine(b *strings.Builder) {
	if r.Machine == "trs80" {
		fmt.Fprintf(b, "Machine: TRS-80\n")
	} else {
		fmt.Fprintf(b, "Machine: MSX\n")
	}
}

func (r *Result) writeBlocks(b *strings.Builder) {
	if r.Valid {
		fmt.Fprintf(b, "Status: valid\n")
		fmt.Fprintf(b, "Blocks: %d\n", len(r.Blocks))
		fmt.Fprintf(b, "Data size: %d bytes\n", r.DataSize)
		for i, blk := range r.Blocks {
			fmt.Fprintf(b, "  Block %d: type=%s offset=%d size=%d",
				i+1, blk.Type, blk.Offset, blk.Size)
			if blk.Filename != "" {
				fmt.Fprintf(b, " name=%q", blk.Filename)
			}
			if blk.LoadAddr != 0 || blk.ExecAddr != 0 {
				fmt.Fprintf(b, " load=0x%04X exec=0x%04X", blk.LoadAddr, blk.ExecAddr)
			}
			if len(blk.Raw) > 0 {
				hexStr := make([]string, len(blk.Raw))
				for j, v := range blk.Raw {
					hexStr[j] = fmt.Sprintf("%02X", v)
				}
				fmt.Fprintf(b, " raw=[%s]", strings.Join(hexStr, " "))
			}
			fmt.Fprintf(b, "\n")
		}
	} else {
		fmt.Fprintf(b, "Status: invalid (no valid %s structure found)\n", strings.ToUpper(r.Format))
	}
}
