package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/jmoiron/sqlx"
	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

var ValidColors = []string{"tomato", "red", "ruby", "crimson", "pink", "plum",
	"purple", "violet", "iris", "indigo", "blue", "cyan",
	"teal", "jade", "green", "grass", "brown", "orange",
	"sky", "mint", "lime", "yellow", "amber", "gold",
	"bronze", "gray"}

func Truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}

	if utf8.RuneCountInString(s) < max {
		return s
	}

	return string([]rune(s)[:max])
}

func HasGroupPermission(uId uint64, gId uint64, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, ctx context.Context) bool {
	return true
}

func HasChannelPermission(uId uint64, cId uint64, permission model.Permission, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, ctx context.Context) bool {
	return true
}

func HasCommunityPermission(uId uint64, cId uint64, permission model.Permission, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, ctx context.Context) bool {

	rk := model.PermissionRedisKey(uId, cId)

	rp, err := rRdb.Get(ctx, rk).Result()

	if err == redis.Nil {

		q := `SELECT *
			  FROM communities_users
			  WHERE user_id = ? AND community_id = ?`

		var cU model.CommunitiesUsers

		err = db.Get(&cU, q, uId, cId)

		if err != nil {
			slog.Warn("Does not have permission 💀",
				slog.Uint64("uId", uId),
				slog.Uint64("cId", cId),
				slog.String("error", err.Error()))

			return false
		}

		mCu, err := json.Marshal(cU)

		if err != nil {
			slog.Warn("Redis error setting v 💀",
				slog.Uint64("uId", uId),
				slog.Uint64("cId", cId),
				slog.String("error", err.Error()))

			return false
		}

		go func() {
			_, err = wRdb.Set(ctx, rk, mCu, 1*time.Hour).Result()

			if err != nil {
				slog.Warn("Redis error setting v 💀",
					slog.Uint64("uId", uId),
					slog.Uint64("cId", cId),
					slog.String("error", err.Error()))
			}
		}()

		return cU.HasCommunityPermission(permission)

	} else if err != nil {
		slog.Error("Redis problem 💀",
			slog.String("error", err.Error()),
			slog.Uint64("uId", uId),
			slog.Uint64("cId", cId),
			slog.String("area", "selecting permissions from Redis"))

		return false
	} else {

		cU := model.CommunitiesUsers{}

		json.Unmarshal([]byte(rp), &cU)

		if err != nil {
			slog.Warn("String conv from bool after redis v 💀",
				slog.Uint64("uId", uId),
				slog.Uint64("cId", cId),
				slog.String("error", err.Error()))

			return false
		}

		return cU.HasCommunityPermission(permission)
	}
}

func ServerOwnerPermissions() model.Permissions {
	return model.Permissions{
		ViewChannels:    true,
		ManageChannels:  true,
		ManageCommunity: true,
		CreateInvite:    true,
		KickMembers:     true,
		BanMembers:      true,
		SendMessages:    true,
		AttachMedia:     true,
	}
}

func RecalculateAndUpdatePermissionsForUsers(userIds []uint64, community model.Communities, tx *sqlx.Tx, wRdb *redis.Client, rRdb *redis.Client, ctx context.Context) error {

	if len(userIds) == 0 {
		return nil
	}

	communityRoles := []model.CommunityRoles{}

	sr := `SELECT *
	       FROM community_roles
		   WHERE community_id = ?`

	err := tx.Select(&communityRoles, sr, community.ID)

	if err != nil {
		return err
	}

	communityRolesUsers := []model.CommuniyRolesUsers{}

	sru := `SELECT *
	        FROM community_roles_users
	        WHERE community_id = ?`

	err = tx.Select(&communityRolesUsers, sru, community.ID)

	if err != nil {
		return err
	}

	rolesUserMap := make(map[string]bool)

	for _, r := range communityRolesUsers {
		rolesUserMap[fmt.Sprintf("%d-%d", r.CommunityRoleID, r.UserID)] = true
	}

	for _, uid := range userIds {

		permissions := community.Permissions

		if uid == community.OwnerID {
			permissions = ServerOwnerPermissions()
		} else {
			for _, role := range communityRoles {

				_, found := rolesUserMap[fmt.Sprintf("%d-%d", role.ID, uid)]

				if !found {
					continue
				}

				if role.ViewChannels {
					permissions.ViewChannels = true
				}

				if role.ManageChannels {
					permissions.ManageChannels = true
				}

				if role.ManageCommunity {
					permissions.ManageCommunity = true
				}

				if role.CreateInvite {
					permissions.CreateInvite = true
				}

				if role.KickMembers {
					permissions.KickMembers = true
				}

				if role.BanMembers {
					permissions.BanMembers = true
				}

				if role.SendMessages {
					permissions.SendMessages = true
				}

				if role.AttachMedia {
					permissions.AttachMedia = true
				}
			}
		}

		up := `UPDATE communities_users
		       SET view_channels = ?,
			       manage_channels = ?,
			       manage_community = ?,
			       create_invite = ?,
			       kick_members = ?,
			       ban_members = ?,
			       send_messages = ?,
			       attach_media = ?
		       WHERE user_id = ?
		       AND community_id = ?`

		_, err = tx.Exec(up, permissions.ViewChannels, permissions.ManageChannels, permissions.ManageCommunity, permissions.CreateInvite,
			permissions.KickMembers, permissions.BanMembers, permissions.SendMessages, permissions.AttachMedia, uid, community.ID)

		if err != nil {
			return err
		}

		rk := model.PermissionRedisKey(uid, community.ID)

		_, err = wRdb.Del(ctx, rk).Result()

		if err != nil {
			return err
		}
	}

	return nil
}

