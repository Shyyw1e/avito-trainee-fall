CREATE TABLE teams (
    team_name  TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    user_id    TEXT PRIMARY KEY,
    username   TEXT NOT NULL,
    team_name  TEXT NOT NULL REFERENCES teams(team_name) ON DELETE RESTRICT,
    is_active  BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_users_team_active ON users(team_name, is_active);

CREATE TYPE pr_status AS ENUM ('OPEN', 'MERGED');

CREATE TABLE prs (
    pr_id      TEXT PRIMARY KEY,
    pr_name    TEXT NOT NULL,
    author_id  TEXT NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
    status     pr_status NOT NULL DEFAULT 'OPEN',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    merged_at  TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_prs_author ON prs(author_id);
CREATE INDEX IF NOT EXISTS idx_prs_status ON prs(status);

CREATE TABLE pr_reviewers (
    pr_id       TEXT NOT NULL REFERENCES prs(pr_id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
    slot        INT  NOT NULL CHECK (slot IN (0,1)),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (pr_id, slot),
    UNIQUE (pr_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_pr_reviewers_user ON pr_reviewers(user_id);
CREATE INDEX IF NOT EXISTS idx_pr_reviewers_pr   ON pr_reviewers(pr_id);

CREATE TABLE pr_events (
    id             BIGSERIAL PRIMARY KEY,
    pr_id          TEXT NOT NULL REFERENCES prs(pr_id) ON DELETE CASCADE,
    event_type     TEXT NOT NULL,
    actor_user_id  TEXT REFERENCES users(user_id) ON DELETE RESTRICT,
    old_user_id    TEXT REFERENCES users(user_id) ON DELETE RESTRICT,
    new_user_id    TEXT REFERENCES users(user_id) ON DELETE RESTRICT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
