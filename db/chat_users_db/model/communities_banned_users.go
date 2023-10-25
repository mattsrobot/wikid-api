package model

import (
	"time"
)

type CommunitiesBannedUsers struct {
	CreatedAt   time.Time `db:"created_at"`
	CommunityID uint64    `db:"community_id"`
	UserID      uint64    `db:"user_id"`
}

var COMMUNITIES_BANNED_USERS_TYPE = "CommunitiesBannedUsers"
