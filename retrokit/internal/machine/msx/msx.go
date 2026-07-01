// Package msx implements the machine.Machine interface for MSX computers.
//
// MSX uses the Kansas City standard at 1200 or 2400 baud. Cassette images use
// an 8-byte header pattern followed by a 10-byte file-type marker (ASCII, BIN
// or BASIC) and blocks terminated by 0x1A.
package msx

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"

	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
	"github.com/carlosrabelo/retrokit/retrokit/internal/wav"
)

const (
	name = "msx"

	longPulseFreq    = 1200
	shortPulseFreq   = 2400
	longHeaderCount  = 16000
	shortHeaderCount = 4000
	defaultBaud      = 1200

	shortSilence = wav.DefaultSampleRate
	longSilence  = wav.DefaultSampleRate * 2

	thresholdSilence = 100
	thresholdHeader  = 25
)

var (
	headerPattern = []byte{0x1F, 0xA6, 0xDE, 0xBA, 0xCC, 0x13, 0x7D, 0x74}
	asciiPattern  = bytes.Repeat([]byte{0xEA}, 10)
	binPattern    = bytes.Repeat([]byte{0xD0}, 10)
	basicPattern  = bytes.Repeat([]byte{0xD3}, 10)
)

// MSX implements machine.Machine for MSX computers.
type MSX struct{}

func init() { machine.Register(&MSX{}) }

func (m *MSX) Name() string         { return name }
func (m *MSX) DisplayName() string  { return "MSX" }
func (m *MSX) BaudRates() []int     { return []int{1200, 2400} }
func (m *MSX) DefaultBaudRate() int { return defaultBaud }

// Detect reports whether data contains an MSX cassette header followed by a
// recognized file-type marker.
func (m *MSX) Detect(data []byte) bool {
	for pos := 0; pos+18 <= len(data); pos++ {
		if !bytes.Equal(data[pos:pos+8], headerPattern) {
			continue
		}
		ft := data[pos+8 : pos+18]
		if bytes.Equal(ft, asciiPattern) || bytes.Equal(ft, binPattern) || bytes.Equal(ft, basicPattern) {
			return true
		}
	}
	return false
}

// Parse analyzes the MSX CAS structure and returns its blocks.
func (m *MSX) Parse(data []byte) *machine.CASAnalysis {
	res := &machine.CASAnalysis{Blocks: []machine.Block{}}
	pos := 0

	for pos+8 <= len(data) {
		if !bytes.Equal(data[pos:pos+8], headerPattern) {
			pos++
			continue
		}
		pos += 8
		if pos+10 > len(data) {
			break
		}
		ft := data[pos : pos+10]
		pos += 10

		blockType := "unknown"
		switch {
		case bytes.Equal(ft, asciiPattern):
			blockType = "ASCII"
		case bytes.Equal(ft, binPattern):
			blockType = "BIN"
		case bytes.Equal(ft, basicPattern):
			blockType = "BASIC"
		}

		start := pos

		var loadAddr, execAddr uint16
		if blockType == "BIN" && pos+4 <= len(data) {
			loadAddr = binary.LittleEndian.Uint16(data[pos:])
			execAddr = binary.LittleEndian.Uint16(data[pos+2:])
		}

		for pos+8 <= len(data) {
			if bytes.Equal(data[pos:pos+8], headerPattern) {
				break
			}
			if data[pos] == 0x1A {
				pos++
				break
			}
			pos++
		}
		if pos+8 > len(data) {
			for pos < len(data) {
				if data[pos] == 0x1A {
					pos++
					break
				}
				pos++
			}
		}

		blockSize := pos - start
		res.Blocks = append(res.Blocks, machine.Block{
			Type:     blockType,
			Offset:   start,
			Size:     blockSize,
			LoadAddr: loadAddr,
			ExecAddr: execAddr,
		})
		res.DataSize += blockSize
	}

	res.Valid = len(res.Blocks) > 0
	return res
}

