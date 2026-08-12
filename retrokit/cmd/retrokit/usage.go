package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
)

func runMachineCommand(m machine.Machine, args []string) {
	if len(args) < 1 {
		machineUsage(m)
		os.Exit(1)
	}

	switch args[0] {
	case "cas2wav":
		runCas2Wav(m, args[1:])
	case "wav2cas":
		runWav2Cas(m, args[1:])
	case "bin2cas":
		runBin2Cas(m, args[1:])
	case "cas2bin":
		runCas2Bin(m, args[1:])
	case "info":
		runInfo(m, args[1:])
	case "help", "-h", "--help":
		machineUsage(m)
	default:
		fmt.Fprintf(os.Stderr, "unknown command for %s: %s\n\n", m.Name(), args[0])
		machineUsage(m)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `usage: retrokit <machine> <command> [options] <in> <out>

machines (use as a prefix for machine-specific commands):
%s

commands:
  <machine> cas2wav [opts] <in.cas> <out.wav>   convert CAS to WAV
  <machine> wav2cas [opts] <in.wav> <out.cas>   convert WAV to CAS
  <machine> bin2cas [opts] <in.bin> <out.cas>   wrap binary in CAS
  <machine> cas2bin <in.cas> <out.bin>          extract binary from CAS
  <machine> info <file>                         show file information

  machines                                      list supported machines
  help                                          show this help

examples:
  retrokit msx cas2wav -b 2400 prog.cas prog.wav
  retrokit trs80 wav2cas -t 5 recording.wav out.cas
  retrokit msx info prog.cas

`, machinesBlock())
}

func machineUsage(m machine.Machine) {
	fmt.Fprintf(os.Stderr, `usage: retrokit %s <command> [options] <in> <out>

commands:
  cas2wav [opts] <in.cas> <out.wav>   convert CAS to WAV
  wav2cas [opts] <in.wav> <out.cas>   convert WAV to CAS
  bin2cas [opts] <in.bin> <out.cas>   wrap binary in CAS
  cas2bin <in.cas> <out.bin>          extract binary from CAS
  info <file>                         show file information

machine: %s (%s), baud rates: %s

`, m.Name(), m.DisplayName(), m.Name(), joinInts(m.BaudRates()))
}

func listMachines() {
	for _, name := range machine.Names() {
		m, _ := machine.Lookup(name)
		fmt.Printf("%-10s %s (baud: %s)\n", m.Name(), m.DisplayName(), joinInts(m.BaudRates()))
	}
}

func machinesBlock() string {
	var b strings.Builder
	for _, name := range machine.Names() {
		m, _ := machine.Lookup(name)
		fmt.Fprintf(&b, "  %-10s %s\n", m.Name(), m.DisplayName())
	}
	return b.String()
}

func joinInts(xs []int) string {
	parts := make([]string, len(xs))
	for i, v := range xs {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, ", ")
}

func validBaud(m machine.Machine, baud int) bool {
	for _, b := range m.BaudRates() {
		if b == baud {
			return true
		}
	}
	return false
}
