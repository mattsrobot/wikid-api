CREATE TABLE communities_users
(
  created_at    DATETIME NOT NULL,
  community_id  BIGINT unsigned NOT NULL,
  user_id       BIGINT unsigned NOT NULL
);

CREATE INDEX communities_users_community_id ON communities_users (community_id);
CREATE INDEX communities_users_user_id ON communities_users (user_id);
