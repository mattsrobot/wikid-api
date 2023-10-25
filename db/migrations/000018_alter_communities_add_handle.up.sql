ALTER TABLE communities ADD COLUMN handle VARCHAR(255) NOT NULL;
CREATE UNIQUE INDEX communities_handle_uq ON communities (handle);
