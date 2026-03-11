package transcription

import (
	"path/filepath"
	"strconv"
	"time"
)

type ChunkPlan struct {
	Index          int    `json:"index"`
	StartMS        int    `json:"startMs"`
	DurationMS     int    `json:"durationMs"`
	AudioPath      string `json:"audioPath"`
	TranscriptPath string `json:"transcriptPath"`
	AlignmentPath  string `json:"alignmentPath"`
}

func ShouldChunk(duration time.Duration, prefs RunPreferences) bool {
	return duration > chunkDuration(prefs)
}

func BuildChunkPlan(workDir string, duration time.Duration, prefs RunPreferences) []ChunkPlan {
	perChunk := chunkDuration(prefs)
	if duration <= 0 || duration <= perChunk {
		return nil
	}

	totalMS := int(duration / time.Millisecond)
	perChunkMS := int(perChunk / time.Millisecond)
	chunks := make([]ChunkPlan, 0, (totalMS/perChunkMS)+1)
	for startMS, index := 0, 1; startMS < totalMS; startMS, index = startMS+perChunkMS, index+1 {
		durationMS := perChunkMS
		if remaining := totalMS - startMS; remaining < durationMS {
			durationMS = remaining
		}
		chunks = append(chunks, ChunkPlan{
			Index:          index,
			StartMS:        startMS,
			DurationMS:     durationMS,
			AudioPath:      filepath.Join(workDir, "chunks", "chunk-"+leftPad(index)+".wav"),
			TranscriptPath: filepath.Join(workDir, "artifacts", "chunk-"+leftPad(index)+"-transcript.json"),
			AlignmentPath:  filepath.Join(workDir, "artifacts", "chunk-"+leftPad(index)+"-alignment.json"),
		})
	}

	return chunks
}

func chunkDuration(prefs RunPreferences) time.Duration {
	minutes := prefs.AlignmentChunkMinutes
	if minutes <= 0 {
		minutes = defaultPreferences().AlignmentChunkMinutes
	}
	return time.Duration(minutes) * time.Minute
}

func leftPad(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}
