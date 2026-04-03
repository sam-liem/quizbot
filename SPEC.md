# QuizBot — Adaptive Quiz Preparation Tool

## Overview

A generalizable quiz preparation tool that helps users study for any test through spaced repetition, progress tracking, and multi-channel delivery. Built as a single Go binary with clean interface boundaries designed for future SaaS expansion.

The first target is the Life in the UK test(24 multiple choice questions), but the system is quiz-agnostic — any test can be loaded as a self-contained quiz pack.

## Constraints

- **Language**: Go
- **Single user for v1**, but all data access scoped by user ID for SaaS readiness
- **Minimize cloud cost**: single binary, SQLite, free-tier cloud instance
- **Open source / BYO-key model**: users provide their own LLM API key and Telegram bot token
- **Clean interfaces**: all major subsystems behind interfaces for testability and future extensibility
- **Security-first**: sanitize all LLM output, encrypt API keys at rest, validate all input

---

## Data Model

### Quiz Pack

The self-contained unit of content. Portable and shareable.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier |
| name | string | Display name (e.g., "Life in the UK") |
| description | string | What this quiz prepares you for |
| version | string | Pack version for updates |
| test_format | TestFormat | Mock test parameters |
| topics | []Topic | List of topics in this pack |
| questions | []Question | All questions in this pack |

**TestFormat**:

| Field | Type | Description |
|-------|------|-------------|
| question_count | int | Number of questions in a mock test |
| pass_mark_pct | float | Percentage required to pass |
| time_limit_sec | int | Time limit in seconds (0 = untimed) |

**Topic**:

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique within the pack |
| name | string | Display name |
| description | string | Optional description |

**Question**:

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique within the pack |
| topic_id | string | Reference to parent topic |
| text | string | The question text |
| choices | []string | Ordered list of answer options |
| correct_index | int | Zero-based index of correct answer |
| explanation | string | Optional static explanation of the correct answer |

### Question State (Spaced Repetition)

Per-user, per-question scheduling metadata.

| Field | Type | Description |
|-------|------|-------------|
| user_id | string | Owner |
| question_id | string | Reference to question |
| pack_id | string | Reference to quiz pack |
| ease_factor | float | SM-2 ease factor (default 2.5, floor 1.3) |
| interval_days | float | Days until next review |
| repetition_count | int | Times reviewed |
| next_review_at | timestamp | When this question is next due |
| last_result | enum | correct / wrong / skipped |
| last_reviewed_at | timestamp | When last attempted |

### Topic Stats

Per-user, per-topic, per-quiz-pack aggregate statistics.

| Field | Type | Description |
|-------|------|-------------|
| user_id | string | Owner |
| pack_id | string | Reference to quiz pack |
| topic_id | string | Reference to topic |
| total_attempts | int | Total questions answered in this topic |
| correct_count | int | Total correct answers |
| rolling_accuracy | float | Accuracy over last N attempts |
| current_streak | int | Consecutive correct answers |
| best_streak | int | All-time best streak |

### Study Session

A record of a study sitting.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier |
| user_id | string | Owner |
| pack_id | string | Quiz pack studied |
| mode | enum | mock / practice / scheduled |
| started_at | timestamp | When the session began |
| ended_at | timestamp | When the session ended (null if active) |
| question_count | int | Total questions in this session |
| correct_count | int | Correct answers so far |
| attempts | []QuestionAttempt | Individual question results |

**QuestionAttempt**:

| Field | Type | Description |
|-------|------|-------------|
| question_id | string | Which question |
| answer_index | int | What the user chose |
| correct | bool | Whether they got it right |
| time_taken_ms | int | How long they took to answer |
| answered_at | timestamp | When they answered |

### Quiz Session (Resumable)

Tracks an in-progress quiz for resume support.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Unique identifier |
| user_id | string | Owner |
| pack_id | string | Quiz pack |
| mode | enum | mock / practice |
| question_ids | []string | Ordered list of questions in this quiz |
| current_index | int | Where the user is |
| answers | map[string]int | question_id → chosen answer index |
| started_at | timestamp | When the quiz started |
| time_limit_sec | int | Time limit (0 = untimed) |
| status | enum | in_progress / completed / abandoned |

### User Preferences (stored in DB, runtime-modifiable)

