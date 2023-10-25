package model

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
)

type CommunityInvites struct {
	ID           uint64       `db:"id"`
	CreatedAt    time.Time    `db:"created_at"`
	UpdatedAt    sql.NullTime `db:"updated_at"`
	ExpiresAt    time.Time    `db:"expires_at"`
	ExpiresOnUse bool         `db:"expires_on_use"`
	Salt         string       `db:"object_salt"`
	Code         string       `db:"code"`
	UserID       uint64       `db:"user_id"`
	CommunityID  uint64       `db:"community_id"`
}

func (c CommunityInvites) ToFiberMap(showPermissions bool) fiber.Map {
	info := fiber.Map{
		"code":       c.Code,
		"created_at": c.CreatedAt.Format(time.RFC3339),
		"expires_at": c.ExpiresAt.Format(time.RFC3339),
	}

	return info
}

var COMMUNITY_INVITES_TYPE = "CommunityInvites"
