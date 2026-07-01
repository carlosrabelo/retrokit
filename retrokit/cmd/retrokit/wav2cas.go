package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
	"github.com/carlosrabelo/retrokit/retrokit/internal/wav2cas"
)

func runWav2Cas(m machine.Machine, args []string) {
	fs := flag.NewFlagSet(m.Name()+" wav2cas", flag.ExitOnError)
	normalize := fs.Bool("n", false, "normalize amplitude level")
	phase := fs.Bool("p", true, "phase shift signal (default true)")
	window := fs.Float64("w", 1.5, "window factor")
	envelope := fs.Int("e", 2, "level of envelope correction")
	threshold := fs.Int("t", 5, "threshold factor")
	fs.Parse(args)

	if fs.NArg() != 2 {
		fs.Usage()
		os.Exit(1)
	}

	opts := wav2cas.Options{
		Threshold: *threshold,
		Envelope:  *envelope,
		Normalize: *normalize,
		Phase:     *phase,
		Window:    *window,
		Machine:   m,
	}

	if err := wav2cas.Convert(fs.Arg(0), fs.Arg(1), opts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
