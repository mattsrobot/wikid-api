CREATE TABLE community_invites (
    id BIGINT unsigned NOT NULL AUTO_INCREMENT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME,
    expires_at DATETIME NOT NULL,
    object_salt VARCHAR(255) NOT NULL,
    community_id BIGINT unsigned NOT NULL,
    user_id BIGINT unsigned NOT NULL,
    code VARCHAR(255) NOT NULL,
    PRIMARY KEY (id)
);

CREATE INDEX community_invites_community_id_idx ON community_invites (community_id);
CREATE INDEX community_invites_user_id_idx ON community_invites (user_id);
CREATE INDEX community_invites_expires_at_idx ON community_invites (expires_at);
CREATE UNIQUE INDEX community_invites_code_uq ON community_invites (code);
