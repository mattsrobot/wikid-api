CREATE TABLE files (
  id            BIGINT unsigned NOT NULL AUTO_INCREMENT,
  created_at    DATETIME NOT NULL,
  user_id       BIGINT unsigned NOT NULL,
  object_salt   VARCHAR(255) NOT NULL,
  content_size  BIGINT unsigned NOT NULL,
  file_name      VARCHAR(255) NOT NULL,
  PRIMARY KEY   (id)
);

CREATE INDEX files_user_id ON files (user_id);
