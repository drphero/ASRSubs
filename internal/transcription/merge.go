package transcription

import "sort"

func MergeTimeline(chunks [][]WordTimestamp) []WordTimestamp {
	merged := make([]WordTimestamp, 0)
	for _, chunk := range chunks {
		merged = append(merged, chunk...)
	}

	sort.SliceStable(merged, func(i, j int) bool {
		if merged[i].StartMS == merged[j].StartMS {
			return merged[i].EndMS < merged[j].EndMS
		}
		return merged[i].StartMS < merged[j].StartMS
	})

	for index := 1; index < len(merged); index++ {
		previous := merged[index-1]
		current := &merged[index]
		if current.StartMS < previous.EndMS && previous.EndMS-current.StartMS <= 250 {
			shift := previous.EndMS - current.StartMS
			current.StartMS += shift
			current.EndMS += shift
		}
		if current.EndMS <= current.StartMS {
			current.EndMS = current.StartMS + 120
		}
	}

	return merged
}
