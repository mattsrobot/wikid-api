CREATE TABLE beta_users
(
  id            BIGINT unsigned NOT NULL AUTO_INCREMENT,
  email         VARCHAR(255) NOT NULL,
  created_at    DATETIME NOT NULL,
  updated_at    DATETIME,
  redeemed      BOOLEAN DEFAULT 0 NOT NULL,
  PRIMARY KEY   (id)
);

CREATE UNIQUE INDEX beta_users_email_uq ON beta_users (email);
CREATE UNIQUE INDEX beta_users_redeemed_uq ON beta_users (redeemed);
