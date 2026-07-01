// Package cas2bin extracts the raw binary payload from a cassette image (CAS)
// using a machine.Machine provided by the caller to perform the extraction.
package cas2bin

import (
	"fmt"
	"os"

	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
)

// BlockInfo describes an extracted block.
type BlockInfo = machine.Block

// Result holds the outcome of a CAS to binary extraction.
type Result struct {
	Machine string
	Blocks  []BlockInfo
}

// Convert reads a CAS file, strips its framing using the given machine, and
// writes the raw payload as a binary file.
func Convert(inPath, outPath string, m machine.Machine) (*Result, error) {
	data, err := os.ReadFile(inPath)
	if err != nil {
		return nil, fmt.Errorf("reading input file: %w", err)
	}

	payload, blocks := m.Extract(data)
	if len(blocks) == 0 {
		return nil, fmt.Errorf("no valid CAS blocks found")
	}

	if err := os.WriteFile(outPath, payload, 0644); err != nil {
		return nil, fmt.Errorf("writing output file: %w", err)
	}
	return &Result{Machine: m.Name(), Blocks: blocks}, nil
}
