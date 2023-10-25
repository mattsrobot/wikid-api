ALTER TABLE messages ADD COLUMN parent_id BIGINT unsigned NOT NULL DEFAULT 0;
CREATE INDEX messages_parent_id_idx ON messages (parent_id);
