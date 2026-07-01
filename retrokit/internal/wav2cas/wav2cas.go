// Package wav2cas converts WAV audio recordings to cassette images (CAS) using a
// machine.Machine to perform the platform-specific decoding.
package wav2cas

import (
	"fmt"
	"os"

	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
	"github.com/carlosrabelo/retrokit/retrokit/internal/wav"
)

// Options controls a WAV to CAS conversion.
type Options struct {
	Threshold int
	Envelope  int
	Normalize bool
	Phase     bool
	Window    float64
	Machine   machine.Machine
}

// Convert reads a WAV file, decodes it to CAS bytes using the given machine,
// and writes the result as a CAS file.
func Convert(inPath, outPath string, opts Options) error {
	if opts.Machine == nil {
		return fmt.Errorf("no machine specified")
	}

	samples, sampleRate, err := wav.Read(inPath, opts.Phase)
	if err != nil {
		return fmt.Errorf("reading WAV: %w", err)
	}

	data := opts.Machine.Decode(samples, sampleRate, machine.DecodeOpts{
		Threshold: opts.Threshold,
		Envelope:  opts.Envelope,
		Normalize: opts.Normalize,
		Window:    opts.Window,
	})

	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	return nil
}
