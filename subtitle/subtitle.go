// Package subtitle abstracts subtitle downloading, format conversion,
// embedding and temporary file lifecycle management out of the downloader.
package subtitle

import (
	"os"
	"strings"
)

// Source describes a remote caption to be processed.
type Source struct {
	URL       string
	Ext       string
	Transform TransformFunc
}

// FetchFunc downloads a URL and returns its body.
type FetchFunc func(url string) ([]byte, error)

// FilePathFunc resolves the final output path for a caption file
// with the given extension.
type FilePathFunc func(ext string) (string, error)

// Options defines options used by a Processor.
type Options struct {
	// Embed converts downloaded captions and keeps them in temporary
	// files managed by the Processor, ready to be embedded.
	Embed bool
	// Fetch downloads a caption URL, required.
	Fetch FetchFunc
	// FilePath resolves the final caption path, required when Embed is false.
	FilePath FilePathFunc
	// Strategy embeds the subtitles, defaults to FFmpegStrategy.
	Strategy EmbedStrategy
}

// Processor downloads, converts and embeds subtitles, managing the
// lifecycle of all temporary files it creates.
type Processor struct {
	option    Options
	tracks    []Track
	tempFiles []string
}

// New returns a new Processor.
func New(option Options) *Processor {
	return &Processor{
		option: option,
	}
}

// Tracks returns the subtitle tracks registered so far.
func (p *Processor) Tracks() []Track {
	return p.tracks
}

// Add downloads a caption, applies its transform, converts it to SRT when a
// converter is registered for its format, and registers the resulting track.
// When Embed is disabled the caption is written to its final path instead
// and kept on disk.
func (p *Processor) Add(language string, source Source) error {
	body, err := p.option.Fetch(source.URL)
	if err != nil {
		return err
	}
	if source.Transform != nil {
		body, err = source.Transform(body)
		if err != nil {
			return err
		}
	}

	ext := strings.TrimPrefix(strings.ToLower(source.Ext), ".")
	if !p.option.Embed {
		path, err := p.option.FilePath(source.Ext)
		if err != nil {
			return err
		}
		return os.WriteFile(path, body, 0644)
	}

	if converter, ok := ConverterFor(ext); ok {
		if converted, err := converter.Convert(body); err == nil {
			body = converted
			ext = converter.ToExt()
		}
	}
	path, err := p.writeTemp(ext, body)
	if err != nil {
		return err
	}
	p.tracks = append(p.tracks, Track{
		Path:     path,
		Language: language,
	})
	return nil
}

// Embed embeds all registered tracks into the video and cleans up the
// temporary files afterwards.
func (p *Processor) Embed(videoPath string) error {
	defer p.Cleanup()
	if len(p.tracks) == 0 {
		return nil
	}
	strategy := p.option.Strategy
	if strategy == nil {
		strategy = FFmpegStrategy{}
	}
	return strategy.Embed(videoPath, p.tracks)
}

// Cleanup removes all temporary files created by the Processor.
func (p *Processor) Cleanup() {
	for _, path := range p.tempFiles {
		os.Remove(path) // nolint
	}
	p.tempFiles = nil
}

// writeTemp writes content to a temporary file tracked by the Processor.
func (p *Processor) writeTemp(ext string, content []byte) (string, error) {
	file, err := os.CreateTemp("", "lux-caption-*."+ext)
	if err != nil {
		return "", err
	}
	defer file.Close() // nolint

	if _, err := file.Write(content); err != nil {
		os.Remove(file.Name()) // nolint
		return "", err
	}
	p.tempFiles = append(p.tempFiles, file.Name())
	return file.Name(), nil
}
