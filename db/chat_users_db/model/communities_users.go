package model

import (
	"time"
)

type CommunitiesUsers struct {
	CreatedAt         time.Time `db:"created_at"`
	CommunityID       uint64    `db:"community_id"`
	SelectedChannelID uint64    `db:"selected_channel_id"`
	UserID            uint64    `db:"user_id"`
	Permissions
}

func (c CommunitiesUsers) HasCommunityPermission(permission Permission) bool {
	switch permission {
	case ViewChannels:
		return c.ViewChannels
	case ManageChannels:
		return c.ManageChannels
	case ManageCommunity:
		return c.ManageCommunity
	case CreateInvite:
		return c.CreateInvite
	case KickMembers:
		return c.KickMembers
	case BanMembers:
		return c.BanMembers
	case SendMessages:
		return c.SendMessages
	case AttachMedia:
		return c.AttachMedia
	default:
		return false
	}
}

var COMMUNITIES_USERS_TYPE = "CommunitiesUsers"
