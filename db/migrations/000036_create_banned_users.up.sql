CREATE TABLE communities_banned_users (
    created_at DATETIME NOT NULL,
    community_id BIGINT unsigned NOT NULL,
    user_id BIGINT unsigned NOT NULL
);

CREATE INDEX communities_banned_users_community_id_idx ON communities_banned_users (community_id);
CREATE INDEX communities_banned_users_user_id_idx ON communities_banned_users (user_id);