| Field | Type | Description |
|-------|------|-------------|
| user_id | string | Owner |
| delivery_interval_min | int | Minutes between scheduled questions |
| max_unanswered | int | Max pending questions before pausing delivery (default 3) |
| active_pack_ids | []string | Which packs are currently active |
| focus_mode | enum | single / interleaved |
| notify_inactivity | bool | Inactivity reminders (default true) |
| notify_inactivity_days | int | Days before first reminder (default 2) |
| notify_weak_topic | bool | Weak topic alerts (default true) |
| notify_weak_topic_pct | float | Accuracy threshold for alert (default 50%) |
| notify_milestones | bool | Milestone celebrations (default true) |
| notify_readiness | bool | Weekly readiness summary (default true) |
| notify_streak | bool | Streak warnings (default true) |
| quiet_hours_start | string | Quiet hours start, 24h format (default "22:00") |
| quiet_hours_end | string | Quiet hours end, 24h format (default "08:00") |

### Infrastructure Config (file, not in DB, gitignored)

| Field | Type | Description |
|-------|------|-------------|
| telegram_bot_token | string | Telegram Bot API token |
| sqlite_path | string | Path to SQLite database file |
| llm_api_key | string | LLM API key (encrypted at rest in DB for SaaS; plaintext in config file for self-hosted) |
| listen_address | string | Address for any HTTP endpoints |
| log_level | string | Logging verbosity |
| encryption_key | string | Key for encrypting secrets in the database |

---

## Architecture

### Approach: Monolith with Clean Internal Boundaries

Single Go binary with well-separated internal packages. All major subsystems communicate through interfaces, not concrete types. Both the Telegram bot and CLI are thin adapters over a shared core engine.

### Core Interfaces

**Repository** — data access layer:
```
All methods take a user context for tenant scoping.

- SaveQuizPack / GetQuizPack / ListQuizPacks
- GetQuestionState / UpdateQuestionState
- GetTopicStats / UpdateTopicStats
- CreateSession / GetSession / UpdateSession
- GetQuizSession / SaveQuizSession
- GetPreferences / UpdatePreferences
- GetQuestionHistory (with filters and sorting)
```
v1: SQLite implementation. SaaS: swap to Postgres.

**Messenger** — communication platform abstraction:
```
- SendQuestion(chatID, question, choices) → messageID
- SendFeedback(chatID, result, explanation)
- SendStats(chatID, stats)
- SendNotification(chatID, notification)
- ListenForAnswers() → channel of (chatID, messageID, answer)
- ListenForCommands() → channel of (chatID, command, args)
```
v1: Telegram implementation. Future: Discord, Signal, Slack, WhatsApp.

**QuestionParser** — quiz pack import:
```
- Parse(reader) → QuizPack
```
Three implementations: YAML, JSON, Markdown. Auto-detected by file extension with format flag override.

**Explainer** — answer explanations:
```
- Explain(question, userAnswer, correctAnswer) → string
```
v1: StaticExplainer returns pre-authored explanation. v2: LLMExplainer calls an LLM API with the user's own API key. Output always sanitized before reaching any messenger or CLI.

---

## Core Engine

### Quiz Engine

- `StartQuiz(packID, mode, options) → QuizSession` — creates a session. Mock mode uses the pack's test format. Practice mode accepts custom question count and optional topic filter.
- `NextQuestion(sessionID) → Question` — returns the next question. Practice mode respects spaced repetition priority. Mock mode uses random selection.
- `SubmitAnswer(sessionID, questionID, answerIndex) → AnswerResult` — scores the answer, returns correct/wrong with explanation, updates spaced repetition state and topic stats.
- `GetSessionStatus(sessionID) → SessionStatus` — current progress, score, time remaining.
- `ResumeSession(sessionID) → QuizSession` — resumes an interrupted session.
- `AbandonSession(sessionID)` — marks as abandoned, records attempts made so far.

### Spaced Repetition (SM-2 Algorithm)

- **Correct answer**: increase interval based on ease factor, maintain or increase ease factor.
- **Wrong answer**: reset interval to 1 day, decrease ease factor (floor of 1.3).
- **Question selection**: pick the question with the most overdue review date, weighted toward weak topics.
- New (unseen) questions are introduced gradually, mixed with review questions.

### Scheduler

- Runs as a goroutine inside the main process.
- Checks delivery interval from user preferences.
- Selects next question via spaced repetition.
- Delivers through the Messenger interface.
- Tracks pending (unanswered) questions — pauses delivery when the count hits `max_unanswered`.
- Unanswered questions are marked as skipped after a configurable timeout.

### Stats Engine

- **Readiness score**: estimated pass probability based on topic accuracies weighted by question distribution in the pack.
- **Topic breakdown**: per-topic accuracy, trend direction, question count, weakest areas highlighted.
- **Question history**: drill-down into individual question attempts with filters (topic, date range, result) and sorting (by accuracy, by next review date, by attempts).
- **Session history**: past study sessions with duration, mode, score, and date.

