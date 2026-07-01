package main

import (
	"fmt"
	"os"

	"github.com/carlosrabelo/retrokit/retrokit/internal/cas2bin"
	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
)

func runCas2Bin(m machine.Machine, args []string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: retrokit %s cas2bin <in.cas> <out.bin>\n", m.Name())
		os.Exit(1)
	}

	res, err := cas2bin.Convert(args[0], args[1], m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "machine: %s\n", res.Machine)
	for _, b := range res.Blocks {
		fmt.Fprintf(os.Stderr, "  block: type=%s offset=%d size=%d\n", b.Type, b.Offset, b.Size)
	}
}
