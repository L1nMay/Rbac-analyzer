-- 003_scans.sql
-- Stores RBAC scans (raw yaml + computed analysis json) for history and diff.

CREATE TABLE IF NOT EXISTS scans (
  id            BIGSERIAL PRIMARY KEY,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  title         TEXT NOT NULL DEFAULT '',
  source        TEXT NOT NULL DEFAULT 'upload',
  sha256        TEXT NOT NULL UNIQUE,
  rbac_yaml     BYTEA NOT NULL,
  analysis_json JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS scans_created_at_idx ON scans(created_at DESC);
