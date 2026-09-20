package subtitle

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// xmlToSRTConverter converts YouTube XML subtitles to SRT format.
type xmlToSRTConverter struct{}

// NewXMLToSRTConverter returns a Converter for YouTube XML subtitles.
func NewXMLToSRTConverter() Converter {
	return xmlToSRTConverter{}
}

func (xmlToSRTConverter) FromExt() string { return "xml" }
func (xmlToSRTConverter) ToExt() string   { return "srt" }

// xmlTranscript mirrors the structure of YouTube XML subtitles.
type xmlTranscript struct {
	Body struct {
		P []struct {
			T    int    `xml:"t,attr"`
			D    int    `xml:"d,attr"`
			Text string `xml:",chardata"`
			S    []struct {
				Text string `xml:",chardata"`
			} `xml:"s"`
		} `xml:"p"`
	} `xml:"body"`
}

func (xmlToSRTConverter) Convert(data []byte) ([]byte, error) {
	var transcript xmlTranscript
	if err := xml.Unmarshal(data, &transcript); err != nil {
		return nil, fmt.Errorf("invalid XML subtitle: %w", err)
	}

	var srt strings.Builder
	index := 1
	for _, p := range transcript.Body.P {
		var text string
		if len(p.S) > 0 {
			var sb strings.Builder
			for _, s := range p.S {
				sb.WriteString(s.Text)
			}
			text = sb.String()
		} else {
			text = p.Text
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		fmt.Fprintf(&srt, "%d\n%s --> %s\n%s\n\n",
			index, FormatSRTTime(p.T), FormatSRTTime(p.T+p.D), text)
		index++
	}
	return []byte(srt.String()), nil
}

// vttTimestamp matches a WebVTT cue timestamp, e.g. 00:01:02.345 or 01:02.345.
var vttTimestamp = regexp.MustCompile(`(\d{2}:\d{2}(?::\d{2})?)\.(\d{3})`)

// vttToSRTConverter converts WebVTT subtitles to SRT format.
type vttToSRTConverter struct{}

// NewVTTToSRTConverter returns a Converter for WebVTT subtitles.
func NewVTTToSRTConverter() Converter {
	return vttToSRTConverter{}
}

func (vttToSRTConverter) FromExt() string { return "vtt" }
func (vttToSRTConverter) ToExt() string   { return "srt" }

func (vttToSRTConverter) Convert(data []byte) ([]byte, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.TrimPrefix(text, "\ufeff")
	blocks := strings.Split(text, "\n\n")

	var srt strings.Builder
	index := 1
	for i, block := range blocks {
		lines := strings.Split(strings.Trim(block, "\n"), "\n")
		if len(lines) == 0 {
			continue
		}
		// Skip the WEBVTT header and NOTE/STYLE/REGION blocks.
		head := strings.TrimSpace(lines[0])
		if i == 0 && strings.HasPrefix(head, "WEBVTT") {
			continue
		}
		if strings.HasPrefix(head, "NOTE") || head == "STYLE" || head == "REGION" {
			continue
		}
		// A cue identifier line may precede the timing line.
		timingIdx := 0
		if !vttTimestamp.MatchString(head) && len(lines) > 1 {
			timingIdx = 1
		}
		if !strings.Contains(lines[timingIdx], "-->") {
			continue
		}
		// Normalize short timestamps (MM:SS,mmm) to HH:MM:SS,mmm.
		timing := normalizeSRTTimingLine(lines[timingIdx])
		if timing == "" {
			continue
		}
		srt.WriteString(strconv.Itoa(index) + "\n" + timing + "\n")
		for _, line := range lines[timingIdx+1:] {
			srt.WriteString(line + "\n")
		}
		srt.WriteString("\n")
		index++
	}
	return []byte(srt.String()), nil
}

// normalizeSRTTimingLine expands MM:SS,mmm timestamps to HH:MM:SS,mmm and
// strips cue settings after the end timestamp. It returns an empty string
// when the line is not a valid timing line.
func normalizeSRTTimingLine(line string) string {
	parts := strings.SplitN(line, "-->", 2)
	if len(parts) != 2 {
		return ""
	}
	start := normalizeSRTTimestamp(strings.TrimSpace(parts[0]))
	// Drop cue settings (e.g. "align:start position:0%") after the end time.
	end := strings.TrimSpace(strings.Fields(parts[1])[0])
	end = normalizeSRTTimestamp(end)
	if start == "" || end == "" {
		return ""
	}
	// SRT uses a comma as the millisecond separator.
	return vttTimestamp.ReplaceAllString(start+" --> "+end, "$1,$2")
}

// normalizeSRTTimestamp converts MM:SS,mmm to HH:MM:SS,mmm; already
// normalized timestamps are returned unchanged.
func normalizeSRTTimestamp(ts string) string {
	if !vttTimestamp.MatchString(ts) {
		return ""
	}
	if len(strings.Split(ts, ":")) == 2 {
		return "00:" + ts
	}
	return ts
}

// FormatSRTTime formats milliseconds as an SRT timestamp (HH:MM:SS,mmm).
func FormatSRTTime(ms int) string {
	hours := ms / 3600000
	ms %= 3600000
	minutes := ms / 60000
	ms %= 60000
	seconds := ms / 1000
	ms %= 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, ms)
}
