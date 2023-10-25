package model

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/macwilko/exotic-auth/security_helpers"
)

type Channels struct {
	ID          uint64       `db:"id"`
	CreatedAt   time.Time    `db:"created_at"`
	UpdatedAt   sql.NullTime `db:"updated_at"`
	CommunityID uint64       `db:"community_id"`
	GroupID     uint64       `db:"group_id"`
	Salt        string       `db:"object_salt"`
	Name        string       `db:"name"`
	Handle      string       `db:"handle"`
}

func (c Channels) ToFiberMap() fiber.Map {
	return fiber.Map{
		"id":         security_helpers.Encode(c.ID, CHANNELS_TYPE, c.Salt),
		"created_at": c.CreatedAt.Format(time.RFC3339),
		"name":       c.Name,
		"handle":     c.Handle,
	}
}

var CHANNELS_TYPE = "Channel"
