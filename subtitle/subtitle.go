// Package subtitle provides a unified subtitle processing pipeline:
// format conversion (via registered converters), embedding into video
// (via pluggable strategies) and temporary file lifecycle management.
package subtitle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// TransformFunc transforms raw caption bytes into the final subtitle
// content that should be written to disk.
type TransformFunc func([]byte) ([]byte, error)

// Converter converts subtitle data from one format to another.
type Converter interface {
	// FromExt is the source file extension this converter handles
	// (lowercase, without dot), e.g. "xml".
	FromExt() string
	// ToExt is the target file extension (lowercase, without dot), e.g. "srt".
	ToExt() string
	// Convert converts the raw subtitle data.
	Convert(data []byte) ([]byte, error)
}

// Track describes a single subtitle file to be embedded into a video.
type Track struct {
	Path string
	Lang string
}

// Processor coordinates subtitle conversion, embedding and the lifecycle
// of the temporary files produced along the way.
type Processor struct {
	mu           sync.Mutex
	converters   map[string]Converter
	strategies   map[string]EmbedStrategy
	embedderName string
	tempFiles    []string
}

// New returns a Processor with the built-in converters
// (XML->SRT, VTT->SRT) and the ffmpeg embed strategy registered.
func New() *Processor {
	p := &Processor{
		converters: make(map[string]Converter),
		strategies: make(map[string]EmbedStrategy),
	}
	p.RegisterConverter(NewXMLToSRTConverter())
	p.RegisterConverter(NewVTTToSRTConverter())
	p.RegisterEmbedStrategy(DefaultEmbedStrategy, NewFFmpegEmbedStrategy())
	p.embedderName = DefaultEmbedStrategy
	return p
}

// RegisterConverter registers a format converter, replacing any existing
// converter for the same source extension.
func (p *Processor) RegisterConverter(c Converter) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.converters[normalizeExt(c.FromExt())] = c
}

// Converter returns the converter registered for the given extension,
// or nil if there is none.
func (p *Processor) Converter(ext string) Converter {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.converters[normalizeExt(ext)]
}

// Convertible reports whether a converter is registered for the extension.
func (p *Processor) Convertible(ext string) bool {
	return p.Converter(ext) != nil
}

// Convert converts raw subtitle data from the given source extension.
func (p *Processor) Convert(ext string, data []byte) ([]byte, error) {
	c := p.Converter(ext)
	if c == nil {
		return nil, fmt.Errorf("no subtitle converter registered for extension %q", ext)
	}
	return c.Convert(data)
}

// ConvertFile converts the subtitle file at path using the converter
// registered for its extension and writes the result next to the source
// file with the target extension. The produced file is tracked as a
// temporary file and removed by Cleanup.
func (p *Processor) ConvertFile(path string) (string, error) {
	ext := normalizeExt(filepath.Ext(path))
	c := p.Converter(ext)
	if c == nil {
		return "", fmt.Errorf("no subtitle converter registered for extension %q", ext)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	converted, err := c.Convert(content)
	if err != nil {
		return "", err
	}
	outPath := replaceExt(path, c.ToExt())
	if err := os.WriteFile(outPath, converted, 0644); err != nil {
		return "", err
	}
	p.TrackFile(outPath)
	return outPath, nil
}

// TrackFile marks path as a temporary file to be removed by Cleanup.
func (p *Processor) TrackFile(path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tempFiles = append(p.tempFiles, path)
}

// TempFiles returns a copy of the currently tracked temporary files.
func (p *Processor) TempFiles() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	files := make([]string, len(p.tempFiles))
	copy(files, p.tempFiles)
	return files
}

// Cleanup removes all tracked temporary files. It keeps going on
// individual failures and reports the first error encountered.
func (p *Processor) Cleanup() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var firstErr error
	for _, path := range p.tempFiles {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}
	p.tempFiles = nil
	return firstErr
}

// normalizeExt normalizes an extension for map lookup: lowercase, no dot.
func normalizeExt(ext string) string {
	return strings.TrimPrefix(strings.ToLower(ext), ".")
}

// replaceExt replaces the extension of path with the given one
// (with or without leading dot).
func replaceExt(path, ext string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + "." + normalizeExt(ext)
}
