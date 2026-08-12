# retrokit

Command-line tools for converting retro computer cassette tape images (CAS) to audio (WAV) and back, built on an extensible per-platform architecture.

## Highlights

- Convert CAS tape images to WAV audio files for playback on real retro computers
- Decode WAV recordings back to CAS tape images for archival and emulation
- Inspect CAS, CMD, and WAV files to display detailed structure and metadata
- Wrap raw binaries into CAS, and extract binaries back out of CAS
- Pluggable `Machine` interface: each platform encapsulates its own modulation, block layout, and file types
- Currently supports **MSX** and **TRS-80**; designed to grow to ZX81, Apple II, Commodore, and more

## Prerequisites

- **Go 1.26+** — required to build from source; [download](https://go.dev/dl/)

## Installation

### Build from Source

```bash
git clone https://github.com/carlosrabelo/retrokit.git
cd retrokit
make build
```

Install to `~/.local/bin` (default), or system-wide to `/usr/local/bin` (sudo only for the copy):

```bash
make install
make install-system
make uninstall
make uninstall-system
```

## Usage

The machine is always required as a prefix for all commands; only `machines` and `help` are top-level.

```
retrokit <machine> <command> [options] <in> <out>
retrokit <machine> info <file>
retrokit machines
```

### cas2wav — Convert CAS to WAV

```bash
./bin/retrokit msx cas2wav -b 1200 program.cas program.wav
./bin/retrokit trs80 cas2wav program.cas program.wav
```

### wav2cas — Convert WAV to CAS

```bash
./bin/retrokit msx wav2cas -t 5 recording.wav output.cas
```

### bin2cas — Wrap a binary into CAS

```bash
./bin/retrokit msx bin2cas -t bin --addr C000 --exec C000 program.bin program.cas
```

### cas2bin — Extract a binary from CAS

```bash
./bin/retrokit msx cas2bin program.cas program.bin
```

### info — Inspect a file

```bash
./bin/retrokit msx info program.cas
./bin/retrokit trs80 info program.cmd
./bin/retrokit msx info recording.wav
```

### machines — List supported machines

```bash
./bin/retrokit machines
```

## Supported Machines

| Machine | Cassette format | Baud rates | File types |
| --- | --- | --- | --- |
| MSX | Kansas City (8-byte header + type marker) | 1200, 2400 | ASCII, BIN, BASIC |
| TRS-80 | 256-byte preamble + 0xA5 sync | 500 | data (CMD disk format also inspectable) |

## Project Layout

```
retrokit/                  # Go source (cmd/internal)
    cmd/retrokit/          # CLI entry point
    internal/
        wav/               # shared WAV read/write/inspect
        machine/           # Machine interface + registry
            msx/           # MSX implementation
            trs80/         # TRS-80 implementation (+ CMD parser)
            all/           # registers every machine via init()
        cas2wav/           # CAS to WAV (delegates to Machine)
        wav2cas/           # WAV to CAS (delegates to Machine)
        bin2cas/           # binary to CAS (delegates to Machine)
        cas2bin/           # CAS to binary (delegates to Machine)
        info/              # file inspection (delegates to Machine)
bin/                       # Compiled binaries (git-ignored)
.make/                     # Build and install scripts
```

### Adding a new platform

Implement the `machine.Machine` interface in a new package under `retrokit/internal/machine/<name>/`, register it in `init()`, and add a blank import to `retrokit/internal/machine/all/all.go`. No command package needs to change.

## Development

```bash
make build             # Compile binary to bin/retrokit
make test              # Run all tests
make quality           # Format, vet, and lint
make install           # Install binary to ~/.local/bin
make install-system    # Install binary to /usr/local/bin
make uninstall         # Remove from ~/.local/bin
make uninstall-system  # Remove from /usr/local/bin
```

## License

This project is licensed under the MIT License — see [LICENSE](LICENSE) for details.