// Wrap wraps a raw binary payload into the MSX CAS format.
func (m *MSX) Wrap(payload []byte, opts machine.WrapOpts) []byte {
	if opts.Addr != 0 {
		hdr := make([]byte, 4)
		binary.LittleEndian.PutUint16(hdr[0:2], opts.Addr)
		binary.LittleEndian.PutUint16(hdr[2:4], opts.Exec)
		payload = append(hdr, payload...)
	}

	var buf bytes.Buffer
	buf.Write(headerPattern)

	switch opts.Type {
	case "ascii":
		buf.Write(asciiPattern)
	case "basic":
		buf.Write(basicPattern)
	default:
		buf.Write(binPattern)
	}

	buf.Write(payload)
	buf.WriteByte(0x1A)
	return buf.Bytes()
}

// Extract strips MSX CAS framing and returns the raw payload and blocks.
func (m *MSX) Extract(data []byte) ([]byte, []machine.Block) {
	var out bytes.Buffer
	var blocks []machine.Block
	pos := 0

	for pos+8 <= len(data) {
		if !bytes.Equal(data[pos:pos+8], headerPattern) {
			pos++
			continue
		}
		headerStart := pos
		pos += 8

		if pos+10 > len(data) {
			break
		}
		ft := data[pos : pos+10]
		pos += 10

		blockType := "unknown"
		switch {
		case bytes.Equal(ft, asciiPattern):
			blockType = "ascii"
		case bytes.Equal(ft, binPattern):
			blockType = "bin"
		case bytes.Equal(ft, basicPattern):
			blockType = "basic"
		}

		start := out.Len()
		for pos < len(data) {
			if pos+8 <= len(data) && bytes.Equal(data[pos:pos+8], headerPattern) {
				break
			}
			if data[pos] == 0x1A {
				pos++
				break
			}
			out.WriteByte(data[pos])
			pos++
		}

		blocks = append(blocks, machine.Block{
			Type:   blockType,
			Offset: headerStart,
			Size:   out.Len() - start,
		})
	}

	return out.Bytes(), blocks
}

// Encode produces PCM samples from MSX CAS bytes.
func (m *MSX) Encode(data []byte, opts machine.EncodeOpts) []byte {
	if opts.BaudRate == 0 {
		opts.BaudRate = defaultBaud
	}
	e := &encoder{baud: opts.BaudRate, gap: opts.GapTime}

	buf := new(bytes.Buffer)
	pos := 0

	for pos+8 <= len(data) {
		if bytes.Equal(data[pos:pos+8], headerPattern) {
			pos += 8
			if pos+10 > len(data) {
				break
			}
			ft := data[pos : pos+10]
			pos += 10

			switch {
			case bytes.Equal(ft, asciiPattern):
				e.writeASCII(data, &pos, buf)
			case bytes.Equal(ft, binPattern), bytes.Equal(ft, basicPattern):
				e.writeBinary(data, &pos, buf)
			default:
				e.writeUnknown(data, &pos, buf)
			}
		} else {
			pos++
		}
	}
	return buf.Bytes()
}

// Decode produces MSX CAS bytes from PCM samples.
func (m *MSX) Decode(samples []int8, sampleRate uint32, opts machine.DecodeOpts) []byte {
	if opts.Threshold == 0 {
		opts = defaultDecodeOpts()
	}
	if opts.Normalize {
		normalizeAmplitude(samples)
	}
	for i := 0; i < opts.Envelope; i++ {
		correctEnvelope(samples)
	}

	out := new(bytes.Buffer)
	index := 0
	written := 0
	headerWritten := false

	skipSilence(samples, &index, opts.Threshold)

	for index < len(samples) {
		if isSilence(samples, index, opts.Threshold) {
			fmt.Fprintf(os.Stderr, "[%.1f] skipping silence\n", float64(index)/float64(sampleRate))
			skipSilence(samples, &index, opts.Threshold)
		}

		if isHeader(samples, index, len(samples), opts.Window, opts.Threshold) {
			fmt.Fprintf(os.Stderr, "[%.1f] header detected\n", float64(index)/float64(sampleRate))
			avg := skipHeader(samples, &index, len(samples), opts.Window, opts.Threshold)

			if !headerWritten {
				if index >= len(samples) || isSilence(samples, index, opts.Threshold) {
					continue
				}
				for ; written&7 != 0; written++ {
					out.Write([]byte{0x00})
				}
				out.Write(headerPattern)
				written += 8
				headerWritten = true
			}

			fmt.Fprintf(os.Stderr, "[%.1f] data block\n", float64(index)/float64(sampleRate))

			for index < len(samples) && !isSilence(samples, index, opts.Threshold) {
				b, ok := readByte(samples, &index, len(samples), avg, opts.Window, opts.Threshold)
				if !ok {
					break
				}
				out.Write([]byte{b})
				written++
				headerWritten = false
				if b == 0x1A {
					break
				}
			}
		} else {
			for index < len(samples) && !isSilence(samples, index, opts.Threshold) {
				index++
			}
		}
	}

	fmt.Fprintf(os.Stderr, "All done...\n")
	return out.Bytes()
}

