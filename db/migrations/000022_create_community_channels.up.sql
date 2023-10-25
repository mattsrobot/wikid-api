CREATE TABLE channels
(
  id            BIGINT unsigned NOT NULL AUTO_INCREMENT,
  created_at    DATETIME NOT NULL,
  updated_at    DATETIME,
  community_id  BIGINT unsigned NOT NULL,
  object_salt   VARCHAR(255) NOT NULL,
  name          VARCHAR(255) NOT NULL,
  PRIMARY KEY   (id)
);

CREATE INDEX channels_community_id ON channels (community_id);
