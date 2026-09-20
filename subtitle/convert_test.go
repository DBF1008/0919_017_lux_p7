package subtitle

import (
	"strings"
	"testing"
)

func TestXMLToSRTConverter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			name:    "simple cues",
			content: `<?xml version="1.0" encoding="utf-8" ?><timedtext><body><p t="0" d="1000">Hello</p><p t="1000" d="2000">World</p></body></timedtext>`,
			want:    "1\n00:00:00,000 --> 00:00:01,000\nHello\n\n2\n00:00:01,000 --> 00:00:03,000\nWorld\n\n",
		},
		{
			name:    "nested s elements",
			content: `<timedtext><body><p t="500" d="1500"><s>Hello </s><s>there</s></p></body></timedtext>`,
			want:    "1\n00:00:00,500 --> 00:00:02,000\nHello there\n\n",
		},
		{
			name:    "empty cues are skipped",
			content: `<timedtext><body><p t="0" d="1000">  </p><p t="1000" d="1000">kept</p></body></timedtext>`,
			want:    "1\n00:00:01,000 --> 00:00:02,000\nkept\n\n",
		},
		{
			name:    "xml entities are decoded",
			content: `<timedtext><body><p t="0" d="1000">a &amp; b &lt;tag&gt;</p></body></timedtext>`,
			want:    "1\n00:00:00,000 --> 00:00:01,000\na & b <tag>\n\n",
		},
		{
			name:    "invalid xml",
			content: `<timedtext><body><p t="0"`,
			wantErr: true,
		},
	}
	converter := NewXMLToSRTConverter()
	if converter.FromExt() != "xml" || converter.ToExt() != "srt" {
		t.Fatalf("unexpected converter extensions: %s -> %s", converter.FromExt(), converter.ToExt())
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := converter.Convert([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Fatalf("Convert() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if string(got) != tt.want {
				t.Errorf("Convert() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVTTToSRTConverter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "basic vtt",
			content: "WEBVTT\n\n" +
				"00:00:01.000 --> 00:00:02.500\nHello\n\n" +
				"00:00:03.000 --> 00:00:04.000\nWorld\n",
			want: "1\n00:00:01,000 --> 00:00:02,500\nHello\n\n" +
				"2\n00:00:03,000 --> 00:00:04,000\nWorld\n\n",
		},
		{
			name: "cue identifiers and settings",
			content: "WEBVTT\n\n" +
				"cue-1\n00:00:01.000 --> 00:00:02.000 align:start position:0%\nHi\n",
			want: "1\n00:00:01,000 --> 00:00:02,000\nHi\n\n",
		},
		{
			name: "short timestamps are expanded",
			content: "WEBVTT\n\n" +
				"01:02.500 --> 01:03.000\nShort\n",
			want: "1\n00:01:02,500 --> 00:01:03,000\nShort\n\n",
		},
		{
			name: "note blocks are skipped",
			content: "WEBVTT\n\n" +
				"NOTE this is a comment\nspanning two lines\n\n" +
				"00:00:01.000 --> 00:00:02.000\nReal\n",
			want: "1\n00:00:01,000 --> 00:00:02,000\nReal\n\n",
		},
		{
			name: "crlf and multi-line cue",
			content: "WEBVTT\r\n\r\n" +
				"00:00:01.000 --> 00:00:03.000\r\nline one\r\nline two\r\n",
			want: "1\n00:00:01,000 --> 00:00:03,000\nline one\nline two\n\n",
		},
	}
	converter := NewVTTToSRTConverter()
	if converter.FromExt() != "vtt" || converter.ToExt() != "srt" {
		t.Fatalf("unexpected converter extensions: %s -> %s", converter.FromExt(), converter.ToExt())
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := converter.Convert([]byte(tt.content))
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("Convert() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatSRTTime(t *testing.T) {
	tests := []struct {
		ms   int
		want string
	}{
		{0, "00:00:00,000"},
		{999, "00:00:00,999"},
		{61001, "00:01:01,001"},
		{3661234, "01:01:01,234"},
	}
	for _, tt := range tests {
		if got := FormatSRTTime(tt.ms); got != tt.want {
			t.Errorf("FormatSRTTime(%d) = %s, want %s", tt.ms, got, tt.want)
		}
	}
}

func TestProcessorConvertUnregistered(t *testing.T) {
	p := New()
	if _, err := p.Convert("ass", []byte("data")); err == nil {
		t.Fatal("expected error for unregistered extension")
	}
	if !strings.Contains(p.Converter("xml").ToExt(), "srt") {
		t.Fatal("expected xml converter to be registered by default")
	}
	if !p.Convertible(".vtt") || !p.Convertible("XML") {
		t.Fatal("expected extension lookup to be case and dot insensitive")
	}
}
