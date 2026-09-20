package subtitle

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// fakeConverter is a test converter uppercasing its input.
type fakeConverter struct{}

func (fakeConverter) FromExt() string { return "foo" }
func (fakeConverter) ToExt() string   { return "bar" }
func (fakeConverter) Convert(data []byte) ([]byte, error) {
	out := make([]byte, len(data))
	for i, b := range data {
		if b >= 'a' && b <= 'z' {
			b -= 'a' - 'A'
		}
		out[i] = b
	}
	return out, nil
}

func TestProcessorRegisterConverter(t *testing.T) {
	p := New()
	p.RegisterConverter(fakeConverter{})
	if !p.Convertible("foo") {
		t.Fatal("expected foo converter to be registered")
	}
	got, err := p.Convert("foo", []byte("abc"))
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if string(got) != "ABC" {
		t.Errorf("Convert() = %q, want %q", got, "ABC")
	}
}

func TestProcessorConvertFile(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "video.xml")
	content := `<timedtext><body><p t="0" d="1000">Hello</p></body></timedtext>`
	if err := os.WriteFile(xmlPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	p := New()
	srtPath, err := p.ConvertFile(xmlPath)
	if err != nil {
		t.Fatalf("ConvertFile() error = %v", err)
	}
	wantPath := filepath.Join(dir, "video.srt")
	if srtPath != wantPath {
		t.Errorf("ConvertFile() path = %s, want %s", srtPath, wantPath)
	}
	got, err := os.ReadFile(srtPath)
	if err != nil {
		t.Fatalf("converted file not written: %v", err)
	}
	want := "1\n00:00:00,000 --> 00:00:01,000\nHello\n\n"
	if string(got) != want {
		t.Errorf("converted content = %q, want %q", got, want)
	}
	// The converted file must be tracked for cleanup.
	tempFiles := p.TempFiles()
	if len(tempFiles) != 1 || tempFiles[0] != srtPath {
		t.Errorf("TempFiles() = %v, want [%s]", tempFiles, srtPath)
	}
}

func TestProcessorConvertFileUnknownExt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "video.xyz")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	p := New()
	if _, err := p.ConvertFile(path); err == nil {
		t.Fatal("expected error for unknown extension")
	}
}

func TestProcessorCleanup(t *testing.T) {
	dir := t.TempDir()
	p := New()
	var paths []string
	for i := 0; i < 3; i++ {
		path := filepath.Join(dir, fmt.Sprintf("sub%d.srt", i))
		if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		p.TrackFile(path)
		paths = append(paths, path)
	}
	// Tracking a non-existent file must not break Cleanup.
	p.TrackFile(filepath.Join(dir, "missing.srt"))

	if err := p.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	for _, path := range paths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed", path)
		}
	}
	if len(p.TempFiles()) != 0 {
		t.Error("expected temp file list to be reset after Cleanup")
	}
}

// fakeEmbedStrategy records the tracks it was asked to embed.
type fakeEmbedStrategy struct {
	called    bool
	videoPath string
	tracks    []Track
}

func (f *fakeEmbedStrategy) Name() string { return "fake" }
func (f *fakeEmbedStrategy) Embed(videoPath string, tracks []Track) error {
	f.called = true
	f.videoPath = videoPath
	f.tracks = tracks
	return nil
}

func TestProcessorEmbedStrategySelection(t *testing.T) {
	p := New()
	fake := &fakeEmbedStrategy{}
	p.RegisterEmbedStrategy("fake", fake)

	if err := p.SetEmbedStrategy("does-not-exist"); err == nil {
		t.Fatal("expected error for unknown strategy")
	}
	if err := p.SetEmbedStrategy("fake"); err != nil {
		t.Fatalf("SetEmbedStrategy() error = %v", err)
	}

	tracks := []Track{{Path: "a.srt", Lang: "en"}}
	if err := p.Embed("video.mp4", tracks); err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if !fake.called || fake.videoPath != "video.mp4" || len(fake.tracks) != 1 {
		t.Errorf("fake strategy not invoked correctly: %+v", fake)
	}
}