---

## Notifications and Engagement

All enabled by default (Duolingo-style). Each type individually toggleable via preferences.

### Notification Types

| Type | Trigger | Frequency Cap | Default |
|------|---------|---------------|---------|
| Inactivity Reminder | No study in N days (configurable, default 2) | Once per day, escalating tone | On |
| Weak Topic Alert | Rolling accuracy on a topic drops below threshold (default 50%) | Once per topic per day | On |
| Milestone Celebration | Round question counts, topic mastery thresholds, streak milestones | No cap (positive reinforcement) | On |
| Readiness Update | Weekly progress summary with estimated pass rate and weak topics | Once per week (configurable) | On |
| Streak Warning | Haven't studied today and streak is at risk | Once per day | On |

### Delivery Rules

- All notifications go through the Messenger interface.
- The scheduler checks notification conditions on a regular tick (hourly).
- Quiet hours respected (default 10pm–8am, configurable). Notifications queued and delivered after quiet hours end.

---

## Telegram Integration

### Interaction Model

- Questions sent with inline keyboard buttons (A/B/C/D) for tap-to-answer.
- Text replies also accepted for future free-text question types.
- Bot commands:
  - `/start` — initialize, set up preferences
  - `/quiz` — start a pop quiz (options: `--mock`, `--topic X`, `--count N`)
  - `/resume` — resume an interrupted session
  - `/stats` — show progress summary
  - `/packs` — list and manage quiz packs
  - `/config` — view and modify preferences
  - `/help` — list available commands

### Scheduled Delivery

- Bot sends a question at the configured interval.
- Waits for an answer (via button tap or text reply).
- If unanswered, queues the next question up to `max_unanswered` (default 3), then pauses.
- On answer: sends feedback, updates stats, resumes delivery if paused.

---

## CLI Interface

Direct caller of the core engine. Does not implement the Messenger interface — it's a separate entry path.

### Commands

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

### Interactive Quiz

- Questions rendered in the terminal with numbered choices.
- User types the number to answer.
- Immediate feedback with explanation.
- Progress bar for mock tests. Timer display if timed.
- Ctrl+C gracefully saves session for later resume.

The CLI and Telegram bot run simultaneously in the same process. The CLI talks directly to the core engine. The bot goroutine handles Telegram traffic concurrently.

---

## Quiz Pack Format

Self-contained quiz packages supporting multiple authoring formats: YAML, JSON, and Markdown with frontmatter.

### YAML Example

```yaml
name: "Life in the UK"
description: "Official Life in the UK Test preparation"
version: "1.0.0"
test_format:
  question_count: 24
  pass_mark_pct: 75.0
  time_limit_sec: 2700  # 45 minutes

topics:
  - id: history
    name: "History of the UK"
  - id: values
    name: "British Values"

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

### JSON and Markdown

JSON follows the same schema. Markdown uses YAML frontmatter for pack metadata and a structured format for questions within the file. All formats are normalized to the same internal representation on import.

Import validates against the schema before persisting — malformed or oversized packs are rejected with descriptive errors.

---

## Security

### LLM Output Sanitization (v2, designed now)

- All LLM responses pass through a sanitizer before reaching any messenger or CLI output.
- Strip or escape formatting characters that could break Telegram HTML/Markdown rendering.
- Reject responses containing bot command patterns (`/start`, `/quiz`, etc.) to prevent command injection.
- Truncate to a max length to prevent token-stuffing.
- The Explainer interface returns sanitized output — callers never see raw LLM text.

### API Key Storage

- User-provided LLM API keys encrypted at rest in the database (AES-256-GCM).
- Encryption key derived from a secret in the infrastructure config file.
- Keys never logged, never included in error messages, never returned in stats or API responses.

### Telegram Security

- Validate incoming updates using Telegram's secret token mechanism.
- Restrict bot to authorized chat IDs (configurable allowlist in v1, per-user auth in SaaS).
- Rate limit incoming messages to prevent abuse.

### Input Validation

- Quiz pack imports validated against schema before persisting.
- Answer submissions validated against the active session's expected question.
- CLI import paths validated against path traversal.

### Tenant Isolation (SaaS-ready)

- All repository methods scoped by user ID, even in v1 (single user).
- No shared mutable state between user contexts.
- Repository interface enforces tenant scoping at the type level.

---

## Multi-Quiz Management

- One or more quiz packs can be active simultaneously.
- **Single focus mode**: one pack active, all questions come from it.
- **Interleaved mode**: multiple packs active, scheduler rotates between them. Spaced repetition runs independently per pack.
- User switches mode and active packs via CLI or Telegram commands.
- Stats are always scoped per-pack. A global summary view aggregates across active packs.

---

## Session Management

### Quiz Sessions (Resumable)

- Starting a quiz creates a QuizSession with the full ordered question list.
- Progress is saved after each answer.
- If interrupted (Ctrl+C in CLI, user goes silent in Telegram), the session persists as in_progress.
- `resume` command picks up from the last unanswered question.
- Timed sessions track elapsed time — timer continues from where it left off.
- Sessions older than a configurable threshold (default 24 hours) are auto-abandoned.

### Study Sessions (Historical)

- Every interaction (quiz, practice, scheduled question) is recorded in a study session.
- Tracks start/end time, question count, accuracy, mode, and individual attempts.
- Viewable in stats as a session log — "Today: 2 sessions, 35 questions, 74% accuracy."

---

## Configuration

### Split Config Model

**Infrastructure config** (file: `config.yaml`, gitignored):
- Telegram bot token
- SQLite database path
- Encryption key for secrets
- Listen address
- Log level

Template committed as `config.example.yaml`.

**User preferences** (in database, runtime-modifiable):
- Delivery interval, max unanswered queue, notification toggles and thresholds, active packs, focus mode, quiet hours.
- Modifiable via CLI (`quizbot config set`) or Telegram (`/config`).
- Changes take effect immediately — no restart required.

---

## Project Structure

```
cmd/
  quizbot/                  — main entrypoint, dependency wiring
