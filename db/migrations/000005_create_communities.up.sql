CREATE TABLE communities
(
  id            BIGINT unsigned NOT NULL AUTO_INCREMENT,
  created_at    DATETIME NOT NULL,
  updated_at    DATETIME,
  self_hosted   BOOLEAN NOT NULL,
  owner_id      BIGINT unsigned NOT NULL,
  object_salt   VARCHAR(255) NOT NULL,
  name          VARCHAR(255) NOT NULL,
  PRIMARY KEY   (id)
);

CREATE INDEX communities_owner_id ON communities (owner_id);
CREATE INDEX communities_self_hosted_id ON communities (self_hosted);
