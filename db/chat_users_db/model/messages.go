package model

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/macwilko/exotic-auth/security_helpers"
)

type Messages struct {
	ID          uint64       `db:"id"`
	CreatedAt   time.Time    `db:"created_at"`
	UpdatedAt   sql.NullTime `db:"updated_at"`
	Text        string       `db:"text"`
	Salt        string       `db:"object_salt"`
	UserID      uint64       `db:"user_id"`
	ChannelID   uint64       `db:"channel_id"`
	CommunityID uint64       `db:"community_id"`
	Edited      bool         `db:"edited"`
	ParentID    uint64       `db:"parent_id"`
}

func (c Messages) ToFiberMap() fiber.Map {
	return fiber.Map{
		"id":         security_helpers.Encode(c.ID, MESSAGES_TYPE, c.Salt),
		"created_at": c.CreatedAt.Format(time.RFC3339),
		"text":       c.Text,
		"edited":     c.Edited,
	}
}

var MESSAGES_TYPE = "Messages"
