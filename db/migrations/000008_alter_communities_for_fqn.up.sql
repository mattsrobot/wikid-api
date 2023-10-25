ALTER TABLE communities ADD COLUMN fqn VARCHAR(255);
CREATE UNIQUE INDEX communities_fqn_uq ON communities (fqn);
