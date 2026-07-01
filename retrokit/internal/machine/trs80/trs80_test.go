package trs80

import (
	"bytes"
	"testing"

	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
)

func TestParseSystemRoundTrip(t *testing.T) {
	prog := &SystemProgram{
		Filename:  "TEST",
		EntryAddr: 0x4280,
		Chunks: []SystemChunk{
			{LoadAddr: 0x4280, Data: []byte{0x3E, 0x55, 0xD3, 0x88, 0x76}},
		},
	}

	encoded := prog.Encode()

	got := ParseSystem(encoded)
	if got == nil {
		t.Fatal("ParseSystem returned nil for valid SYSTEM data")
	}
	if got.Filename != "TEST" {
		t.Errorf("filename: got %q, want %q", got.Filename, "TEST")
	}
	if got.EntryAddr != 0x4280 {
		t.Errorf("entry addr: got 0x%04X, want 0x4280", got.EntryAddr)
	}
	if len(got.Chunks) != 1 {
		t.Fatalf("chunks: got %d, want 1", len(got.Chunks))
	}
	if got.Chunks[0].LoadAddr != 0x4280 {
		t.Errorf("chunk load addr: got 0x%04X", got.Chunks[0].LoadAddr)
	}
	if !bytes.Equal(got.Chunks[0].Data, prog.Chunks[0].Data) {
		t.Errorf("chunk data: got % X, want % X", got.Chunks[0].Data, prog.Chunks[0].Data)
	}
}

func TestParseSystemRejectsNonSystem(t *testing.T) {
	if ParseSystem([]byte{0xD3, 0xD3, 0xD3}) != nil {
		t.Error("expected nil for non-SYSTEM data")
	}
	if ParseSystem([]byte{0x3E, 0x55}) != nil {
		t.Error("expected nil for non-SYSTEM data")
	}
}

func TestSystemChunkSplitting(t *testing.T) {
	// A payload larger than 256 bytes should be split into multiple chunks.
	bigData := make([]byte, 300)
	for i := range bigData {
		bigData[i] = byte(i)
	}
	prog := &SystemProgram{
		Filename:  "BIG",
		EntryAddr: 0x8000,
		Chunks: []SystemChunk{
			{LoadAddr: 0x8000, Data: bigData},
		},
	}

	encoded := prog.Encode()
	got := ParseSystem(encoded)
	if got == nil {
		t.Fatal("ParseSystem returned nil")
	}
	if len(got.Chunks) != 2 {
		t.Fatalf("expected 2 chunks (split at 256), got %d", len(got.Chunks))
	}
	// First chunk: 256 bytes at 0x8000
	if len(got.Chunks[0].Data) != 256 {
		t.Errorf("first chunk size: got %d, want 256", len(got.Chunks[0].Data))
	}
	if got.Chunks[0].LoadAddr != 0x8000 {
		t.Errorf("first chunk addr: got 0x%04X", got.Chunks[0].LoadAddr)
	}
	// Second chunk: 44 bytes at 0x8100
	if len(got.Chunks[1].Data) != 44 {
		t.Errorf("second chunk size: got %d, want 44", len(got.Chunks[1].Data))
	}
	if got.Chunks[1].LoadAddr != 0x8100 {
		t.Errorf("second chunk addr: got 0x%04X, want 0x8100", got.Chunks[1].LoadAddr)
	}
}

func TestSystemChecksum(t *testing.T) {
	addr := uint16(0x4280)
	data := []byte{0x3E, 0x55, 0xD3}
	got := computeChecksum(addr, data)
	// (0x42 + 0x80 + 0x3E + 0x55 + 0xD3) & 0xFF
	want := byte((0x42 + 0x80 + 0x3E + 0x55 + 0xD3) & 0xFF)
	if got != want {
		t.Errorf("checksum: got 0x%02X, want 0x%02X", got, want)
	}
}

func TestIsBasicPayload(t *testing.T) {
	if !IsBasicPayload([]byte{0xD3, 0xD3, 0xD3, 0x65}) {
		t.Error("expected BASIC payload to be detected")
	}
	if IsBasicPayload([]byte{0x55, 0x54, 0x45}) {
		t.Error("non-BASIC should not be detected")
	}
}

