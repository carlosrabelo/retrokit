// Package trs80 implements the machine.Machine interface for TRS-80 Model I/III/4
// computers, plus parsing of the CMD disk-executable format.
//
// TRS-80 cassette images use a 256-byte preamble (255 zero bytes followed by a
// 0xA5 sync byte) before the payload. The audio modulation is a fixed-rate
// clock/data pulse scheme. CMD files use a tagged-block structure.
package trs80

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
	"github.com/carlosrabelo/retrokit/retrokit/internal/wav"
)

const (
	name             = "trs80"
	syncByte    byte = 0xA5
	leaderBytes      = 255
	defaultBaud      = 500
)

// TRS80 implements machine.Machine for TRS-80 computers.
type TRS80 struct{}

func init() { machine.Register(&TRS80{}) }

func (t *TRS80) Name() string         { return name }
func (t *TRS80) DisplayName() string  { return "TRS-80" }
func (t *TRS80) BaudRates() []int     { return []int{defaultBaud} }
func (t *TRS80) DefaultBaudRate() int { return defaultBaud }

// HasHeader reports whether data begins with the TRS-80 cassette preamble
// (255 zero bytes followed by the 0xA5 sync byte).
func HasHeader(data []byte) bool {
	return hasHeader(data)
}

func hasHeader(data []byte) bool {
	if len(data) < leaderBytes+1 {
		return false
	}
	for i := 0; i < leaderBytes; i++ {
		if data[i] != 0 {
			return false
		}
	}
	return data[leaderBytes] == syncByte
}

// Detect reports whether data begins with the TRS-80 cassette preamble.
func (t *TRS80) Detect(data []byte) bool {
	return hasHeader(data)
}

// Parse analyzes the TRS-80 CAS structure, interpreting the payload as SYSTEM
// (3BN), BASIC, or raw data.
func (t *TRS80) Parse(data []byte) *machine.CASAnalysis {
	if len(data) < leaderBytes+1 || !hasHeader(data) {
		return &machine.CASAnalysis{Blocks: []machine.Block{}}
	}
	return cassetteToAnalysis(data)
}

// Wrap wraps a raw binary payload into the TRS-80 CAS format.
func (t *TRS80) Wrap(payload []byte, opts machine.WrapOpts) []byte {
	if opts.Addr != 0 {
		hdr := make([]byte, 4)
		binary.LittleEndian.PutUint16(hdr[0:2], opts.Addr)
		binary.LittleEndian.PutUint16(hdr[2:4], opts.Exec)
		payload = append(hdr, payload...)
	}

	var buf bytes.Buffer
	buf.Write(bytes.Repeat([]byte{0}, leaderBytes))
	buf.WriteByte(syncByte)
	buf.Write(payload)
	return buf.Bytes()
}

// Extract strips the TRS-80 preamble and returns the raw payload.
func (t *TRS80) Extract(data []byte) ([]byte, []machine.Block) {
	if len(data) < leaderBytes+1 || !hasHeader(data) {
		return data, nil
	}
	payload := data[leaderBytes+1:]

	blockType := "raw"
	if ParseSystem(payload) != nil {
		blockType = "system"
	} else if IsBasicPayload(payload) {
		blockType = "basic"
	}

	return payload, []machine.Block{{Type: blockType, Offset: leaderBytes + 1, Size: len(payload)}}
}

// Encode produces PCM samples from TRS-80 CAS bytes.
func (t *TRS80) Encode(data []byte, opts machine.EncodeOpts) []byte {
	buf := new(bytes.Buffer)

	writeSilence(buf, wav.DefaultSampleRate/2)

	toEncode := make([]byte, 0, leaderBytes+1+len(data))
	toEncode = append(toEncode, bytes.Repeat([]byte{0}, leaderBytes)...)
	toEncode = append(toEncode, syncByte)
	toEncode = append(toEncode, data...)

	for _, b := range toEncode {
		writeTrs80Byte(buf, b)
	}

	writeSilence(buf, wav.DefaultSampleRate/2)

	return buf.Bytes()
}

// Decode produces TRS-80 CAS bytes from PCM samples.
func (t *TRS80) Decode(samples []int8, sampleRate uint32, opts machine.DecodeOpts) []byte {
	if opts.Threshold == 0 {
		opts = defaultDecodeOpts()
	}

	halfBit := int(math.Round(float64(sampleRate) * 0.001))

	index := 0
	skipSilence(samples, &index, opts.Threshold)

	var decoded bytes.Buffer
	var currentByte byte
	bitCount := 0

	for index < len(samples) {
		pulseStart := -1
		for i := index; i < len(samples); i++ {
			v := int(samples[i])
			if v < 0 {
				v = -v
			}
			if v >= opts.Threshold {
				if i == 0 {
					pulseStart = 0
				} else {
					prev := int(samples[i-1])
					if prev < 0 {
						prev = -prev
					}
					if prev < opts.Threshold {
						pulseStart = i
					}
				}
				if pulseStart >= 0 {
					index = i
					break
				}
			}
		}
		if pulseStart < 0 {
			break
		}

		dataStart := pulseStart + halfBit

		bit := byte(0)
		checkEnd := dataStart + halfBit
		if checkEnd > len(samples) {
			checkEnd = len(samples)
		}
		for i := dataStart; i < checkEnd; i++ {
			v := int(samples[i])
			if v < 0 {
				v = -v
			}
			if v >= opts.Threshold {
				bit = 1
				break
			}
		}

		currentByte = (currentByte << 1) | bit
		bitCount++

		if bitCount == 8 {
			decoded.WriteByte(currentByte)
			currentByte = 0
			bitCount = 0
		}

		index = pulseStart + halfBit*2
	}

	if bitCount > 0 {
		currentByte <<= 8 - bitCount
		decoded.WriteByte(currentByte)
	}

	data := decoded.Bytes()
	if len(data) >= leaderBytes+1 && hasHeader(data[:leaderBytes+1]) {
		data = data[leaderBytes+1:]
	}

	return data
}

func defaultDecodeOpts() machine.DecodeOpts {
	return machine.DecodeOpts{
		Threshold: 5,
		Window:    1.5,
	}
}

func writeTrs80Byte(buf *bytes.Buffer, b byte) {
	for i := 7; i >= 0; i-- {
		writeTrs80Pulse(buf)
		if b&(1<<i) != 0 {
			writeTrs80Pulse(buf)
		} else {
			writeTrs80HalfSilence(buf)
		}
	}
}

func writeTrs80Pulse(buf *bytes.Buffer) {
	halfBit := int(math.Round(float64(wav.DefaultSampleRate) * 0.001))
	offset := halfBit / 4
	for i := 0; i < halfBit; i++ {
		var sample int
		rel := i - offset
		if rel >= 0 && rel < halfBit/2 {
			sample = int(math.Sin(2*math.Pi*float64(rel)/(float64(halfBit)/2)) * 127)
		}
		buf.WriteByte(byte(sample) ^ 128)
	}
}

func writeTrs80HalfSilence(buf *bytes.Buffer) {
	halfBit := int(math.Round(float64(wav.DefaultSampleRate) * 0.001))
	writeSilence(buf, halfBit)
}

func writeSilence(buf *bytes.Buffer, samples int) {
	for i := 0; i < samples; i++ {
		buf.WriteByte(128)
	}
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
