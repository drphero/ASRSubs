package intake

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateMediaFileRejectsUnsupportedType(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "notes.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	service := NewService()
	_, err := service.ValidateMediaFile(path)
	if err == nil || err.Error() != "This file type isn't supported." {
		t.Fatalf("expected unsupported type error, got %v", err)
	}
}

func TestValidateMediaFileReturnsMetadata(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "sample.wav")
	if err := os.WriteFile(path, wavFixture(2), 0o644); err != nil {
		t.Fatalf("write wav: %v", err)
	}

	service := NewService()
	metadata, err := service.ValidateMediaFile(path)
	if err != nil {
		t.Fatalf("validate media: %v", err)
	}

	if metadata.Name != "sample.wav" {
		t.Fatalf("unexpected name: %s", metadata.Name)
	}

	if !metadata.HasKnownDuration {
		t.Fatalf("expected duration to be available")
	}

	if metadata.DurationLabel != "0:02" {
		t.Fatalf("unexpected duration label: %s", metadata.DurationLabel)
	}
}

func wavFixture(seconds int) []byte {
	sampleRate := 8000
	bitsPerSample := 16
	channels := 1
	byteRate := sampleRate * channels * bitsPerSample / 8
	dataSize := seconds * byteRate
	fileSize := 36 + dataSize

	buf := make([]byte, 44+dataSize)
	copy(buf[0:4], []byte("RIFF"))
	put32LE(buf[4:8], uint32(fileSize))
	copy(buf[8:12], []byte("WAVE"))
	copy(buf[12:16], []byte("fmt "))
	put32LE(buf[16:20], 16)
	put16LE(buf[20:22], 1)
	put16LE(buf[22:24], uint16(channels))
	put32LE(buf[24:28], uint32(sampleRate))
	put32LE(buf[28:32], uint32(byteRate))
	put16LE(buf[32:34], uint16(channels*bitsPerSample/8))
	put16LE(buf[34:36], uint16(bitsPerSample))
	copy(buf[36:40], []byte("data"))
	put32LE(buf[40:44], uint32(dataSize))

	return buf
}

func put16LE(dst []byte, value uint16) {
	dst[0] = byte(value)
	dst[1] = byte(value >> 8)
}

func put32LE(dst []byte, value uint32) {
	dst[0] = byte(value)
	dst[1] = byte(value >> 8)
	dst[2] = byte(value >> 16)
	dst[3] = byte(value >> 24)
}
