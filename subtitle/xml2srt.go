package subtitle

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// xmlCue is a single parsed cue of a timedtext XML document.
type xmlCue struct {
	start    int
	duration int
	text     string
}

// parseXMLCues parses YouTube timedtext XML subtitles with a streaming
// decoder instead of a rigid struct mapping, so that nested elements
// (eg: <s> segments), HTML entities and slightly malformed documents
// are handled gracefully.
func parseXMLCues(content []byte) ([]xmlCue, error) {
	decoder := xml.NewDecoder(bytes.NewReader(content))
	decoder.Strict = false
	decoder.Entity = xml.HTMLEntity

	var cues []xmlCue
	var cue xmlCue
	var text strings.Builder
	inParagraph := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Tolerate truncated documents: flush the cue being parsed.
			if err == io.ErrUnexpectedEOF || strings.Contains(err.Error(), "unexpected EOF") {
				if inParagraph {
					cue.text = strings.TrimSpace(text.String())
					cues = append(cues, cue)
				}
				break
			}
			return nil, errors.WithStack(err)
		}
		switch element := token.(type) {
		case xml.StartElement:
			if element.Name.Local == "p" {
				inParagraph = true
				cue = xmlCue{
					start:    timeAttr(element, "t"),
					duration: timeAttr(element, "d"),
				}
				text.Reset()
			}
		case xml.EndElement:
			if element.Name.Local == "p" && inParagraph {
				inParagraph = false
				cue.text = strings.TrimSpace(text.String())
				cues = append(cues, cue)
			}
		case xml.CharData:
			if inParagraph {
				text.Write([]byte(element))
			}
		}
	}
	return cues, nil
}

// timeAttr extracts a millisecond timestamp attribute, tolerating both
// integer and decimal values.
func timeAttr(element xml.StartElement, name string) int {
	for _, attr := range element.Attr {
		if attr.Name.Local != name {
			continue
		}
		if value, err := strconv.Atoi(attr.Value); err == nil {
			return value
		}
		if value, err := strconv.ParseFloat(attr.Value, 64); err == nil {
			return int(value)
		}
	}
	return 0
}

// ConvertXMLToSRT converts YouTube XML subtitles to SRT format.
func ConvertXMLToSRT(content []byte) ([]byte, error) {
	cues, err := parseXMLCues(content)
	if err != nil {
		return nil, err
	}

	var srt strings.Builder
	index := 1
	for _, cue := range cues {
		// Skip empty lines
		if cue.text == "" {
			continue
		}
		fmt.Fprintf(&srt, "%d\n%s --> %s\n%s\n\n",
			index, formatSRTTime(cue.start), formatSRTTime(cue.start+cue.duration), cue.text)
		index++
	}
	return []byte(srt.String()), nil
}

func formatSRTTime(ms int) string {
	hours := ms / 3600000
	ms %= 3600000
	minutes := ms / 60000
	ms %= 60000
	seconds := ms / 1000
	ms %= 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, ms)
}

type xmlToSRTConverter struct{}

func (xmlToSRTConverter) FromExt() string { return "xml" }
func (xmlToSRTConverter) ToExt() string   { return "srt" }

func (xmlToSRTConverter) Convert(content []byte) ([]byte, error) {
	return ConvertXMLToSRT(content)
}