internal/
  core/
    engine.go               — quiz engine (start, answer, resume)
    scheduler.go            — question delivery scheduling
    repetition.go           — SM-2 spaced repetition algorithm
    stats.go                — readiness score, topic breakdown, history
    notifier.go             — notification condition checks and triggers
  model/                    — shared domain types (QuizPack, Question, Session, etc.)
  store/
    repository.go           — interface definitions
    sqlite/                 — SQLite implementation
    migrations/             — schema migrations
  messenger/
    messenger.go            — interface definition
    telegram/               — Telegram bot implementation
  importer/
    importer.go             — parser interface
    yaml.go
    json.go
    markdown.go
  explainer/
    explainer.go            — interface definition
    static.go               — v1: authored explanation lookup
  security/
    sanitizer.go            — LLM output sanitization
    crypto.go               — API key encryption/decryption
  cli/
    root.go                 — CLI entrypoint and subcommands
    quiz.go                 — interactive quiz commands
    stats.go                — stats display
    import.go               — quiz pack import
    config.go               — preference management
config/
  config.go                 — infrastructure config loading
  config.example.yaml       — committed template
```

---

## Testing Strategy

- **Unit tests**: every package gets `_test.go` files. Core logic (spaced repetition, scoring, stats calculation) tested thoroughly with table-driven tests.
- **Interface mocks**: Repository, Messenger, Explainer interfaces have mock implementations for isolated testing.
- **Integration tests**: SQLite repository tested against real in-memory SQLite. Importer tested against fixture files in `testdata/` directories.
- **CLI tests**: command functions called directly with captured stdout.
- **Telegram adapter tests**: tested against mock HTTP responses, no real API calls.
- **CI**: GitHub Actions running `go test ./...`, `go vet`, and `golangci-lint` on every push.

### Code Quality

- `golangci-lint` with reasonable defaults.
- All exported types and interfaces have doc comments.
- No global mutable state — dependencies injected via constructors.
- Errors wrapped with context: `fmt.Errorf("doing X: %w", err)`.

---

## v1 Scope

What ships first:

- Manual question authoring (YAML/JSON/Markdown import)
- Multiple choice questions only
- Static explanations (pre-authored)
- SM-2 spaced repetition with topic-weighted question selection
- Telegram bot with inline keyboard buttons and scheduled delivery
- CLI with interactive quiz, stats, and import
- SQLite storage with repository interface
- Progress tracking: readiness score, topic breakdown, question history
- Notifications: inactivity, weak topics, milestones, streaks, readiness
- Resumable quiz sessions and study session logging
- Single user, single instance

## v2 and Beyond

- LLM-generated explanations (BYO API key, output sanitized)
- Additional question types (true/false, free-text with LLM judging, select-all-that-apply)
- LLM-generated questions from source material
- Additional messenger adapters (Discord, Signal, Slack)
- Multi-user / SaaS (Postgres, per-user auth, tenant isolation)
- Web dashboard for stats visualization
