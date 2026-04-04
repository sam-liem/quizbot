package config_test

import (
	"os"
	"testing"

	"github.com/sam-liem/quizbot/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func TestLoadConfig_Valid(t *testing.T) {
	yaml := `
telegram_bot_token: "bot123"
sqlite_path: "test.db"
llm_api_key: "llmkey"
listen_address: ":9090"
log_level: "debug"
encryption_key: "deadbeefdeadbeefdeadbeefdeadbeef"
`
	path := writeTempYAML(t, yaml)

	cfg, err := config.LoadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, "bot123", cfg.TelegramBotToken)
	assert.Equal(t, "test.db", cfg.SQLitePath)
	assert.Equal(t, "llmkey", cfg.LLMAPIKey)
	assert.Equal(t, ":9090", cfg.ListenAddress)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "deadbeefdeadbeefdeadbeefdeadbeef", cfg.EncryptionKey)
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := config.LoadConfig("/nonexistent/path/config.yaml")
	require.Error(t, err)
}

func TestLoadConfig_Defaults(t *testing.T) {
	yaml := `
telegram_bot_token: "bot456"
sqlite_path: "app.db"
encryption_key: "cafebabecafebabecafebabecafebabe"
`
	path := writeTempYAML(t, yaml)

	cfg, err := config.LoadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, ":8080", cfg.ListenAddress)
}

func TestLoadConfig_MissingRequired(t *testing.T) {
	yaml := `
sqlite_path: "app.db"
encryption_key: "cafebabecafebabecafebabecafebabe"
`
	path := writeTempYAML(t, yaml)

	_, err := config.LoadConfig(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "telegram_bot_token")
}

func TestLoadConfig_MissingEncryptionKey(t *testing.T) {
	yaml := `
telegram_bot_token: "bot789"
sqlite_path: "app.db"
`
	path := writeTempYAML(t, yaml)

	_, err := config.LoadConfig(path)
	require.Error(t, err)
}
