package model

import (
	"database/sql"
	"time"
)

type Communities struct {
	ID                  uint64         `db:"id"`
	CreatedAt           time.Time      `db:"created_at"`
	UpdatedAt           sql.NullTime   `db:"updated_at"`
	Private             bool           `db:"private"`
	SelfHosted          bool           `db:"self_hosted"`
	OwnerID             uint64         `db:"owner_id"`
	Salt                string         `db:"object_salt"`
	Name                string         `db:"name"`
	Handle              string         `db:"handle"`
	FQN                 string         `db:"fqn"`
	RailwayServiceID    sql.NullString `db:"railway_service_id"`
	RailwayDeployStatus sql.NullString `db:"railway_deploy_status"`
	CFRecordID          sql.NullString `db:"cf_fqn_zone_id"`
	Ready               bool           `db:"ready"`
	Permissions
}

var COMMUNITIES_TYPE = "Communities"
