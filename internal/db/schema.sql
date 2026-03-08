CREATE TABLE IF NOT EXISTS digests (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    date        TEXT NOT NULL UNIQUE,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    raw_json    TEXT
);

CREATE TABLE IF NOT EXISTS news_items (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    digest_id    INTEGER NOT NULL REFERENCES digests(id),
    title        TEXT NOT NULL,
    summary      TEXT NOT NULL,
    url          TEXT,
    source       TEXT,
    priority     INTEGER DEFAULT 5,
    discarded    INTEGER DEFAULT 0,
    opened       INTEGER DEFAULT 0,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS preferences (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    topic        TEXT NOT NULL UNIQUE,
    weight       REAL DEFAULT -1.0,
    occurrences  INTEGER DEFAULT 1,
    updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS newsletter_sources (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    name         TEXT NOT NULL,
    sender_email TEXT NOT NULL UNIQUE,
    active       INTEGER DEFAULT 1
);
