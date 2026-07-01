package msx

import (
	"bytes"
	"testing"

	"github.com/carlosrabelo/retrokit/retrokit/internal/machine"
	"github.com/carlosrabelo/retrokit/retrokit/internal/wav"
)

func TestPulseLength(t *testing.T) {
	// At 1200 baud, a 1200 Hz pulse = 36 samples
	buf := new(bytes.Buffer)
	writePulse(buf, 1200, longPulseFreq)
	if buf.Len() != 36 {
		t.Errorf("1200Hz pulse at 1200 baud: got %d samples, want 36", buf.Len())
	}

	// At 1200 baud, a 2400 Hz pulse = 18 samples
	buf.Reset()
	writePulse(buf, 1200, shortPulseFreq)
	if buf.Len() != 18 {
		t.Errorf("2400Hz pulse at 1200 baud: got %d samples, want 18", buf.Len())
	}
}

func TestWriteByte(t *testing.T) {
	buf := new(bytes.Buffer)
	writeByte(buf, 1200, 0x55) // 01010101

	// Each byte = 1 start + 8 data + 4 stop = 13 pulses
	// At 1200 baud:
	//   start: 1 long (36) = 36
	//   bit 0 (LSB=1): 2 short (18*2) = 36
	//   bit 1 (0): 1 long = 36
	//   bit 2 (1): 2 short = 36
	//   bit 3 (0): 1 long = 36
	//   bit 4 (1): 2 short = 36
	//   bit 5 (0): 1 long = 36
	//   bit 6 (1): 2 short = 36
	//   bit 7 (0): 1 long = 36
	//   stop: 4 short (18*4) = 72
	// Total = 36*5 + 36*4 + 72 = 180 + 144 + 72 = 396
	expected := 396
	if buf.Len() != expected {
		t.Errorf("writeByte(0x55): got %d samples, want %d", buf.Len(), expected)
	}
}

func TestDetect(t *testing.T) {
	m := &MSX{}

	var buf bytes.Buffer
	buf.Write(headerPattern)
	buf.Write(binPattern)
	buf.Write([]byte{0x01, 0x02, 0x1A})
	if !m.Detect(buf.Bytes()) {
		t.Error("expected MSX CAS to be detected")
	}

	// Raw bytes without a header should not be detected.
	if m.Detect([]byte{0x00, 0x01, 0x02, 0x03}) {
		t.Error("expected non-CAS data to not be detected")
	}
}

func TestWrapExtractRoundTrip(t *testing.T) {
	m := &MSX{}
	payload := []byte{0xAA, 0xBB, 0xCC, 0xDD}

	cas := m.Wrap(payload, machine.WrapOpts{Type: "bin"})

	got, blocks := m.Extract(cas)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("round trip: got % X, want % X", got, payload)
	}
}

func TestSampleRateConstant(t *testing.T) {
	if wav.DefaultSampleRate != 43200 {
		t.Errorf("DefaultSampleRate: got %d, want 43200", wav.DefaultSampleRate)
	}
}
