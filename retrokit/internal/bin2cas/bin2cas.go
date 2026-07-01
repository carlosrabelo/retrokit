// Package bin2cas wraps a raw binary payload into a cassette image (CAS) using a
// machine.Machine to produce the platform-specific framing.
package bin2cas

import (
	"fmt"
	"os"

	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
)

// Options controls a binary to CAS conversion.
type Options struct {
	Machine machine.Machine
	Type    string
	Addr    uint16
	Exec    uint16
}

// Convert reads a binary file, wraps it in the given machine's CAS format, and
// writes the result as a CAS file.
func Convert(inPath, outPath string, opts Options) error {
	if opts.Machine == nil {
		return fmt.Errorf("no machine specified")
	}

	payload, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("reading input file: %w", err)
	}

	data := opts.Machine.Wrap(payload, machine.WrapOpts{
		Type: opts.Type,
		Addr: opts.Addr,
		Exec: opts.Exec,
	})

	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}
	return nil
}

// ValidTypes returns the wrap payload types supported across machines.
func ValidTypes() []string {
	return []string{"bin", "basic", "ascii"}
}
