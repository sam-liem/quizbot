package importer

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/sam-liem/quizbot/internal/model"
)

// YAMLParser implements QuestionParser for YAML-formatted quiz packs.
type YAMLParser struct{}

// Parse decodes a YAML-encoded QuizPack from r.
func (p *YAMLParser) Parse(r io.Reader) (*model.QuizPack, error) {
	var pack model.QuizPack
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	if err := dec.Decode(&pack); err != nil {
		return nil, fmt.Errorf("decoding yaml: %w", err)
	}
	return &pack, nil
}
