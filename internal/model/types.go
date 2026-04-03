package model

type TestFormat struct {
	QuestionCount int     `json:"question_count" yaml:"question_count"`
	PassMarkPct   float64 `json:"pass_mark_pct" yaml:"pass_mark_pct"`
	TimeLimitSec  int     `json:"time_limit_sec" yaml:"time_limit_sec"`
}

type Topic struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type Question struct {
	ID           string   `json:"id" yaml:"id"`
	TopicID      string   `json:"topic_id" yaml:"topic_id"`
	Text         string   `json:"text" yaml:"text"`
	Choices      []string `json:"choices" yaml:"choices"`
	CorrectIndex int      `json:"correct_index" yaml:"correct_index"`
	Explanation  string   `json:"explanation,omitempty" yaml:"explanation,omitempty"`
}

type QuizPack struct {
	ID          string     `json:"id" yaml:"id"`
	Name        string     `json:"name" yaml:"name"`
	Description string     `json:"description" yaml:"description"`
	Version     string     `json:"version" yaml:"version"`
	TestFormat  TestFormat `json:"test_format" yaml:"test_format"`
	Topics      []Topic    `json:"topics" yaml:"topics"`
	Questions   []Question `json:"questions" yaml:"questions"`
}
