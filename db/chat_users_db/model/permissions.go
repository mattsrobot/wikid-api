package model

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type Permission int

const (
	ViewChannels Permission = iota + 1
	ManageChannels
	ManageCommunity
	CreateInvite
	KickMembers
	BanMembers
	SendMessages
	AttachMedia
)

func (w Permission) String() string {
	return [...]string{
		"view_channels",
		"manage_channels",
		"manage_community",
		"create_invite",
		"kick_members",
		"ban_members",
		"send_messages",
		"attach_media"}[w-1]
}

func (w Permission) EnumIndex() int {
	return int(w)
}

type Permissions struct {
	ViewChannels    bool `db:"view_channels"`
	ManageChannels  bool `db:"manage_channels"`
	ManageCommunity bool `db:"manage_community"`
	CreateInvite    bool `db:"create_invite"`
	KickMembers     bool `db:"kick_members"`
	BanMembers      bool `db:"ban_members"`
	SendMessages    bool `db:"send_messages"`
	AttachMedia     bool `db:"attach_media"`
}

func (c Permissions) ToFiberMap() fiber.Map {
	return fiber.Map{
		"view_channels":    c.ViewChannels,
		"manage_channels":  c.ManageChannels,
		"manage_community": c.ManageCommunity,
		"create_invite":    c.CreateInvite,
		"kick_members":     c.KickMembers,
		"ban_members":      c.BanMembers,
		"send_messages":    c.SendMessages,
		"attach_media":     c.AttachMedia,
	}
}

func PermissionRedisKey(uId uint64, cId uint64) string {
	return fmt.Sprintf("user-%d-%d-permissions", uId, cId)
}
