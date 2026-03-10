package intake

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
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

type Service struct{}

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

func NewService() *Service {
	return &Service{}
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

	duration, hasDuration := detectDuration(cleanedPath, file)

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

func detectDuration(path string, file *os.File) (time.Duration, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".wav":
		return wavDuration(file)
	default:
		return 0, false
	}
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
