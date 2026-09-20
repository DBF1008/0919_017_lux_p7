package subtitle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testXMLCaption = `<timedtext><body><p t="0" d="1000">Hello</p></body></timedtext>`

func fetchFunc(body []byte, err error) FetchFunc {
	return func(url string) ([]byte, error) {
		return body, err
	}
}

func TestProcessorAddEmbed(t *testing.T) {
	processor := New(Options{
		Embed: true,
		Fetch: fetchFunc([]byte(testXMLCaption), nil),
	})
	if err := processor.Add("en", Source{URL: "http://example.com/caption.xml", Ext: "xml"}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	defer processor.Cleanup()

	tracks := processor.Tracks()
	if len(tracks) != 1 {
		t.Fatalf("Tracks() length = %d, want 1", len(tracks))
	}
	track := tracks[0]
	if track.Language != "en" {
		t.Errorf("track language = %q, want en", track.Language)
	}
	if !strings.HasSuffix(track.Path, ".srt") {
		t.Errorf("track path = %q, want a .srt file", track.Path)
	}
	content, err := os.ReadFile(track.Path)
	if err != nil {
		t.Fatalf("temp file should exist before cleanup: %v", err)
	}
	want := "1\n00:00:00,000 --> 00:00:01,000\nHello\n\n"
	if string(content) != want {
		t.Errorf("temp file content = %q, want %q", content, want)
	}
}

func TestProcessorAddWithoutEmbed(t *testing.T) {
	dir := t.TempDir()
	processor := New(Options{
		Embed: false,
		Fetch: fetchFunc([]byte("caption body"), nil),
		FilePath: func(ext string) (string, error) {
			return filepath.Join(dir, "caption."+ext), nil
		},
	})
	if err := processor.Add("en", Source{URL: "http://example.com/caption.srt", Ext: "srt"}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if len(processor.Tracks()) != 0 {
		t.Errorf("Tracks() should be empty when Embed is disabled")
	}
	content, err := os.ReadFile(filepath.Join(dir, "caption.srt"))
	if err != nil {
		t.Fatalf("caption file should be kept: %v", err)
	}
	if string(content) != "caption body" {
		t.Errorf("caption file content = %q", content)
	}
}

func TestProcessorAddTransform(t *testing.T) {
	processor := New(Options{
		Embed: true,
		Fetch: fetchFunc([]byte("raw"), nil),
	})
	source := Source{
		URL: "http://example.com/caption.json",
		Ext: "json",
		Transform: func(body []byte) ([]byte, error) {
			return []byte("transformed " + string(body)), nil
		},
	}
	if err := processor.Add("en", source); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	defer processor.Cleanup()

	content, err := os.ReadFile(processor.Tracks()[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "transformed raw" {
		t.Errorf("transform not applied, content = %q", content)
	}
}

type fakeStrategy struct {
	videoPath string
	tracks    []Track
	called    bool
}

func (s *fakeStrategy) Embed(videoPath string, tracks []Track) error {
	s.called = true
	s.videoPath = videoPath
	s.tracks = tracks
	return nil
}

func TestProcessorEmbedAndCleanup(t *testing.T) {
	strategy := &fakeStrategy{}
	processor := New(Options{
		Embed:    true,
		Fetch:    fetchFunc([]byte(testXMLCaption), nil),
		Strategy: strategy,
	})
	if err := processor.Add("en", Source{URL: "http://example.com/caption.xml", Ext: "xml"}); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	tempPath := processor.Tracks()[0].Path

	if err := processor.Embed("video.mp4"); err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if !strategy.called {
		t.Fatal("strategy.Embed() was not called")
	}
	if strategy.videoPath != "video.mp4" {
		t.Errorf("strategy videoPath = %q", strategy.videoPath)
	}
	if len(strategy.tracks) != 1 || strategy.tracks[0].Language != "en" {
		t.Errorf("strategy tracks = %+v", strategy.tracks)
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Errorf("temp file %q should be removed after Embed", tempPath)
	}
}

func TestProcessorEmbedWithoutTracks(t *testing.T) {
	strategy := &fakeStrategy{}
	processor := New(Options{
		Embed:    true,
		Fetch:    fetchFunc(nil, nil),
		Strategy: strategy,
	})
	if err := processor.Embed("video.mp4"); err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if strategy.called {
		t.Error("strategy.Embed() should not be called without tracks")
	}
}