func TestParseCMDLoadAndExec(t *testing.T) {
	// Build a minimal CMD: one LOAD block + TRANSFER + EOF
	var buf bytes.Buffer
	// LOAD: tag=0x01, len=4, data=[addrLo, addrHi, 0xAA, 0xBB]
	buf.WriteByte(cmdLoadBlock)
	buf.WriteByte(4)
	buf.WriteByte(0x80) // addr lo
	buf.WriteByte(0x42) // addr hi
	buf.WriteByte(0xAA)
	buf.WriteByte(0xBB)
	// TRANSFER: tag=0x02, len=2, data=[entryLo, entryHi]
	buf.WriteByte(cmdTransferAddr)
	buf.WriteByte(2)
	buf.WriteByte(0x80)
	buf.WriteByte(0x42)
	// EOF: tag=0x03, len=0
	buf.WriteByte(cmdEOF)
	buf.WriteByte(0)

	res := ParseCMD(buf.Bytes())
	if !res.Valid {
		t.Fatal("expected valid CMD")
	}
	if len(res.Blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(res.Blocks))
	}

	loadBlk := res.Blocks[0]
	if loadBlk.Type != "load_0x01" {
		t.Errorf("block 0 type: got %q", loadBlk.Type)
	}
	if loadBlk.LoadAddr != 0x4280 {
		t.Errorf("block 0 load addr: got 0x%04X", loadBlk.LoadAddr)
	}

	execBlk := res.Blocks[1]
	if execBlk.Type != "exec" {
		t.Errorf("block 1 type: got %q", execBlk.Type)
	}
	if execBlk.ExecAddr != 0x4280 {
		t.Errorf("block 1 exec addr: got 0x%04X", execBlk.ExecAddr)
	}

	if res.Blocks[2].Type != "end" {
		t.Errorf("block 2 type: got %q", res.Blocks[2].Type)
	}
}

func TestParseCMDLengthZeroMeans256(t *testing.T) {
	// A LOAD block with length byte 0 should read 256 bytes of data.
	var buf bytes.Buffer
	buf.WriteByte(cmdLoadBlock)
	buf.WriteByte(0) // length = 256
	payload := make([]byte, 256)
	payload[0] = 0x80 // addr lo
	payload[1] = 0x42 // addr hi
	buf.Write(payload)

	res := ParseCMD(buf.Bytes())
	if len(res.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(res.Blocks))
	}
	if res.Blocks[0].Size != 2+256 {
		t.Errorf("block size: got %d, want %d", res.Blocks[0].Size, 2+256)
	}
}

func TestCassetteToAnalysisRaw(t *testing.T) {
	// A raw payload (no SYSTEM/BASIC markers) should produce a "raw" block.
	data := buildCassette([]byte{0xC3, 0x7B, 0x0D, 0x00, 0x00, 0x00})
	res := cassetteToAnalysis(data)
	if !res.Valid {
		t.Fatal("expected valid")
	}
	if len(res.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(res.Blocks))
	}
	if res.Blocks[0].Type != "raw" {
		t.Errorf("block type: got %q, want raw", res.Blocks[0].Type)
	}
}

func TestCassetteToAnalysisSystem(t *testing.T) {
	prog := &SystemProgram{Filename: "PROG", EntryAddr: 0x4280, Chunks: []SystemChunk{
		{LoadAddr: 0x4280, Data: []byte{0x3E, 0x55, 0x76}},
	}}
	data := buildCassette(prog.Encode())
	res := cassetteToAnalysis(data)
	if !res.Valid {
		t.Fatal("expected valid")
	}
	if res.Blocks[0].Type != "system" {
		t.Errorf("block type: got %q, want system", res.Blocks[0].Type)
	}
	if res.Blocks[0].Filename != "PROG" {
		t.Errorf("filename: got %q", res.Blocks[0].Filename)
	}
}

func TestCassetteToAnalysisBasic(t *testing.T) {
	payload := []byte{0xD3, 0xD3, 0xD3, 0x65, 0x00, 0x00}
	data := buildCassette(payload)
	res := cassetteToAnalysis(data)
	if res.Blocks[0].Type != "basic" {
		t.Errorf("block type: got %q, want basic", res.Blocks[0].Type)
	}
}

// buildCassette wraps a payload with the TRS-80 cassette preamble.
func buildCassette(payload []byte) []byte {
	buf := make([]byte, leaderBytes+1+len(payload))
	// leaderBytes zeros already zero
	buf[leaderBytes] = syncByte
	copy(buf[leaderBytes+1:], payload)
	return buf
}

// Ensure machine.Block is used so the import is not flagged.
var _ machine.Block
