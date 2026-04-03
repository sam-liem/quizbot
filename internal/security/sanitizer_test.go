package security_test

import (
	"strings"
	"testing"

	"github.com/sam-liem/quizbot/internal/security"
	"github.com/stretchr/testify/require"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "clean input passes through",
			input: "What is the capital of England?",
			want:  "What is the capital of England?",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "strip /start command",
			input: "/start the quiz now",
			want:  " the quiz now",
		},
		{
			name:  "strip /quiz command",
			input: "/quiz me",
			want:  " me",
		},
		{
			name:  "strip /resume command",
			input: "/resume",
			want:  "",
		},
		{
			name:  "strip /stats command",
			input: "/stats",
			want:  "",
		},
		{
			name:  "strip /packs command",
			input: "/packs list all",
			want:  " list all",
		},
		{
			name:  "strip /config command",
			input: "/config update",
			want:  " update",
		},
		{
			name:  "strip /help command",
			input: "/help me please",
			want:  " me please",
		},
		{
			name:  "command in middle of text",
			input: "Please /start the quiz now",
			want:  "Please  the quiz now",
		},
		{
			name:  "escape HTML less-than",
			input: "value < 10",
			want:  "value &lt; 10",
		},
		{
			name:  "escape HTML greater-than",
			input: "value > 10",
			want:  "value &gt; 10",
		},
		{
			name:  "escape HTML ampersand",
			input: "fish & chips",
			want:  "fish &amp; chips",
		},
		{
			name:  "escape HTML double quote",
			input: `say "hello"`,
			want:  "say &#34;hello&#34;",
		},
		{
			name:  "escape HTML single quote",
			input: "it's fine",
			want:  "it&#39;s fine",
		},
		{
			name:  "long string truncated to MaxOutputLength",
			input: strings.Repeat("a", security.MaxOutputLength+100),
			want:  strings.Repeat("a", security.MaxOutputLength),
		},
		{
			name:  "exact MaxOutputLength not truncated",
			input: strings.Repeat("b", security.MaxOutputLength),
			want:  strings.Repeat("b", security.MaxOutputLength),
		},
		{
			name:  "mixed: command + HTML + truncation",
			input: "/start <b>" + strings.Repeat("x", security.MaxOutputLength),
			// /start stripped → " <b>xxx..."
			// html escaped → " &lt;b&gt;xxx..."
			// then truncated to MaxOutputLength
			want: (" &lt;b&gt;" + strings.Repeat("x", security.MaxOutputLength))[:security.MaxOutputLength],
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := security.Sanitize(tc.input)
			require.Equal(t, tc.want, got)
		})
	}
}
