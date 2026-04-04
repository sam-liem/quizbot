# QuizBot

Adaptive quiz preparation tool with spaced repetition, Telegram delivery, and CLI. Built as a single Go binary with clean interface boundaries.

The first target is the **Life in the UK test** (24 multiple-choice questions, 75% pass mark, 45 minutes), but the system is quiz-agnostic -- any test can be loaded as a self-contained quiz pack.

## Features

- **SM-2 spaced repetition** with topic-weighted question selection
- **Three quiz pack formats**: YAML, JSON, Markdown with schema validation
- **Telegram bot**: inline keyboard answers, scheduled delivery, bot commands
- **CLI**: interactive quiz, stats, import, pack management, config
- **Notifications**: inactivity reminders, weak topic alerts, milestones, streaks, readiness
- **Resumable sessions**: interrupt and pick up where you left off
- **Progress tracking**: readiness score, topic breakdown, question history
- **Security**: AES-256-GCM key encryption, LLM output sanitizer

## Quick Start

```sh
# Build
go build -o quizbot ./cmd/quizbot

# Copy and edit config
cp config.example.yaml config.yaml
# Fill in your telegram_bot_token, telegram_chat_id, and encryption_key

# Import a quiz pack
./quizbot import path/to/pack.yaml

# Start an interactive quiz
./quizbot quiz start --pack life-in-uk

# Start a mock test
./quizbot quiz start --pack life-in-uk --mock

# View stats
./quizbot stats --pack life-in-uk

# Run as daemon (Telegram bot + scheduled delivery)
./quizbot
```

## Quiz Pack Format

Quiz packs can be authored in YAML, JSON, or Markdown. Example YAML:

```yaml
id: "life-in-uk"
name: "Life in the UK"
description: "Official Life in the UK Test preparation"
version: "1.0.0"
test_format:
  question_count: 24
  pass_mark_pct: 75.0
  time_limit_sec: 2700

topics:
  - id: history
    name: "History of the UK"

questions:
  - id: q001
    topic_id: history
    text: "When was the Magna Carta sealed?"
    choices:
      - "1205"
      - "1210"
      - "1215"
      - "1220"
    correct_index: 2
    explanation: "The Magna Carta was sealed by King John in 1215 at Runnymede."
```

## CLI Commands

```
quizbot quiz start [--pack X] [--topic Y] [--count N] [--mock]
quizbot quiz resume
quizbot stats [--pack X] [--topic Y] [--detailed]
quizbot import <file> [--format yaml|json|md]
quizbot packs list
quizbot packs activate <pack-id>
quizbot packs deactivate <pack-id>
quizbot config get <key>
quizbot config set <key> <value>
quizbot config list
```

## Telegram Bot Commands

| Command | Description |
|---------|-------------|
| `/start` | Initialize and set up preferences |
| `/quiz` | Start a quiz (`--mock`, `--topic X`, `--count N`) |
| `/resume` | Resume an interrupted session |
| `/stats` | Show progress summary |
| `/packs` | List and manage quiz packs |
| `/config` | View and modify preferences |
| `/help` | List available commands |

## Configuration

Infrastructure config (`config.yaml`, gitignored):

```yaml
telegram_bot_token: "YOUR_TOKEN"
telegram_chat_id: "YOUR_CHAT_ID"
sqlite_path: "quizbot.db"
encryption_key: "YOUR_32_BYTE_HEX_KEY"
listen_address: ":8080"
log_level: "info"
```

User preferences are stored in the database and modifiable at runtime via `quizbot config set` or Telegram `/config`.

## Development

```sh
# Run tests
go test ./...

# Run tests with race detector
go test -race ./...

# Vet
go vet ./...

# Lint
golangci-lint run
```

## Architecture

Single Go binary with well-separated internal packages. All major subsystems communicate through interfaces:

- **Repository** -- data access (SQLite in v1, Postgres for SaaS)
- **Messenger** -- communication platform (Telegram in v1)
- **Explainer** -- answer explanations (static in v1, LLM in v2)
- **QuestionParser** -- quiz pack import (YAML, JSON, Markdown)

See [SPEC.md](SPEC.md) for the full design document.

## License

MIT
