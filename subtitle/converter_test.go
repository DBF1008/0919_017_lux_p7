package subtitle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConvertXMLToSRT(t *testing.T) {
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
			content: `<timedtext><body><p t="500" d="1500"><s>Hello</s> <s>there</s></p></body></timedtext>`,
			want:    "1\n00:00:00,500 --> 00:00:02,000\nHello there\n\n",
		},
		{
			name:    "html entities",
			content: `<timedtext><body><p t="0" d="1000">Tom &amp; Jerry&nbsp;forever</p></body></timedtext>`,
			want:    "1\n00:00:00,000 --> 00:00:01,000\nTom & Jerry forever\n\n",
		},
		{
			name:    "empty cues are skipped",
			content: `<timedtext><body><p t="0" d="1000"> </p><p t="1000" d="1000">Hi</p></body></timedtext>`,
			want:    "1\n00:00:01,000 --> 00:00:02,000\nHi\n\n",
		},
		{
			name:    "decimal timestamps",
			content: `<timedtext><body><p t="1500.0" d="500.5">Hi</p></body></timedtext>`,
			want:    "1\n00:00:01,500 --> 00:00:02,000\nHi\n\n",
		},
		{
			name:    "malformed xml",
			content: `<timedtext><body><p t="0" d="1000">Hi</p>`,
			want:    "1\n00:00:00,000 --> 00:00:01,000\nHi\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertXMLToSRT([]byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Fatalf("ConvertXMLToSRT() error = %v, wantErr %v", err, tt.wantErr)
			}
			if string(got) != tt.want {
				t.Errorf("ConvertXMLToSRT() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConvertVTTToSRT(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "simple cues",
			content: "WEBVTT\n\n" +
				"00:00:01.000 --> 00:00:04.000\nHello\n\n" +
				"00:00:05.000 --> 00:00:07.500\nWorld\n",
			want: "1\n00:00:01,000 --> 00:00:04,000\nHello\n\n2\n00:00:05,000 --> 00:00:07,500\nWorld\n\n",
		},
		{
			name: "cue identifiers and settings",
			content: "WEBVTT\n\n" +
				"cue-1\n00:01.000 --> 00:04.000 align:start position:0%\nHello\n\n",
			want: "1\n00:00:01,000 --> 00:00:04,000\nHello\n\n",
		},
		{
			name: "note blocks are skipped",
			content: "WEBVTT\n\n" +
				"NOTE this is a comment\nspanning lines\n\n" +
				"00:00:01.000 --> 00:00:02.000\nHi\n",
			want: "1\n00:00:01,000 --> 00:00:02,000\nHi\n\n",
		},
		{
			name: "multi line payload",
			content: "WEBVTT\n\n" +
				"00:00:01.000 --> 00:00:02.000\nline one\nline two\n",
			want: "1\n00:00:01,000 --> 00:00:02,000\nline one\nline two\n\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertVTTToSRT([]byte(tt.content))
			if err != nil {
				t.Fatalf("ConvertVTTToSRT() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("ConvertVTTToSRT() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConverterRegistry(t *testing.T) {
	for _, ext := range []string{"xml", "vtt", ".XML"} {
		if _, ok := ConverterFor(ext); !ok {
			t.Errorf("ConverterFor(%q) not found", ext)
		}
	}
	if _, ok := ConverterFor("ass"); ok {
		t.Errorf("ConverterFor(ass) should not be registered")
	}
}

func TestConvertFile(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "caption.xml")
	content := `<timedtext><body><p t="0" d="1000">Hello</p></body></timedtext>`
	if err := os.WriteFile(xmlPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	converter, ok := ConverterFor("xml")
	if !ok {
		t.Fatal("xml converter not registered")
	}
	srtPath, err := ConvertFile(xmlPath, converter)
	if err != nil {
		t.Fatalf("ConvertFile() error = %v", err)
	}
	if want := filepath.Join(dir, "caption.srt"); srtPath != want {
		t.Errorf("ConvertFile() path = %q, want %q", srtPath, want)
	}
	got, err := os.ReadFile(srtPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "1\n00:00:00,000 --> 00:00:01,000\nHello\n\n"
	if string(got) != want {
		t.Errorf("ConvertFile() content = %q, want %q", got, want)
	}
}
