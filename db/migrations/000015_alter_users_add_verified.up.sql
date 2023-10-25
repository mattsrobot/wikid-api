ALTER TABLE users ADD COLUMN verified BOOLEAN DEFAULT 0 NOT NULL;
CREATE INDEX users_verified_idx ON users (verified);