func defaultDecodeOpts() machine.DecodeOpts {
	return machine.DecodeOpts{
		Threshold: 5,
		Envelope:  2,
		Window:    1.5,
	}
}

type encoder struct {
	baud int
	gap  int
}

func (e *encoder) writeASCII(data []byte, pos *int, buf *bytes.Buffer) {
	silence := e.silenceLen()
	writeSilence(buf, silence)
	writeHeader(buf, e.baud, longHeaderCount)

	eof := e.writeDataBlock(data, pos, buf)

	for !eof && *pos < len(data) {
		if *pos+8 > len(data) || !bytes.Equal(data[*pos:*pos+8], headerPattern) {
			break
		}
		*pos += 8
		writeSilence(buf, shortSilence)
		writeHeader(buf, e.baud, shortHeaderCount)
		eof = e.writeDataBlock(data, pos, buf)
	}
	writeSilence(buf, longSilence)
}

func (e *encoder) writeBinary(data []byte, pos *int, buf *bytes.Buffer) {
	silence := e.silenceLen()
	writeSilence(buf, silence)
	writeHeader(buf, e.baud, longHeaderCount)
	eof := e.writeDataBlock(data, pos, buf)

	if eof {
		writeSilence(buf, longSilence)
		return
	}

	writeSilence(buf, shortSilence)
	writeHeader(buf, e.baud, shortHeaderCount)

	if *pos+8 <= len(data) {
		*pos += 8
	}
	e.writeDataBlock(data, pos, buf)

	writeSilence(buf, longSilence)
}

func (e *encoder) writeUnknown(data []byte, pos *int, buf *bytes.Buffer) {
	fmt.Fprintf(os.Stderr, "unknown file type: using long header\n")
	silence := e.silenceLen()
	writeSilence(buf, silence)
	writeHeader(buf, e.baud, longHeaderCount)
	e.writeDataBlock(data, pos, buf)
	writeSilence(buf, longSilence)
}

func (e *encoder) writeDataBlock(data []byte, pos *int, buf *bytes.Buffer) (eof bool) {
	for *pos+8 <= len(data) {
		if bytes.Equal(data[*pos:*pos+8], headerPattern) {
			return false
		}
		b := data[*pos]
		writeByte(buf, e.baud, b)
		if b == 0x1A {
			*pos++
			return true
		}
		*pos++
	}
	for *pos < len(data) {
		b := data[*pos]
		writeByte(buf, e.baud, b)
		if b == 0x1A {
			*pos++
			return true
		}
		*pos++
	}
	return false
}

func (e *encoder) silenceLen() int {
	if e.gap > 0 {
		return wav.DefaultSampleRate * e.gap
	}
	return longSilence
}

func writeByte(buf *bytes.Buffer, baud int, b byte) {
	writePulse(buf, baud, longPulseFreq)

	for i := 0; i < 8; i++ {
		if b&1 != 0 {
			writePulse(buf, baud, shortPulseFreq)
			writePulse(buf, baud, shortPulseFreq)
		} else {
			writePulse(buf, baud, longPulseFreq)
		}
		b >>= 1
	}

	for i := 0; i < 4; i++ {
		writePulse(buf, baud, shortPulseFreq)
	}
}

func writePulse(buf *bytes.Buffer, baud int, freq int) {
	length := float64(wav.DefaultSampleRate) / (float64(baud) * float64(freq) / 1200.0)
	scale := 2.0 * math.Pi / length

	for n := 0; n < int(length); n++ {
		sample := byte(int(math.Sin(float64(n)*scale)*127)) ^ 128
		buf.WriteByte(sample)
	}
}

func writeHeader(buf *bytes.Buffer, baud int, pulses int) {
	count := pulses * (baud / 1200)
	for i := 0; i < count; i++ {
		writePulse(buf, baud, shortPulseFreq)
	}
}

