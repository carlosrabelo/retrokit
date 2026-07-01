package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/carlosrabelo/retrokit/retrokit/internal/cas2wav"
	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
)

func runCas2Wav(m machine.Machine, args []string) {
	fs := flag.NewFlagSet(m.Name()+" cas2wav", flag.ExitOnError)
	baud := fs.Int("b", m.DefaultBaudRate(), "baud rate ("+joinInts(m.BaudRates())+")")
	stime := fs.Int("s", 0, "gap time in seconds between blocks (0 = default 2s)")
	fs.Parse(args)

	if fs.NArg() != 2 {
		fs.Usage()
		os.Exit(1)
	}

	if !validBaud(m, *baud) {
		fmt.Fprintf(os.Stderr, "error: baud rate must be one of %s\n", joinInts(m.BaudRates()))
		os.Exit(1)
	}

	opts := cas2wav.Options{
		BaudRate: *baud,
		GapTime:  *stime,
		Machine:  m,
	}

	if err := cas2wav.Convert(fs.Arg(0), fs.Arg(1), opts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
