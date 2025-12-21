-- 002_plans.sql

CREATE TABLE IF NOT EXISTS plans (
  id TEXT PRIMARY KEY,            -- free/pro/enterprise
  name TEXT NOT NULL,
  max_clusters INT NOT NULL,
  scheduled_scans BOOLEAN NOT NULL,
  diff BOOLEAN NOT NULL,
  alerts BOOLEAN NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO plans (id, name, max_clusters, scheduled_scans, diff, alerts)
VALUES
  ('free', 'Free', 1, false, false, false),
  ('pro', 'Pro', 50, true, true, true),
  ('enterprise', 'Enterprise', 1000000, true, true, true)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS subscriptions (
  org_id UUID PRIMARY KEY REFERENCES orgs(id) ON DELETE CASCADE,
  plan_id TEXT NOT NULL REFERENCES plans(id),
  status TEXT NOT NULL DEFAULT 'active', -- active/past_due/canceled
  stripe_customer_id TEXT NOT NULL DEFAULT '',
  stripe_subscription_id TEXT NOT NULL DEFAULT '',
  current_period_end TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