func RecalculateAndUpdatePermissionsForUser(uId uint64, community model.Communities, tx *sqlx.Tx, wRdb *redis.Client, rRdb *redis.Client, ctx context.Context) model.Permissions {

	permissions := community.Permissions

	if uId == community.OwnerID {
		permissions = ServerOwnerPermissions()
	} else {
		var communityRoleIds []uint64

		sr := `SELECT community_role_id
				   FROM community_roles_users
				   WHERE community_id = ?
				   AND user_id = ?`

		err := tx.Select(&communityRoleIds, sr, community.ID, uId)

		if err != nil {
			slog.Error("Database problem 💀",
				slog.String("error", err.Error()),
				slog.Uint64("uId", uId),
				slog.Uint64("cId", community.ID),
				slog.String("area", "can't scan roles"))

			return model.Permissions{}
		}

		communityRoles := []model.CommunityRoles{}

		if len(communityRoleIds) > 0 {

			cr := `SELECT *
					   FROM community_roles
					   WHERE id IN (?)`

			rsQuery, rsArgs, err := sqlx.In(cr, communityRoleIds)

			if err != nil {
				slog.Error("Database problem 💀",
					slog.String("error", err.Error()),
					slog.Uint64("uId", uId),
					slog.Uint64("cId", community.ID),
					slog.String("area", "selecting community_roles IN"))

				return model.Permissions{}
			}

			rsQuery = tx.Rebind(rsQuery)

			err = tx.Select(&communityRoles, rsQuery, rsArgs...)

			if err != nil {
				slog.Error("Database problem 💀",
					slog.String("error", err.Error()),
					slog.Uint64("uId", uId),
					slog.Uint64("cId", community.ID),
					slog.String("area", "selecting community_roles IN"))

				return model.Permissions{}
			}
		}

		for _, role := range communityRoles {
			if role.ViewChannels {
				permissions.ViewChannels = true
			}

			if role.ManageChannels {
				permissions.ManageChannels = true
			}

			if role.ManageCommunity {
				permissions.ManageCommunity = true
			}

			if role.CreateInvite {
				permissions.CreateInvite = true
			}

			if role.KickMembers {
				permissions.KickMembers = true
			}

			if role.BanMembers {
				permissions.BanMembers = true
			}

			if role.SendMessages {
				permissions.SendMessages = true
			}

			if role.AttachMedia {
				permissions.AttachMedia = true
			}
		}
	}

	up := `
	UPDATE communities_users
	SET view_channels = ?,
		manage_channels = ?,
		manage_community = ?,
		create_invite = ?,
		kick_members = ?,
		ban_members = ?,
		send_messages = ?,
		attach_media = ?
	WHERE user_id = ?
	AND community_id = ?`

	tx.Exec(up, permissions.ViewChannels, permissions.ManageChannels, permissions.ManageCommunity, permissions.CreateInvite,
		permissions.KickMembers, permissions.BanMembers, permissions.SendMessages, permissions.AttachMedia, uId, community.ID)

	rk := model.PermissionRedisKey(uId, community.ID)

	wRdb.Del(ctx, rk)

	return permissions
}
