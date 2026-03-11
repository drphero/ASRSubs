package transcription

import (
	"fmt"
	"strings"
)

type WordTimestamp struct {
	Text       string  `json:"text"`
	StartMS    int     `json:"startMs"`
	EndMS      int     `json:"endMs"`
	Confidence float64 `json:"confidence,omitempty"`
}

type SubtitleSegment struct {
	Index   int      `json:"index"`
	StartMS int      `json:"startMs"`
	EndMS   int      `json:"endMs"`
	Text    string   `json:"text"`
	Lines   []string `json:"lines"`
}

type Timeline struct {
	Words     []WordTimestamp   `json:"words"`
	Subtitles []SubtitleSegment `json:"subtitles"`
}

func BuildSubtitles(words []WordTimestamp, prefs RunPreferences) ([]SubtitleSegment, error) {
	if len(words) == 0 {
		return nil, fmt.Errorf("no aligned words were available to build subtitles")
	}

	if prefs.OneWordPerSubtitle {
		subtitles := make([]SubtitleSegment, 0, len(words))
		for index, word := range words {
			subtitles = append(subtitles, SubtitleSegment{
				Index:   index + 1,
				StartMS: word.StartMS,
				EndMS:   word.EndMS,
				Text:    word.Text,
				Lines:   []string{word.Text},
			})
		}
		return subtitles, nil
	}

	maxLineLength := prefs.MaxLineLength
	if maxLineLength <= 0 {
		maxLineLength = defaultPreferences().MaxLineLength
	}
	maxLines := prefs.LinesPerSubtitle
	if maxLines <= 0 {
		maxLines = defaultPreferences().LinesPerSubtitle
	}

	var subtitles []SubtitleSegment
	var lineWords []string
	var lines []string
	startMS := 0
	lastEndMS := 0
	open := false

	flush := func() {
		if len(lineWords) > 0 {
			lines = append(lines, strings.Join(lineWords, " "))
			lineWords = nil
		}
		if len(lines) == 0 {
			return
		}
		subtitles = append(subtitles, SubtitleSegment{
			Index:   len(subtitles) + 1,
			StartMS: startMS,
			EndMS:   lastEndMS,
			Text:    strings.Join(lines, "\n"),
			Lines:   append([]string(nil), lines...),
		})
		lines = nil
		open = false
	}

	for _, word := range words {
		if !open {
			startMS = word.StartMS
			open = true
		}

		if len(lineWords) == 0 {
			lineWords = append(lineWords, word.Text)
			lastEndMS = word.EndMS
			continue
		}

		currentLine := strings.Join(append(append([]string(nil), lineWords...), word.Text), " ")
		gap := word.StartMS - lastEndMS
		switch {
		case gap > 1200:
			flush()
			startMS = word.StartMS
			open = true
			lineWords = append(lineWords, word.Text)
		case len(currentLine) <= maxLineLength:
			lineWords = append(lineWords, word.Text)
		case len(lines)+1 < maxLines:
			lines = append(lines, strings.Join(lineWords, " "))
			lineWords = []string{word.Text}
		default:
			flush()
			startMS = word.StartMS
			open = true
			lineWords = []string{word.Text}
		}

		lastEndMS = word.EndMS
	}

	flush()
	return subtitles, nil
}
