package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/carlosrabelo/retrokit/retrokit/internal/bin2cas"
	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
)

func runBin2Cas(m machine.Machine, args []string) {
	fs := flag.NewFlagSet(m.Name()+" bin2cas", flag.ExitOnError)
	typ := fs.String("t", "bin", "data type (bin, basic, ascii; msx only)")
	addr := fs.String("addr", "", "load address in hex (e.g. c000)")
	exec := fs.String("exec", "", "execution address in hex (e.g. c000)")
	fs.Parse(args)

	if fs.NArg() != 2 {
		fmt.Fprintf(os.Stderr, "usage: retrokit %s bin2cas [-t type] [--addr hex] [--exec hex] <in.bin> <out.cas>\n", m.Name())
		os.Exit(1)
	}

	if m.Name() == "msx" {
		switch *typ {
		case "bin", "basic", "ascii":
		default:
			fmt.Fprintf(os.Stderr, "error: type must be bin, basic, or ascii\n")
			os.Exit(1)
		}
	}

	opts := bin2cas.Options{Machine: m, Type: *typ}

	if *addr != "" {
		v, err := strconv.ParseUint(strings.TrimPrefix(*addr, "0x"), 16, 16)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid address: %s\n", *addr)
			os.Exit(1)
		}
		opts.Addr = uint16(v)
	}

	if *exec != "" {
		v, err := strconv.ParseUint(strings.TrimPrefix(*exec, "0x"), 16, 16)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid execution address: %s\n", *exec)
			os.Exit(1)
		}
		opts.Exec = uint16(v)
	}

	if err := bin2cas.Convert(fs.Arg(0), fs.Arg(1), opts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
