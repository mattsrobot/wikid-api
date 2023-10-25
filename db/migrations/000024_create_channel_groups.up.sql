CREATE TABLE channel_groups
(
  id            BIGINT unsigned NOT NULL AUTO_INCREMENT,
  created_at    DATETIME NOT NULL,
  updated_at    DATETIME,
  community_id  BIGINT unsigned NOT NULL,
  object_salt   VARCHAR(255) NOT NULL,
  name          VARCHAR(255) NOT NULL,
  PRIMARY KEY   (id)
);

CREATE INDEX channel_groups_community_idx ON channel_groups (community_id);

ALTER TABLE channels ADD COLUMN group_id BIGINT unsigned NOT NULL;
CREATE INDEX channels_group_id_idx ON channels (group_id);
