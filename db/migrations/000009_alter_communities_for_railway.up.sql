ALTER TABLE communities ADD COLUMN railway_service_id VARCHAR(255);
CREATE UNIQUE INDEX communities_railway_service_id_uq ON communities (railway_service_id);

ALTER TABLE communities ADD COLUMN dnsimple_fqn_zone_id BIGINT unsigned;
CREATE UNIQUE INDEX communities_dnsimple_fqn_zone_id_uq ON communities (dnsimple_fqn_zone_id);
