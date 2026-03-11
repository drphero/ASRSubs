package transcription

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var srtTimeRangePattern = regexp.MustCompile(`^(\d{2}):(\d{2}):(\d{2}),(\d{3}) --> (\d{2}):(\d{2}):(\d{2}),(\d{3})$`)

type SubtitleDraft struct {
	Text              string `json:"text"`
	SuggestedFilename string `json:"suggestedFilename"`
	SourceFilePath    string `json:"sourceFilePath"`
	SourceFileName    string `json:"sourceFileName"`
}

type ValidationIssue struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

func (v *ValidationIssue) Error() string {
	if v == nil {
		return ""
	}
	if v.Line <= 0 {
		return v.Message
	}
	return fmt.Sprintf("line %d: %s", v.Line, v.Message)
}

func SerializeSRT(subtitles []SubtitleSegment) string {
	if len(subtitles) == 0 {
		return ""
	}

	var builder strings.Builder
	for index, subtitle := range subtitles {
		if index > 0 {
			builder.WriteString("\n\n")
		}

		builder.WriteString(strconv.Itoa(index + 1))
		builder.WriteByte('\n')
		builder.WriteString(formatSRTTimestamp(subtitle.StartMS))
		builder.WriteString(" --> ")
		builder.WriteString(formatSRTTimestamp(subtitle.EndMS))
		builder.WriteByte('\n')

		lines := subtitle.Lines
		if len(lines) == 0 && subtitle.Text != "" {
			lines = strings.Split(subtitle.Text, "\n")
		}
		if len(lines) == 0 {
			lines = []string{""}
		}

		for lineIndex, line := range lines {
			if lineIndex > 0 {
				builder.WriteByte('\n')
			}
			builder.WriteString(strings.TrimRight(line, "\r"))
		}
	}

	builder.WriteByte('\n')
	return builder.String()
}

func formatSRTTimestamp(milliseconds int) string {
	if milliseconds < 0 {
		milliseconds = 0
	}

	hours := milliseconds / 3_600_000
	minutes := (milliseconds % 3_600_000) / 60_000
	seconds := (milliseconds % 60_000) / 1_000
	remainder := milliseconds % 1_000

	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, remainder)
}

func ValidateSRT(text string) *ValidationIssue {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	trimmed := strings.TrimSpace(normalized)
	if trimmed == "" {
		return &ValidationIssue{Line: 1, Message: "Subtitle text is empty."}
	}

	lines := strings.Split(normalized, "\n")
	lineIndex := 0
	expectedIndex := 1
	previousStartMS := -1

	for lineIndex < len(lines) {
		for lineIndex < len(lines) && strings.TrimSpace(lines[lineIndex]) == "" {
			lineIndex++
		}
		if lineIndex >= len(lines) {
			break
		}

		blockLine := lineIndex + 1
		numberLine := strings.TrimSpace(lines[lineIndex])
		number, err := strconv.Atoi(numberLine)
		if err != nil {
			return &ValidationIssue{Line: blockLine, Message: "Expected a numeric subtitle index."}
		}
		if number != expectedIndex {
			return &ValidationIssue{Line: blockLine, Message: fmt.Sprintf("Expected subtitle index %d.", expectedIndex)}
		}
		expectedIndex++
		lineIndex++

		if lineIndex >= len(lines) {
			return &ValidationIssue{Line: blockLine, Message: "Missing timestamp line."}
		}

		timestampLineNumber := lineIndex + 1
		startMS, endMS, issue := parseSRTTimeRange(strings.TrimSpace(lines[lineIndex]), timestampLineNumber)
		if issue != nil {
			return issue
		}
		if previousStartMS > startMS {
			return &ValidationIssue{Line: timestampLineNumber, Message: "Subtitle timestamps must stay in chronological order."}
		}
		previousStartMS = startMS
		if endMS <= startMS {
			return &ValidationIssue{Line: timestampLineNumber, Message: "Subtitle end time must be after its start time."}
		}
		lineIndex++

		textLineCount := 0
		for lineIndex < len(lines) && strings.TrimSpace(lines[lineIndex]) != "" {
			if isSRTBlockBoundaryMissing(lines, lineIndex) {
				return &ValidationIssue{Line: lineIndex + 1, Message: "Separate subtitle blocks with a blank line."}
			}
			textLineCount++
			lineIndex++
		}
		if textLineCount == 0 {
			return &ValidationIssue{Line: timestampLineNumber, Message: "Each subtitle block needs at least one text line."}
		}
	}

	return nil
}

func DraftFilenameForMedia(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		name = "subtitles"
	}
	return name + ".srt"
}

func parseSRTTimeRange(line string, lineNumber int) (int, int, *ValidationIssue) {
	match := srtTimeRangePattern.FindStringSubmatch(line)
	if match == nil {
		return 0, 0, &ValidationIssue{Line: lineNumber, Message: "Expected a timestamp range like 00:00:01,000 --> 00:00:02,500."}
	}

	startMS := parseTimestampParts(match[1], match[2], match[3], match[4])
	endMS := parseTimestampParts(match[5], match[6], match[7], match[8])
	return startMS, endMS, nil
}

func parseTimestampParts(hourText string, minuteText string, secondText string, millisecondText string) int {
	hours, _ := strconv.Atoi(hourText)
	minutes, _ := strconv.Atoi(minuteText)
	seconds, _ := strconv.Atoi(secondText)
	milliseconds, _ := strconv.Atoi(millisecondText)

	return (hours * 3_600_000) + (minutes * 60_000) + (seconds * 1_000) + milliseconds
}

func isSRTBlockBoundaryMissing(lines []string, index int) bool {
	if index+1 >= len(lines) {
		return false
	}

	current := strings.TrimSpace(lines[index])
	next := strings.TrimSpace(lines[index+1])
	if current == "" || next == "" {
		return false
	}

	if _, err := strconv.Atoi(current); err != nil {
		return false
	}

	return srtTimeRangePattern.MatchString(next)
}
