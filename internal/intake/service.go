package intake

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	asrruntime "ASRSubs/internal/runtime"
)

type MediaMetadata struct {
	Path             string  `json:"path"`
	Name             string  `json:"name"`
	Extension        string  `json:"extension"`
	Directory        string  `json:"directory"`
	SizeBytes        int64   `json:"sizeBytes"`
	DurationSeconds  float64 `json:"durationSeconds"`
	DurationLabel    string  `json:"durationLabel"`
	HasKnownDuration bool    `json:"hasKnownDuration"`
}

type DurationProber func(ctx context.Context, inputPath string) (time.Duration, error)
type Option func(*Service)

type Service struct {
	probeDuration DurationProber
}

var supportedExtensions = map[string]struct{}{
	".wav":  {},
	".mp3":  {},
	".m4a":  {},
	".aac":  {},
	".flac": {},
	".ogg":  {},
	".opus": {},
	".mp4":  {},
	".mov":  {},
	".m4v":  {},
	".mkv":  {},
	".avi":  {},
	".webm": {},
}

func WithDurationProber(prober DurationProber) Option {
	return func(service *Service) {
		service.probeDuration = prober
	}
}

func NewService(options ...Option) *Service {
	service := &Service{
		probeDuration: probeDurationWithFFprobe,
	}

	for _, option := range options {
		option(service)
	}

	return service
}

func (s *Service) ValidateMediaFile(path string) (MediaMetadata, error) {
	cleanedPath := strings.TrimSpace(path)
	if cleanedPath == "" {
		return MediaMetadata{}, fmt.Errorf("Choose a media file to continue.")
	}

	info, err := os.Stat(cleanedPath)
	if err != nil {
		return MediaMetadata{}, fmt.Errorf("This file could not be read.")
	}

	if info.IsDir() {
		return MediaMetadata{}, fmt.Errorf("Choose a media file, not a folder.")
	}

	extension := strings.ToLower(filepath.Ext(info.Name()))
	if _, ok := supportedExtensions[extension]; !ok {
		return MediaMetadata{}, fmt.Errorf("This file type isn't supported.")
	}

	file, err := os.Open(cleanedPath)
	if err != nil {
		return MediaMetadata{}, fmt.Errorf("This file could not be opened.")
	}
	defer file.Close()

	duration, hasDuration := detectDuration(cleanedPath, file, s.probeDuration)

	metadata := MediaMetadata{
		Path:             cleanedPath,
		Name:             info.Name(),
		Extension:        extension,
		Directory:        filepath.Dir(cleanedPath),
		SizeBytes:        info.Size(),
		DurationSeconds:  duration.Seconds(),
		HasKnownDuration: hasDuration,
	}
	if hasDuration {
		metadata.DurationLabel = formatDuration(duration)
	} else {
		metadata.DurationLabel = "Duration unavailable"
	}

	return metadata, nil
}

func detectDuration(path string, file *os.File, probe DurationProber) (time.Duration, bool) {
	if probe != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		duration, err := probe(ctx, path)
		if err == nil && duration > 0 {
			return duration, true
		}
	}

	if strings.EqualFold(filepath.Ext(path), ".wav") {
		return wavDuration(file)
	}

	return 0, false
}

func wavDuration(file *os.File) (time.Duration, bool) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return 0, false
	}

	header := make([]byte, 12)
	if _, err := io.ReadFull(file, header); err != nil {
		return 0, false
	}

	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return 0, false
	}

	var byteRate uint32
	var dataSize uint32

	for {
		chunkHeader := make([]byte, 8)
		if _, err := io.ReadFull(file, chunkHeader); err != nil {
			return 0, false
		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		switch chunkID {
		case "fmt ":
			payload := make([]byte, chunkSize)
			if _, err := io.ReadFull(file, payload); err != nil {
				return 0, false
			}
			if len(payload) < 12 {
				return 0, false
			}
			byteRate = binary.LittleEndian.Uint32(payload[8:12])
		case "data":
			dataSize = chunkSize
			if _, err := file.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
				return 0, false
			}
		default:
			if _, err := file.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
				return 0, false
			}
		}

		if chunkSize%2 == 1 {
			if _, err := file.Seek(1, io.SeekCurrent); err != nil {
				return 0, false
			}
		}

		if byteRate > 0 && dataSize > 0 {
			seconds := float64(dataSize) / float64(byteRate)
			return time.Duration(seconds * float64(time.Second)), true
		}
	}
}

func formatDuration(duration time.Duration) string {
	if duration < 0 {
		duration = 0
	}

	rounded := duration.Round(time.Second)
	totalSeconds := int(rounded / time.Second)
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}

	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func probeDurationWithFFprobe(ctx context.Context, inputPath string) (time.Duration, error) {
	ffprobePath, err := resolveBinaryPath("ffprobe")
	if err != nil {
		return 0, err
	}

	cmd := exec.CommandContext(
		ctx,
		ffprobePath,
		"-v",
		"error",
		"-show_entries",
		"format=duration",
		"-of",
		"default=noprint_wrappers=1:nokey=1",
		inputPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	return time.ParseDuration(strings.TrimSpace(string(output)) + "s")
}

func resolveBinaryPath(name string) (string, error) {
	envKey := "ASRSUBS_FFMPEG_PATH"
	if name == "ffprobe" {
		envKey = "ASRSUBS_FFPROBE_PATH"
	}

	if candidate := strings.TrimSpace(os.Getenv(envKey)); candidate != "" {
		if fileExists(candidate) {
			return candidate, nil
		}
		return "", fmt.Errorf("%s is set but points to a missing file: %s", envKey, candidate)
	}

	bundledName := name
	if goruntime.GOOS == "windows" {
		bundledName += ".exe"
	}

	if bundled := asrruntime.ResolveBundledResourcePath("bin", bundledName); bundled != "" {
		return bundled, nil
	}

	resolved, err := exec.LookPath(name)
	if err == nil {
		return resolved, nil
	}

	return "", fmt.Errorf(
		"%s is unavailable: package it under bin/%s in the app resources or install it on PATH",
		name,
		bundledName,
	)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
