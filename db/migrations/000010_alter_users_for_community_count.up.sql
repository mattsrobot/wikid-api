ALTER TABLE users ADD COLUMN community_owner_count SMALLINT;
CREATE INDEX users_community_owner_count_idx ON users (community_owner_count);

ALTER TABLE users ADD COLUMN community_participant_count SMALLINT;
CREATE INDEX users_community_participant_count_idx ON users (community_participant_count);
