package subtitle

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestToISO639(t *testing.T) {
	tests := []struct {
		lang string
		want string
	}{
		{"en", "eng"},
		{"en-US", "eng"},
		{"zh_CN", "chi"},
		{"zh-Hans", "chi"},
		{"pt-BR", "por"},
		{"he", "heb"},
		{"id", "ind"},
		{"uk", "ukr"},
		{"vi", "vie"},
		{"th", "tha"},
		{"ENG", "eng"},    // already ISO 639-2
		{"jpn", "jpn"},    // already ISO 639-2
		{"xx", "und"},     // unknown 2-letter code
		{"", "und"},       // empty
		{"  FR  ", "fre"}, // whitespace tolerant
	}
	for _, tt := range tests {
		if got := ToISO639(tt.lang); got != tt.want {
			t.Errorf("ToISO639(%q) = %s, want %s", tt.lang, got, tt.want)
		}
	}
}

func TestSubtitleCodec(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{".mp4", "mov_text"},
		{".MP4", "mov_text"},
		{".webm", "webvtt"},
		{".mkv", ""},
	}
	for _, tt := range tests {
		if got := subtitleCodec(tt.ext); got != tt.want {
			t.Errorf("subtitleCodec(%q) = %q, want %q", tt.ext, got, tt.want)
		}
	}
}

func TestFFmpegEmbedStrategy(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "video.mp4")

	// Generate a 1-second test video.
	cmd := exec.Command(
		"ffmpeg", "-y", "-f", "lavfi", "-i", "color=c=black:s=64x64:d=1",
		"-pix_fmt", "yuv420p", videoPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to generate test video: %v\n%s", err, out)
	}

	srtPath := filepath.Join(dir, "video.srt")
	srt := "1\n00:00:00,000 --> 00:00:00,500\nHello\n\n"
	if err := os.WriteFile(srtPath, []byte(srt), 0644); err != nil {
		t.Fatal(err)
	}

	p := New()
	tracks := []Track{{Path: srtPath, Lang: "en-US"}}
	if err := p.Embed(videoPath, tracks); err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	// Verify the subtitle stream was muxed with the right language.
	probe := exec.Command(
		"ffprobe", "-v", "error", "-select_streams", "s:0",
		"-show_entries", "stream=codec_name:stream_tags=language",
		"-of", "csv=p=0", videoPath,
	)
	out, err := probe.Output()
	if err != nil {
		t.Fatalf("ffprobe failed: %v", err)
	}
	got := string(out)
	if want := "mov_text,eng"; got != want+"\n" {
		t.Errorf("subtitle stream = %q, want %q", got, want)
	}
}
