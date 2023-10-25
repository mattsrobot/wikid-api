ALTER TABLE communities ADD COLUMN ready BOOLEAN DEFAULT 0 NOT NULL;
CREATE INDEX communities_ready_idx ON communities (ready);
