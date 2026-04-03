package importer

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/sam-liem/quizbot/internal/model"
)

// JSONParser implements QuestionParser for JSON-formatted quiz packs.
type JSONParser struct{}

// Parse decodes a JSON-encoded QuizPack from r.
func (p *JSONParser) Parse(r io.Reader) (*model.QuizPack, error) {
	var pack model.QuizPack
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&pack); err != nil {
		return nil, fmt.Errorf("decoding json: %w", err)
	}
	return &pack, nil
}
