CREATE TABLE messages
(
  id            BIGINT unsigned NOT NULL AUTO_INCREMENT,
  created_at    DATETIME NOT NULL,
  updated_at    DATETIME,
  community_id  BIGINT unsigned NOT NULL,
  channel_id    BIGINT unsigned NOT NULL,
  user_id       BIGINT unsigned NOT NULL,
  text          VARCHAR(2000) NOT NULL,
  object_salt   VARCHAR(255) NOT NULL,
  PRIMARY KEY   (id)
);

CREATE INDEX messages_community_id_idx ON messages (community_id);
CREATE INDEX messages_channel_id_idx ON messages (channel_id);
CREATE INDEX messages_user_id_idx ON messages (user_id);
