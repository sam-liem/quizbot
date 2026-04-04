package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sam-liem/quizbot/internal/importer"
)

// validateImportPath checks the given path for path traversal attempts and
// confirms it refers to a regular file (not a symlink or directory).
func validateImportPath(filePath string) error {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Detect ".." components in the cleaned path. filepath.Clean collapses
	// redundant separators and dots but preserves ".." components that are
	// needed to resolve the path. Rejecting any ".." in the cleaned input
	// catches paths like "../../etc/passwd".
	cleaned := filepath.Clean(filePath)
	for _, part := range strings.Split(cleaned, string(filepath.Separator)) {
		if part == ".." {
			return fmt.Errorf("path traversal detected: path must not contain '..' components")
		}
	}

	info, err := os.Lstat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}
		return fmt.Errorf("stat file: %w", err)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("path traversal detected: %s is not a regular file", filePath)
	}

	return nil
}

// RunImport imports a quiz pack from a file. If format is empty, it is
// auto-detected from the file extension.
func (a *App) RunImport(filePath, format string, w io.Writer) error {
	ctx := context.Background()

	if err := validateImportPath(filePath); err != nil {
		return err
	}

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
	defer func() { _ = f.Close() }()

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

	_, _ = fmt.Fprintf(w, "Imported pack: %s (%s) - %d questions\n", pack.Name, pack.ID, len(pack.Questions))
	return nil
}
