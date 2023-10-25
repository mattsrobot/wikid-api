package model

import (
	"database/sql"
	"time"
)

type ChannelGroups struct {
	ID          uint64       `db:"id"`
	CreatedAt   time.Time    `db:"created_at"`
	UpdatedAt   sql.NullTime `db:"updated_at"`
	CommunityID uint64       `db:"community_id"`
	Salt        string       `db:"object_salt"`
	Name        string       `db:"name"`
}

var CHANNEL_GROUPS_TYPE = "ChannelGroup"
