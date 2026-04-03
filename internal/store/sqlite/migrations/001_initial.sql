-- Quiz packs: the self-contained unit of quiz content stored as a JSON blob.
CREATE TABLE IF NOT EXISTS quiz_packs (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    version     TEXT NOT NULL DEFAULT '',
    data        TEXT NOT NULL  -- full JSON-encoded QuizPack
);

-- Per-user, per-question spaced-repetition scheduling metadata.
CREATE TABLE IF NOT EXISTS question_states (
    user_id          TEXT NOT NULL,
    pack_id          TEXT NOT NULL,
    question_id      TEXT NOT NULL,
    ease_factor      REAL    NOT NULL DEFAULT 2.5,
    interval_days    REAL    NOT NULL DEFAULT 0,
    repetition_count INTEGER NOT NULL DEFAULT 0,
    next_review_at   TEXT    NOT NULL DEFAULT '',
    last_result      TEXT    NOT NULL DEFAULT '',
    last_reviewed_at TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (user_id, pack_id, question_id)
);

CREATE INDEX IF NOT EXISTS idx_question_states_review
    ON question_states (user_id, pack_id, next_review_at);

-- Per-user, per-topic aggregate statistics.
CREATE TABLE IF NOT EXISTS topic_stats (
    user_id          TEXT    NOT NULL,
    pack_id          TEXT    NOT NULL,
    topic_id         TEXT    NOT NULL,
    total_attempts   INTEGER NOT NULL DEFAULT 0,
    correct_count    INTEGER NOT NULL DEFAULT 0,
    rolling_accuracy REAL    NOT NULL DEFAULT 0,
    current_streak   INTEGER NOT NULL DEFAULT 0,
    best_streak      INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, pack_id, topic_id)
);

-- Records of completed or in-progress study sittings.
CREATE TABLE IF NOT EXISTS study_sessions (
    id             TEXT    PRIMARY KEY,
    user_id        TEXT    NOT NULL,
    pack_id        TEXT    NOT NULL,
    mode           TEXT    NOT NULL,
    started_at     TEXT    NOT NULL,
    ended_at       TEXT,   -- nullable
    question_count INTEGER NOT NULL DEFAULT 0,
    correct_count  INTEGER NOT NULL DEFAULT 0,
    attempts       TEXT    NOT NULL DEFAULT '[]'  -- JSON array of QuestionAttempt
);

CREATE INDEX IF NOT EXISTS idx_study_sessions_user
    ON study_sessions (user_id, started_at DESC);

-- Active quiz state (live session being answered).
CREATE TABLE IF NOT EXISTS quiz_sessions (
    id             TEXT    PRIMARY KEY,
    user_id        TEXT    NOT NULL,
    pack_id        TEXT    NOT NULL,
    mode           TEXT    NOT NULL,
    question_ids   TEXT    NOT NULL DEFAULT '[]',  -- JSON array of string IDs
    current_index  INTEGER NOT NULL DEFAULT 0,
    answers        TEXT    NOT NULL DEFAULT '{}',  -- JSON map[string]int
    started_at     TEXT    NOT NULL,
    time_limit_sec INTEGER NOT NULL DEFAULT 0,
    status         TEXT    NOT NULL DEFAULT 'in_progress'
);

CREATE INDEX IF NOT EXISTS idx_quiz_sessions_user_status
    ON quiz_sessions (user_id, status);

-- Per-user notification and delivery preferences.
CREATE TABLE IF NOT EXISTS user_preferences (
    user_id                TEXT    PRIMARY KEY,
    delivery_interval_min  INTEGER NOT NULL DEFAULT 60,
    max_unanswered         INTEGER NOT NULL DEFAULT 3,
    active_pack_ids        TEXT    NOT NULL DEFAULT '[]',  -- JSON array
    focus_mode             TEXT    NOT NULL DEFAULT 'single',
    notify_inactivity      INTEGER NOT NULL DEFAULT 1,
    notify_inactivity_days INTEGER NOT NULL DEFAULT 2,
    notify_weak_topic      INTEGER NOT NULL DEFAULT 1,
    notify_weak_topic_pct  REAL    NOT NULL DEFAULT 50.0,
    notify_milestones      INTEGER NOT NULL DEFAULT 1,
    notify_readiness       INTEGER NOT NULL DEFAULT 1,
    notify_streak          INTEGER NOT NULL DEFAULT 1,
    quiet_hours_start      TEXT    NOT NULL DEFAULT '22:00',
    quiet_hours_end        TEXT    NOT NULL DEFAULT '08:00'
);
