PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS events (
    id         INTEGER PRIMARY KEY,
    slug       TEXT UNIQUE NOT NULL,
    name       TEXT NOT NULL,
    date       TEXT,
    location   TEXT,
    status     TEXT DEFAULT 'draft',
    created_at TEXT
);

CREATE TABLE IF NOT EXISTS categories (
    id         INTEGER PRIMARY KEY,
    event_id   INTEGER REFERENCES events(id),
    code       TEXT NOT NULL,
    name       TEXT NOT NULL,
    sort_order INTEGER DEFAULT 0,
    num_runs   INTEGER DEFAULT 2
);

CREATE TABLE IF NOT EXISTS athletes (
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL,
    club       TEXT,
    nation     TEXT,
    bio        TEXT,
    photo_url  TEXT,
    created_at TEXT
);

CREATE TABLE IF NOT EXISTS entries (
    id             INTEGER PRIMARY KEY,
    event_id       INTEGER REFERENCES events(id),
    category_id    INTEGER REFERENCES categories(id),
    athlete_id     INTEGER REFERENCES athletes(id),
    bib_number     INTEGER,
    start_position INTEGER,
    UNIQUE(event_id, athlete_id)
);

CREATE TABLE IF NOT EXISTS runs (
    id               INTEGER PRIMARY KEY,
    entry_id         INTEGER REFERENCES entries(id),
    run_number       INTEGER,
    raw_time_ms      INTEGER,
    penalty_touches  INTEGER DEFAULT 0,
    penalty_misses   INTEGER DEFAULT 0,
    penalty_seconds  INTEGER DEFAULT 0,
    total_time_ms    INTEGER,
    status           TEXT DEFAULT 'ok',
    judged_at        TEXT,
    UNIQUE(entry_id, run_number)
);

CREATE TABLE IF NOT EXISTS sponsors (
    id          INTEGER PRIMARY KEY,
    event_id    INTEGER REFERENCES events(id),
    name        TEXT NOT NULL,
    logo_url    TEXT NOT NULL,
    website_url TEXT,
    tier        TEXT NOT NULL DEFAULT 'supporter',
    sort_order  INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS photos (
    id                INTEGER PRIMARY KEY,
    event_id          INTEGER REFERENCES events(id),
    athlete_id        INTEGER REFERENCES athletes(id),
    image_url         TEXT NOT NULL,
    caption           TEXT,
    photographer_name TEXT,
    created_at        TEXT
);
