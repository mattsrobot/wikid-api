ALTER TABLE beta_users ADD COLUMN enabled BOOLEAN DEFAULT 0 NOT NULL;
CREATE INDEX beta_users_enabled_idx ON beta_users (enabled);
