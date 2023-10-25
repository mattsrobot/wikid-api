DROP INDEX communities_dnsimple_fqn_zone_id_uq ON communities;
ALTER TABLE communities DROP COLUMN dnsimple_fqn_zone_id;

ALTER TABLE communities ADD COLUMN cf_fqn_zone_id VARCHAR(255);
CREATE UNIQUE INDEX communities_cf_fqn_zone_id_uq ON communities (cf_fqn_zone_id);
