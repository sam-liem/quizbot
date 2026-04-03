# QuizBot

Adaptive quiz preparation tool built in Go. See `SPEC.md` for full design.

## Quick Reference

- **Language**: Go
- **Database**: SQLite (via repository interface)
- **Messaging**: Telegram (via messenger interface)
- **Spec**: `SPEC.md`

## Project Structure

```
cmd/quizbot/       — entrypoint
internal/
  core/            — quiz engine, scheduler, spaced repetition, stats, notifications
  model/           — shared domain types
  store/           — repository interface + sqlite implementation
  messenger/       — messenger interface + telegram implementation
  importer/        — quiz pack parsers (yaml, json, markdown)
  explainer/       — explanation interface + static implementation
  security/        — sanitization, encryption
  cli/             — terminal commands
config/            — infrastructure config loading
```

## Build & Run

```sh
go build -o quizbot ./cmd/quizbot
go run ./cmd/quizbot
```

## Testing

```sh
go test ./...
go vet ./...
golangci-lint run
```

## Conventions

- All major subsystems use interfaces (Repository, Messenger, Explainer, QuestionParser)
- All repository methods are scoped by user ID, even in v1
- Dependencies are injected via constructors — no global mutable state
- Errors are wrapped with context: `fmt.Errorf("doing X: %w", err)`
- Table-driven tests for core logic
- Mock implementations of interfaces for isolated testing
- Integration tests use in-memory SQLite

## Security

- LLM output must always pass through the sanitizer before reaching any output channel
- API keys encrypted at rest (AES-256-GCM)
- Never log secrets or include them in error messages
- Validate all quiz pack imports against schema before persisting
- Validate answer submissions against the active session

## Config

- `config.yaml` — infrastructure config (gitignored)
- `config.example.yaml` — committed template
- User preferences stored in DB, modifiable at runtime via CLI or Telegram
