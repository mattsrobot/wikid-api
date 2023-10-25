ALTER TABLE community_roles_users ADD COLUMN community_id BIGINT NOT NULL;
CREATE INDEX community_roles_users_community_id_idx ON community_roles_users (community_id);
ALTER TABLE community_roles ADD COLUMN priority SMALLINT NOT NULL;
