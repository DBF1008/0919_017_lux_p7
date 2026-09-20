package subtitle

import (
	"os"
	"path/filepath"
	"strings"
)

// TransformFunc transforms raw caption bytes before they are processed
// or written to disk. It matches the signature of the transform callback
// used by extractors.CaptionPart.
type TransformFunc func([]byte) ([]byte, error)

// Converter converts subtitle content from one format to another.
type Converter interface {
	// FromExt returns the source extension this converter handles, eg: "xml".
	FromExt() string
	// ToExt returns the target extension this converter produces, eg: "srt".
	ToExt() string
	// Convert converts subtitle content to the target format.
	Convert(content []byte) ([]byte, error)
}

var converters = make(map[string]Converter)

// RegisterConverter registers a Converter, replacing any existing converter
// registered for the same source extension.
func RegisterConverter(converter Converter) {
	converters[strings.ToLower(converter.FromExt())] = converter
}

// ConverterFor returns the Converter registered for the given extension.
func ConverterFor(ext string) (Converter, bool) {
	converter, ok := converters[strings.ToLower(strings.TrimPrefix(ext, "."))]
	return converter, ok
}

// ConvertFile converts the subtitle file at path with the given converter and
// writes the result next to it, returning the path of the converted file.
func ConvertFile(path string, converter Converter) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	converted, err := converter.Convert(content)
	if err != nil {
		return "", err
	}
	convertedPath := strings.TrimSuffix(path, filepath.Ext(path)) + "." + converter.ToExt()
	return convertedPath, os.WriteFile(convertedPath, converted, 0644)
}

func init() {
	RegisterConverter(xmlToSRTConverter{})
	RegisterConverter(vttToSRTConverter{})
}
