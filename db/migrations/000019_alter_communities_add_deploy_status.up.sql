ALTER TABLE communities ADD COLUMN railway_deploy_status VARCHAR(255);
CREATE INDEX communities_railway_deploy_status_idx ON communities (railway_deploy_status);
