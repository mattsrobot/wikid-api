ALTER TABLE communities ADD COLUMN private BOOLEAN DEFAULT 1 NOT NULL;
CREATE INDEX communities_private_idx ON communities (private);
