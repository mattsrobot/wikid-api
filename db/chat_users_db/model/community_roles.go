package model

import (
	"database/sql"
	"maps"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/macwilko/exotic-auth/security_helpers"
)

type CommunityRoles struct {
	ID                    uint64       `db:"id"`
	CreatedAt             time.Time    `db:"created_at"`
	UpdatedAt             sql.NullTime `db:"updated_at"`
	Salt                  string       `db:"object_salt"`
	Color                 string       `db:"color"`
	CommunityID           uint64       `db:"community_id"`
	Name                  string       `db:"name"`
	ShowOnlineDifferently bool         `db:"show_online_differently"`
	Priority              uint16       `db:"priority"`
	Permissions
}

func (c CommunityRoles) ToFiberMap(showPermissions bool) fiber.Map {
	info := fiber.Map{
		"id":                      security_helpers.Encode(c.ID, COMMUNITY_ROLES_TYPE, c.Salt),
		"created_at":              c.CreatedAt.Format(time.RFC3339),
		"name":                    c.Name,
		"color":                   c.Color,
		"priority":                c.Priority,
		"show_online_differently": c.ShowOnlineDifferently,
	}

	permissions := fiber.Map{
		"view_channels":    c.ViewChannels,
		"manage_channels":  c.ManageChannels,
		"manage_community": c.ManageCommunity,
		"create_invite":    c.CreateInvite,
		"kick_members":     c.KickMembers,
		"ban_members":      c.BanMembers,
		"send_messages":    c.SendMessages,
		"attach_media":     c.AttachMedia,
	}

	if showPermissions {
		maps.Copy(info, permissions)
	}

	return info
}

var COMMUNITY_ROLES_TYPE = "CommunityRole"
