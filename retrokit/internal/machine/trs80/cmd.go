package trs80

import (
	"encoding/binary"
	"fmt"

	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
)

// CMD (disk-executable) block tags.
const (
	cmdLoadBlock    byte = 0x01 // [loadAddrLo, loadAddrHi, ...code]
	cmdTransferAddr byte = 0x02 // [entryLo, entryHi]
	cmdEOF          byte = 0x03 // end of file
	cmdModuleName   byte = 0x05 // filename ASCII
	cmdYankedLoad   byte = 0x10 // like 0x01
	cmdMaxType      byte = 0x1F // tags above this are invalid / end-of-data
)

// ParseCMD analyzes a TRS-80 CMD (disk-executable) file and returns its blocks.
//
// The CMD format is a sequence of tagged blocks:
//
//	[tag:1][length:1][data:length]   (length byte 0 means 256 bytes)
//
// If the file is actually a cassette image (256-byte preamble), it is parsed as
// a cassette block instead, mirroring historical behaviour.
func ParseCMD(data []byte) *machine.CASAnalysis {
	res := &machine.CASAnalysis{Blocks: []machine.Block{}}

	if len(data) >= leaderBytes+1 && hasHeader(data) {
		return cassetteToAnalysis(data)
	}

	pos := 0
	for pos < len(data) {
		tag := data[pos]
		if tag > cmdMaxType {
			break
		}
		blockStart := pos
		pos++

		if pos >= len(data) {
			break
		}
		length := int(data[pos])
		pos++
		if length == 0 {
			length = maxSystemChunkData
		}

		dataEnd := pos + length
		if dataEnd > len(data) {
			dataEnd = len(data)
		}
		payload := data[pos:dataEnd]
		pos = dataEnd

		blk := machine.Block{
			Offset: blockStart,
			Size:   1 + 1 + len(payload),
			Raw:    data[blockStart : blockStart+2],
		}

		switch tag {
		case cmdLoadBlock, cmdYankedLoad:
			blk.Type = fmt.Sprintf("load_0x%02X", tag)
			if len(payload) >= 2 {
				blk.LoadAddr = binary.LittleEndian.Uint16(payload[0:2])
			}

		case cmdTransferAddr:
			blk.Type = "exec"
			if len(payload) >= 2 {
				blk.ExecAddr = binary.LittleEndian.Uint16(payload[0:2])
			}

		case cmdEOF:
			blk.Type = "end"

		case cmdModuleName:
			blk.Type = "name"
			blk.Filename = string(payload)

		default:
			blk.Type = fmt.Sprintf("tag_0x%02X", tag)
		}

		res.Blocks = append(res.Blocks, blk)
		res.DataSize += blk.Size
	}

	res.Valid = len(res.Blocks) > 0
	return res
}

// cassetteToAnalysis builds a CASAnalysis from a TRS-80 cassette image,
// interpreting the payload as SYSTEM, BASIC, or raw.
func cassetteToAnalysis(data []byte) *machine.CASAnalysis {
	headerSize := leaderBytes + 1
	payload := data[headerSize:]
	res := &machine.CASAnalysis{
		Valid:    true,
		DataSize: len(payload),
		Blocks:   []machine.Block{},
	}

	if prog := ParseSystem(payload); prog != nil {
		for _, ch := range prog.Chunks {
			res.Blocks = append(res.Blocks, machine.Block{
				Type:     "system",
				Offset:   headerSize,
				Size:     len(ch.Data),
				LoadAddr: ch.LoadAddr,
			})
		}
		if len(res.Blocks) == 0 {
			res.Blocks = append(res.Blocks, machine.Block{
				Type:     "system",
				Offset:   headerSize,
				Size:     0,
				Filename: prog.Filename,
			})
		}
		res.Blocks[0].Filename = prog.Filename
		res.Blocks[len(res.Blocks)-1].ExecAddr = prog.EntryAddr
		return res
	}

	if IsBasicPayload(payload) {
		res.Blocks = append(res.Blocks, machine.Block{
			Type:   "basic",
			Offset: headerSize,
			Size:   len(payload),
		})
		return res
	}

	res.Blocks = append(res.Blocks, machine.Block{
		Type:   "raw",
		Offset: headerSize,
		Size:   len(payload),
	})
	if len(payload) >= 4 {
		res.Blocks[0].LoadAddr = binary.LittleEndian.Uint16(payload[0:2])
		res.Blocks[0].ExecAddr = binary.LittleEndian.Uint16(payload[2:4])
	}
	return res
}
