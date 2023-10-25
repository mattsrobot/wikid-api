ALTER TABLE users ADD COLUMN password_hash VARCHAR(255) NOT NULL;
CREATE INDEX users_password_hash_idx ON users (password_hash);
