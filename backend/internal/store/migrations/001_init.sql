CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE rooms (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug       VARCHAR(16) UNIQUE NOT NULL,
    name       VARCHAR(100) NOT NULL DEFAULT 'Watch Party',
    video_url  TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rooms_slug ON rooms (slug);
