// Package cas2wav converts cassette images (CAS) to WAV audio using a
// machine.Machine to perform the platform-specific encoding.
package cas2wav

import (
	"fmt"
	"os"

	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
	"github.com/carlosrabelo/retrokit/retrokit/internal/wav"
)

// Options controls a CAS to WAV conversion.
type Options struct {
	BaudRate int
	GapTime  int
	Machine  machine.Machine
}

// Convert reads a CAS file, encodes it to PCM samples using the given machine,
// and writes the result as a WAV file.
func Convert(inPath, outPath string, opts Options) error {
	if opts.Machine == nil {
		return fmt.Errorf("no machine specified")
	}

	data, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("reading input file: %w", err)
	}

	samples := opts.Machine.Encode(data, machine.EncodeOpts{
		BaudRate: opts.BaudRate,
		GapTime:  opts.GapTime,
	})

	return wav.Write(outPath, samples)
}
