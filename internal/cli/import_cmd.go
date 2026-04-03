package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/sam-liem/quizbot/internal/importer"
)

// RunImport imports a quiz pack from a file. If format is empty, it is
// auto-detected from the file extension.
func (a *App) RunImport(filePath, format string, w io.Writer) error {
	ctx := context.Background()

	var detectedFormat importer.Format
	var err error

	if format != "" {
		detectedFormat = importer.Format(format)
	} else {
		detectedFormat, err = importer.DetectFormat(filePath)
		if err != nil {
			return fmt.Errorf("detecting format: %w", err)
		}
	}

	parser, err := importer.ParserForFormat(detectedFormat)
	if err != nil {
		return fmt.Errorf("getting parser: %w", err)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	pack, err := parser.Parse(f)
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}

	if err := importer.Validate(*pack); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if err := a.repo.SaveQuizPack(ctx, *pack); err != nil {
		return fmt.Errorf("saving pack: %w", err)
	}

	fmt.Fprintf(w, "Imported pack: %s (%s) - %d questions\n", pack.Name, pack.ID, len(pack.Questions))
	return nil
}