func writeSilence(buf *bytes.Buffer, samples int) {
	for i := 0; i < samples; i++ {
		buf.WriteByte(128)
	}
}

func normalizeAmplitude(samples []int8) {
	max := 0
	for _, s := range samples {
		a := int(s)
		if a < 0 {
			a = -a
		}
		if a > max {
			max = a
		}
	}
	if max == 0 {
		return
	}
	scale := 127.0 / float64(max)
	for i := range samples {
		samples[i] = int8(math.Round(float64(samples[i]) * scale))
	}
}

func correctEnvelope(samples []int8) {
	for i := 1; i < len(samples)-1; i++ {
		v := (0.5*float64(samples[i-1]) +
			1.0*float64(samples[i]) +
			2.0*float64(samples[i+1])) / 3.5
		samples[i] = int8(math.Round(v))
	}
}

func isSilence(samples []int8, index int, threshold int) bool {
	silent := 0
	for index < len(samples) && silent < thresholdSilence {
		v := int(samples[index])
		if v < 0 {
			v = -v
		}
		if v >= threshold {
			return false
		}
		silent++
		index++
	}
	return true
}

func skipSilence(samples []int8, index *int, threshold int) {
	for *index < len(samples) {
		v := int(samples[*index])
		if v < 0 {
			v = -v
		}
		if v >= threshold {
			break
		}
		(*index)++
	}
}

func getPulseWidth(samples []int8, index *int, size int, threshold int) int {
	min := 1000
	max := -1000
	pt := max

	prev := int8(0)
	if *index > 0 {
		prev = samples[(*index)-1]
	}

	width := 0
	for ; *index < size; width++ {
		if samples[*index] > prev {
			if prev == int8(min) {
				if pt-min >= threshold {
					for width > 1 {
						mid := pt - (pt-min)/2
						if int(samples[*index]) >= mid {
							break
						}
						width--
						(*index)--
					}
					return width
				}
				min = 1000
			}
			if int(samples[*index]) > max {
				max = int(samples[*index])
			}
		}

		if samples[*index] < prev {
			if prev == int8(max) {
				if max > pt {
					pt = max
				}
				max = -1000
			}
			if int(samples[*index]) < min {
				min = int(samples[*index])
			}
		}

		prev = samples[*index]
		(*index)++
	}

	return width
}

func isHeader(samples []int8, index int, size int, window float64, threshold int) bool {
	pulses := 0
	biggest := 0

	getPulseWidth(samples, &index, size, threshold)

	for index < size && pulses < thresholdHeader {
		width := getPulseWidth(samples, &index, size, threshold)
		if biggest == 0 {
			biggest = width
		}
		if float64(width) > float64(biggest)*window {
			return false
		}
		if width > biggest {
			biggest = width
		}
		pulses++
	}

	return pulses >= thresholdHeader
}

func skipHeader(samples []int8, index *int, size int, window float64, threshold int) float64 {
	count := 0
	average := 0.0
	total := 0.0

	getPulseWidth(samples, index, size, threshold)

	for *index < size {
		width := getPulseWidth(samples, index, size, threshold)

		if average > 0 && float64(width) > average*window {
			*index -= width
			return average
		}

		count++
		total += float64(width)
		average = total / float64(count)
	}

	return average
}

func readByte(samples []int8, index *int, size int, average float64, window float64, threshold int) (byte, bool) {
	if isSilence(samples, *index, threshold) {
		return 0, false
	}
	width := getPulseWidth(samples, index, size, threshold)
	if isSilence(samples, *index, threshold) || float64(width) < average*window {
		return 0, false
	}

	var value byte

	for bit := 0; bit < 8; bit++ {
		width := getPulseWidth(samples, index, size, threshold)
		if isSilence(samples, *index, threshold) {
			return 0, false
		}

		if float64(width) < average*window {
			value |= 1 << bit
			getPulseWidth(samples, index, size, threshold)
			if isSilence(samples, *index, threshold) {
				return 0, false
			}
		}
	}

	for i := 0; i < 3; i++ {
		getPulseWidth(samples, index, size, threshold)
		if isSilence(samples, *index, threshold) {
			return 0, false
		}
	}
	getPulseWidth(samples, index, size, threshold)

	return value, true
}
