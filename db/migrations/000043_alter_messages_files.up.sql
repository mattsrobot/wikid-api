ALTER TABLE files ADD COLUMN message_id BIGINT;
CREATE INDEX files_message_id_idx ON files (message_id);
