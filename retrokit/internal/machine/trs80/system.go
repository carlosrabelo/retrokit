package trs80

import (
	"bytes"
	"encoding/binary"
	"strings"
)

// SYSTEM (3BN) markers, compatible with the trs80-tool format.
const (
	fileHeaderByte byte = 0x55 // FILE_HEADER
	dataHeaderByte byte = 0x3C // DATA_HEADER
	endMarker      byte = 0x78 // END_OF_FILE_MARKER
	basicHeader    byte = 0xD3 // BASIC_TAPE_HEADER_BYTE
)

// Max chunk data size for a SYSTEM data block.
const maxSystemChunkData = 256

// SystemChunk is a single data chunk within a SYSTEM (3BN) program.
type SystemChunk struct {
	LoadAddr uint16
	Data     []byte
}

// SystemProgram represents a TRS-80 SYSTEM (3BN) cassette payload.
type SystemProgram struct {
	Filename  string // up to 6 chars, space-padded
	Chunks    []SystemChunk
	EntryAddr uint16
}

// ParseSystem decodes a SYSTEM (3BN) payload. Returns nil if data does not
// begin with the FILE_HEADER (0x55).
func ParseSystem(data []byte) *SystemProgram {
	if len(data) < 1 || data[0] != fileHeaderByte {
		return nil
	}

	pos := 1
	if pos+6 > len(data) {
		return nil
	}
	filename := strings.TrimRight(string(data[pos:pos+6]), " ")
	pos += 6

	prog := &SystemProgram{Filename: filename}

	for pos < len(data) {
		marker := data[pos]
		pos++

		if marker == endMarker {
			if pos+2 <= len(data) {
				prog.EntryAddr = binary.LittleEndian.Uint16(data[pos:])
			}
			return prog
		}

		if marker != dataHeaderByte {
			return prog
		}

		if pos+3 > len(data) {
			return prog
		}
		length := int(data[pos])
		pos++
		if length == 0 {
			length = maxSystemChunkData
		}
		loadAddr := binary.LittleEndian.Uint16(data[pos:])
		pos += 2

		dataEnd := pos + length
		if dataEnd > len(data) {
			dataEnd = len(data)
		}
		chunkData := make([]byte, dataEnd-pos)
		copy(chunkData, data[pos:dataEnd])
		pos = dataEnd

		if pos < len(data) {
			pos++ // checksum byte
		}

		prog.Chunks = append(prog.Chunks, SystemChunk{
			LoadAddr: loadAddr,
			Data:     chunkData,
		})
	}

	return prog
}

// Encode serializes a SystemProgram to SYSTEM (3BN) bytes, splitting data into
// chunks of at most 256 bytes and computing checksums.
func (p *SystemProgram) Encode() []byte {
	var buf bytes.Buffer

	buf.WriteByte(fileHeaderByte)
	buf.WriteString(padFilename(p.Filename))

	for _, chunk := range p.Chunks {
		offset := 0
		for offset < len(chunk.Data) {
			end := offset + maxSystemChunkData
			if end > len(chunk.Data) {
				end = len(chunk.Data)
			}
			piece := chunk.Data[offset:end]
			addr := chunk.LoadAddr + uint16(offset)

			lengthField := len(piece) & 0xFF // 0 means 256
			buf.WriteByte(dataHeaderByte)
			buf.WriteByte(byte(lengthField))
			buf.WriteByte(byte(addr & 0xFF))
			buf.WriteByte(byte(addr >> 8))
			buf.Write(piece)

			sum := int(addr>>8) + int(addr&0xFF)
			for _, b := range piece {
				sum += int(b)
			}
			buf.WriteByte(byte(sum & 0xFF))

			offset = end
		}
	}

	buf.WriteByte(endMarker)
	buf.WriteByte(byte(p.EntryAddr & 0xFF))
	buf.WriteByte(byte(p.EntryAddr >> 8))

	return buf.Bytes()
}

// padFilename upper-cases, truncates and right-pads the filename to 6 chars.
func padFilename(name string) string {
	name = strings.ToUpper(strings.TrimSpace(name))
	if len(name) > 6 {
		name = name[:6]
	}
	for len(name) < 6 {
		name += " "
	}
	return name
}

// computeChecksum computes the SYSTEM chunk checksum for a data piece and its
// load address: (addrHi + addrLo + sum(data)) & 0xFF.
func computeChecksum(addr uint16, data []byte) byte {
	sum := int(addr>>8) + int(addr&0xFF)
	for _, b := range data {
		sum += int(b)
	}
	return byte(sum & 0xFF)
}

// IsBasicPayload reports whether data looks like a TRS-80 BASIC cassette
// payload (starts with three 0xD3 bytes).
func IsBasicPayload(data []byte) bool {
	return len(data) >= 3 && data[0] == basicHeader && data[1] == basicHeader && data[2] == basicHeader
}
