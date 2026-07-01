// Package wav reads, writes, and inspects PCM WAV audio files used by the
// cassette conversion tools.
package wav

import (
	"encoding/binary"
	"fmt"
	"os"
)

// DefaultSampleRate is the sample rate used when encoding audio to WAV.
const DefaultSampleRate = 43200

// HeaderSize is the size of the canonical PCM WAV header (fmt + data chunks).
const HeaderSize = 44

// Header is the 36-byte RIFF/WAVE/fmt chunk header.
type Header struct {
	RiffID        [4]byte
	FileSize      uint32
	WaveID        [4]byte
	FmtID         [4]byte
	FmtSize       uint32
	FormatTag     uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
}

// DataHeader is the 8-byte data chunk header.
type DataHeader struct {
	DataID   [4]byte
	DataSize uint32
}

// Read loads a PCM WAV file and returns signed 8-bit mono samples.
// phaseShift inverts the polarity of each sample.
func Read(path string, phaseShift bool) ([]int8, uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("opening WAV: %w", err)
	}
	defer f.Close()

	var hdr Header
	if err := binary.Read(f, binary.LittleEndian, &hdr); err != nil {
		return nil, 0, fmt.Errorf("reading WAV header: %w", err)
	}

	if string(hdr.RiffID[:]) != "RIFF" || string(hdr.WaveID[:]) != "WAVE" {
		return nil, 0, fmt.Errorf("invalid WAV file")
	}
	if hdr.FormatTag != 1 {
		return nil, 0, fmt.Errorf("unsupported format: %d (only PCM)", hdr.FormatTag)
	}

	adder := int(hdr.NumChannels) * int(hdr.BitsPerSample) / 8

	var block DataHeader
	found := false
	var data []int8
	var dataSize uint32

	for {
		pos, _ := f.Seek(0, 1)
		if err := binary.Read(f, binary.LittleEndian, &block); err != nil {
			break
		}
		if string(block.DataID[:]) == "data" {
			dataSize = block.DataSize / uint32(adder)
			data = make([]int8, dataSize)
			found = true
			break
		}
		f.Seek(int64(pos)+1, 0)
	}

	if !found {
		return nil, 0, fmt.Errorf("data chunk not found")
	}

	fmt.Fprintf(os.Stderr, "Reading %s (%d Hz, %d-bits, %s)...\n",
		path, hdr.SampleRate, hdr.BitsPerSample,
		map[uint16]string{1: "mono", 2: "stereo"}[hdr.NumChannels])

	for i := uint32(0); i < dataSize; i++ {
		for j := 1; j < adder; j++ {
			var dummy int8
			binary.Read(f, binary.LittleEndian, &dummy)
		}
		var sample int8
		binary.Read(f, binary.LittleEndian, &sample)
		if hdr.BitsPerSample == 8 {
			v := uint8(sample)
			v ^= 128
			sample = int8(v)
		}
		if phaseShift {
			sample = -sample
		}
		data[i] = sample
	}

	return data, hdr.SampleRate, nil
}

// Write saves 8-bit unsigned mono PCM samples as a WAV file at DefaultSampleRate.
func Write(path string, samples []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	dataSize := uint32(len(samples))
	hdr := Header{
		RiffID:        [4]byte{'R', 'I', 'F', 'F'},
		FileSize:      36 + dataSize,
		WaveID:        [4]byte{'W', 'A', 'V', 'E'},
		FmtID:         [4]byte{'f', 'm', 't', ' '},
		FmtSize:       16,
		FormatTag:     1,
		NumChannels:   1,
		SampleRate:    DefaultSampleRate,
		ByteRate:      DefaultSampleRate,
		BlockAlign:    1,
		BitsPerSample: 8,
	}
	if err := binary.Write(f, binary.LittleEndian, &hdr); err != nil {
		return fmt.Errorf("writing WAV header: %w", err)
	}

	data := DataHeader{
		DataID:   [4]byte{'d', 'a', 't', 'a'},
		DataSize: dataSize,
	}
	if err := binary.Write(f, binary.LittleEndian, &data); err != nil {
		return fmt.Errorf("writing WAV data header: %w", err)
	}

	if _, err := f.Write(samples); err != nil {
		return fmt.Errorf("writing WAV data: %w", err)
	}
	return nil
}

// Info describes the audio properties of a WAV file.
type Info struct {
	SampleRate    uint32
	BitsPerSample uint16
	NumChannels   uint16
	Duration      float64
	Frames        int
}

// Inspect reads only the metadata of a WAV file.
func Inspect(path string) (*Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening WAV: %w", err)
	}
	defer f.Close()

	var hdr Header
	if err := binary.Read(f, binary.LittleEndian, &hdr); err != nil {
		return nil, fmt.Errorf("reading WAV header: %w", err)
	}

	if string(hdr.RiffID[:]) != "RIFF" || string(hdr.WaveID[:]) != "WAVE" {
		return nil, fmt.Errorf("invalid WAV file")
	}
	if hdr.FormatTag != 1 {
		return nil, fmt.Errorf("unsupported format: %d (only PCM)", hdr.FormatTag)
	}

	info := &Info{
		SampleRate:    hdr.SampleRate,
		BitsPerSample: hdr.BitsPerSample,
		NumChannels:   hdr.NumChannels,
	}

	var block DataHeader
	for {
		pos, _ := f.Seek(0, 1)
		if err := binary.Read(f, binary.LittleEndian, &block); err != nil {
			break
		}
		if string(block.DataID[:]) == "data" {
			adder := int(hdr.NumChannels) * int(hdr.BitsPerSample) / 8
			frames := int(block.DataSize / uint32(adder))
			info.Frames = frames
			if hdr.SampleRate > 0 {
				info.Duration = float64(frames) / float64(hdr.SampleRate)
			}
			break
		}
		f.Seek(int64(pos)+1, 0)
	}

	return info, nil
}
