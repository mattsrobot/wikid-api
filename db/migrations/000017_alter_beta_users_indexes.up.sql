DROP INDEX beta_users_redeemed_uq ON beta_users;
CREATE INDEX beta_users_redeemed_idx ON beta_users (redeemed);
