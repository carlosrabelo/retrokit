package cas2wav

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
	_ "github.com/carlosrabelo/retrokit/retrokit/internal/machine/msx"
	"github.com/carlosrabelo/retrokit/retrokit/internal/wav"
)

func makeTestCAS(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer

	// Header + BIN type + small data, then a second block.
	buf.Write([]byte{0x1F, 0xA6, 0xDE, 0xBA, 0xCC, 0x13, 0x7D, 0x74})
	buf.Write(bytes.Repeat([]byte{0xD0}, 10))
	buf.Write([]byte{0x01, 0x02, 0x03, 0x04, 0x1A})

	buf.Write([]byte{0x1F, 0xA6, 0xDE, 0xBA, 0xCC, 0x13, 0x7D, 0x74})
	buf.Write(bytes.Repeat([]byte{0xD0}, 10))
	buf.Write([]byte{0x10, 0x20, 0x30, 0x1A})

	return buf.Bytes()
}

func TestConvert(t *testing.T) {
	dir := t.TempDir()
	casPath := filepath.Join(dir, "test.cas")
	wavPath := filepath.Join(dir, "test.wav")

	casData := makeTestCAS(t)
	if err := os.WriteFile(casPath, casData, 0644); err != nil {
		t.Fatal(err)
	}

	opts := Options{BaudRate: 1200, Machine: machine.MustLookup("msx")}
	if err := Convert(casPath, wavPath, opts); err != nil {
		t.Fatal(err)
	}

	wavData, err := os.ReadFile(wavPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(wavData) < wav.HeaderSize {
		t.Fatal("WAV file too small")
	}

	var hdr wav.Header
	r := bytes.NewReader(wavData[:36])
	if err := binary.Read(r, binary.LittleEndian, &hdr); err != nil {
		t.Fatal(err)
	}

	if string(hdr.RiffID[:]) != "RIFF" {
		t.Errorf("bad RIFF ID: %s", hdr.RiffID)
	}
	if string(hdr.WaveID[:]) != "WAVE" {
		t.Errorf("bad WAVE ID: %s", hdr.WaveID)
	}
	if hdr.SampleRate != wav.DefaultSampleRate {
		t.Errorf("sample rate: got %d, want %d", hdr.SampleRate, wav.DefaultSampleRate)
	}
	if hdr.FormatTag != 1 {
		t.Errorf("format tag: got %d, want 1 (PCM)", hdr.FormatTag)
	}
	if hdr.BitsPerSample != 8 {
		t.Errorf("bits per sample: got %d, want 8", hdr.BitsPerSample)
	}
	if hdr.NumChannels != 1 {
		t.Errorf("channels: got %d, want 1 (mono)", hdr.NumChannels)
	}

	var dataHdr wav.DataHeader
	r2 := bytes.NewReader(wavData[36:44])
	if err := binary.Read(r2, binary.LittleEndian, &dataHdr); err != nil {
		t.Fatal(err)
	}

	expectedDataSize := uint32(len(wavData) - wav.HeaderSize)
	if dataHdr.DataSize != expectedDataSize {
		t.Errorf("data size: header says %d, actual %d", dataHdr.DataSize, expectedDataSize)
	}
	if hdr.FileSize != 36+expectedDataSize {
		t.Errorf("file size: header says %d, want %d", hdr.FileSize, 36+expectedDataSize)
	}

	// Verify no sample exceeds 8-bit range
	for i, s := range wavData[wav.HeaderSize:] {
		if s > 255 {
			t.Errorf("sample at %d exceeds 255: %d", i, s)
		}
	}
}

func TestConvertNoHeaders(t *testing.T) {
	dir := t.TempDir()
	casPath := filepath.Join(dir, "empty.cas")
	wavPath := filepath.Join(dir, "empty.wav")

	// A CAS file with no headers — should produce a WAV with just header, no data
	casData := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	if err := os.WriteFile(casPath, casData, 0644); err != nil {
		t.Fatal(err)
	}

	opts := Options{BaudRate: 1200, Machine: machine.MustLookup("msx")}
	if err := Convert(casPath, wavPath, opts); err != nil {
		t.Fatal(err)
	}

	wavData, err := os.ReadFile(wavPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(wavData) != wav.HeaderSize {
		t.Errorf("no-header CAS: expected only WAV header (%d bytes), got %d", wav.HeaderSize, len(wavData))
	}

	var dataHdr wav.DataHeader
	r := bytes.NewReader(wavData[36:44])
	if err := binary.Read(r, binary.LittleEndian, &dataHdr); err != nil {
		t.Fatal(err)
	}
	if dataHdr.DataSize != 0 {
		t.Errorf("no-header CAS: expected 0 data size, got %d", dataHdr.DataSize)
	}
}

func TestNilMachineFails(t *testing.T) {
	dir := t.TempDir()
	if err := Convert(
		filepath.Join(dir, "in.cas"),
		filepath.Join(dir, "out.wav"),
		Options{},
	); err == nil {
		t.Fatal("expected error for nil machine")
	}
}
