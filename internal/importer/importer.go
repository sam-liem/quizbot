package importer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sam-liem/quizbot/internal/model"
)

// Format identifies the serialisation format of a quiz pack file.
type Format string

const (
	FormatYAML     Format = "yaml"
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
)

// QuestionParser is implemented by each format-specific parser.
type QuestionParser interface {
	Parse(r io.Reader) (*model.QuizPack, error)
}

// DetectFormat returns the Format corresponding to the given filename's
// extension. It returns an error for unknown or missing extensions.
func DetectFormat(filename string) (Format, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".yaml", ".yml":
		return FormatYAML, nil
	case ".json":
		return FormatJSON, nil
	case ".md":
		return FormatMarkdown, nil
	default:
		if ext == "" {
			return "", fmt.Errorf("cannot detect format: %q has no file extension", filename)
		}
		return "", fmt.Errorf("unsupported file extension %q", ext)
	}
}

// ParserForFormat returns the QuestionParser that handles the given Format.
func ParserForFormat(format Format) (QuestionParser, error) {
	switch format {
	case FormatYAML:
		return &YAMLParser{}, nil
	case FormatJSON:
		return &JSONParser{}, nil
	case FormatMarkdown:
		return &MarkdownParser{}, nil
	default:
		return nil, fmt.Errorf("no parser for format %q", format)
	}
}

// ParseFile detects the format from the filename, opens the file, parses it,
// validates the result and returns the QuizPack (or an error).
func ParseFile(path string) (*model.QuizPack, error) {
	format, err := DetectFormat(path)
	if err != nil {
		return nil, fmt.Errorf("detecting format: %w", err)
	}

	parser, err := ParserForFormat(format)
	if err != nil {
		return nil, fmt.Errorf("getting parser: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file %q: %w", path, err)
	}
	defer f.Close()

	pack, err := parser.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("parsing %q: %w", path, err)
	}

	if err := Validate(*pack); err != nil {
		return nil, fmt.Errorf("validating %q: %w", path, err)
	}

	return pack, nil
}
