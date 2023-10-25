package model

import (
	"time"
)

type CommuniyRolesUsers struct {
	CreatedAt       time.Time `db:"created_at"`
	CommunityRoleID uint64    `db:"community_role_id"`
	UserID          uint64    `db:"user_id"`
	CommunityID     uint64    `db:"community_id"`
}

var COMMUNITY_ROLES_USERS_TYPE = "CommunityRoleUsers"
