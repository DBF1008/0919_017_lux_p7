package subtitle

import (
	"fmt"
	"strings"
)

// ConvertVTTToSRT converts WebVTT subtitles to SRT format.
func ConvertVTTToSRT(content []byte) ([]byte, error) {
	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	text = strings.TrimPrefix(text, "\ufeff")
	lines := strings.Split(text, "\n")

	// Skip the WEBVTT header block.
	i := 0
	if len(lines) > 0 && strings.HasPrefix(lines[i], "WEBVTT") {
		for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			i++
		}
	}

	var srt strings.Builder
	index := 1
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		// Skip blank lines and NOTE/STYLE/REGION blocks.
		if line == "" {
			i++
			continue
		}
		if strings.HasPrefix(line, "NOTE") || line == "STYLE" || line == "REGION" {
			for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
				i++
			}
			continue
		}
		// A cue may start with an optional identifier line before the timing line.
		if !strings.Contains(line, "-->") {
			if i+1 < len(lines) && strings.Contains(lines[i+1], "-->") {
				i++
				line = strings.TrimSpace(lines[i])
			} else {
				i++
				continue
			}
		}
		timing := convertVTTTiming(line)
		i++
		var payload []string
		for i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			payload = append(payload, lines[i])
			i++
		}
		if len(payload) == 0 {
			continue
		}
		fmt.Fprintf(&srt, "%d\n%s\n%s\n\n", index, timing, strings.Join(payload, "\n"))
		index++
	}
	return []byte(srt.String()), nil
}

// convertVTTTiming converts a WebVTT timing line to the SRT format,
// eg: "00:00:01.000 --> 00:00:04.000 align:start" -> "00:00:01,000 --> 00:00:04,000".
func convertVTTTiming(line string) string {
	parts := strings.SplitN(line, "-->", 2)
	start := convertVTTTimestamp(strings.TrimSpace(parts[0]))
	end := ""
	if len(parts) == 2 {
		// Drop any cue settings after the end timestamp.
		end = convertVTTTimestamp(strings.Fields(strings.TrimSpace(parts[1]))[0])
	}
	return start + " --> " + end
}

func convertVTTTimestamp(timestamp string) string {
	// Pad "mm:ss.mmm" to "hh:mm:ss,mmm".
	if len(strings.Split(timestamp, ":")) == 2 {
		timestamp = "00:" + timestamp
	}
	return strings.ReplaceAll(timestamp, ".", ",")
}

type vttToSRTConverter struct{}

func (vttToSRTConverter) FromExt() string { return "vtt" }
func (vttToSRTConverter) ToExt() string   { return "srt" }

func (vttToSRTConverter) Convert(content []byte) ([]byte, error) {
	return ConvertVTTToSRT(content)
}
