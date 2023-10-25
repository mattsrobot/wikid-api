CREATE TABLE users
(
  id            BIGINT unsigned NOT NULL AUTO_INCREMENT,
  email         VARCHAR(255) NOT NULL,
  name          VARCHAR(150),
  handle        VARCHAR(255),
  PRIMARY KEY   (id)
);

CREATE UNIQUE INDEX users_email_uq ON users (email);
CREATE UNIQUE INDEX users_handle_uq ON users (handle);
