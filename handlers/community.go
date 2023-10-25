package handlers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/security_helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

func Community(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	slog.Info("Starting fetch community ✅")

	user, userOk := c.Locals("viewer").(model.Users)

	handle := Truncate(strings.ToLower(c.Params("handle")), 255)

	community := model.Communities{}

	err := db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", handle)

	if err != nil {
		slog.Info("No community found 💀 " + handle)
		slog.Error(err.Error())

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	particiapnt := false

	if userOk {
		var rolesCount int

		err = db.Get(&rolesCount, "SELECT count(*) FROM communities_users WHERE user_id = ? AND community_id = ?", user.ID, community.ID)

		if err != nil {
			slog.Error("Can't fetch participant count 💀",
				slog.String("error", err.Error()))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found.",
				}},
			})
		}

		particiapnt = rolesCount > 0
	}

	communityRoles := []model.CommunityRoles{}

	err = db.Select(&communityRoles, "SELECT * FROM community_roles WHERE community_id = ?", community.ID)

	if err != nil {
		slog.Info("Database problem 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	mcr := make([]fiber.Map, len(communityRoles))

	for i, ch := range communityRoles {
		mcr[i] = ch.ToFiberMap(false)
	}

	var topChannels []model.Channels

	err = db.Select(&topChannels, "SELECT * FROM channels WHERE community_id = ? AND group_id = ?", community.ID, 0)

	if err != nil {
		slog.Info("Database problem 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	filteredTopChannels := []model.Channels{}

	for _, channel := range topChannels {
		hasPermission := HasChannelPermission(user.ID, channel.ID, model.ViewChannels, db, wRdb, rRdb, ctx)

		if hasPermission {
			filteredTopChannels = append(filteredTopChannels, channel)
		}
	}

	topChannels = filteredTopChannels

	mtc := make([]fiber.Map, len(topChannels))

	for i, ch := range topChannels {

		var unreadMessages int = 0

		rk := fmt.Sprintf("user-%d-channel-%d", user.ID, ch.ID)

		if val, err := rRdb.Get(ctx, rk).Result(); err == nil {
			if tm, err := time.Parse(time.RFC3339, val); err == nil {

				q := `SELECT count(*)
				      FROM messages
				      WHERE channel_id = ?
				      AND NOT user_id = ?
				      AND created_at > ?`

				err = db.Get(&unreadMessages, q, ch.ID, user.ID, tm)

				if err != nil {
					slog.Error("Database problem 💀",
						slog.String("err", err.Error()))
				}
			}
		}

		mtc[i] = fiber.Map{
			"id":           security_helpers.Encode(ch.ID, model.CHANNELS_TYPE, ch.Salt),
			"name":         ch.Name,
			"handle":       ch.Handle,
			"unread_count": unreadMessages,
		}
	}

	var channelGroups []model.ChannelGroups

	err = db.Select(&channelGroups, "SELECT * FROM channel_groups WHERE community_id = ?", community.ID)

	if err != nil {
		slog.Info("Database problem 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	filteredChannelGroups := []model.ChannelGroups{}

	for _, group := range channelGroups {
		hasPermission := HasGroupPermission(user.ID, group.ID, db, wRdb, rRdb, ctx)

		if hasPermission {
			filteredChannelGroups = append(filteredChannelGroups, group)
		}
	}

	channelGroups = filteredChannelGroups

	mg := make([]fiber.Map, len(channelGroups))

	for i, cg := range channelGroups {

		channels := []model.Channels{}

		err = db.Select(&channels, "SELECT * FROM channels WHERE group_id = ?", cg.ID)

		if err != nil {
			slog.Info("Database problem 💀")

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		mgc := make([]fiber.Map, len(channels))

		for i, ch := range channels {

			var unreadMessages int = 0

			rk := fmt.Sprintf("user-%d-channel-%d", user.ID, ch.ID)

			if val, err := rRdb.Get(ctx, rk).Result(); err == nil {
				if tm, err := time.Parse(time.RFC3339, val); err == nil {

					q := `SELECT count(*)
					      FROM messages
					      WHERE channel_id = ?
					      AND NOT user_id = ?
					      AND created_at > ?`

					err = db.Get(&unreadMessages, q, ch.ID, user.ID, tm)

					if err != nil {
						slog.Error("Database problem 💀",
							slog.String("err", err.Error()))
					}
				}
			}

			mgc[i] = fiber.Map{
				"id":           security_helpers.Encode(ch.ID, model.CHANNELS_TYPE, ch.Salt),
				"name":         ch.Name,
				"handle":       ch.Handle,
				"unread_count": unreadMessages,
			}
		}

		mg[i] = fiber.Map{
			"id":       security_helpers.Encode(cg.ID, model.CHANNEL_GROUPS_TYPE, cg.Salt),
			"name":     cg.Name,
			"channels": mgc,
		}
	}

	var mu *fiber.Map = nil

	showCanJoin := !community.Private
	permissions := community.Permissions

	if userOk {

		var avatarUrl *string = nil

		if user.CFAvatarImagesID.Valid {
			s := os.Getenv("CLOUDFLARE_IMAGES_PROXY") + user.CFAvatarImagesID.String + "/public"
			avatarUrl = &s
		}

		mu = &fiber.Map{
			"id":         security_helpers.Encode(user.ID, model.USERS_TYPE, user.Salt),
			"handle":     user.Handle.String,
			"name":       user.Name.String,
			"avatar_url": avatarUrl,
		}

		showCanJoin = !particiapnt && !community.Private

		communityUser := model.CommunitiesUsers{}

		pq := `SELECT *
		       FROM communities_users
			   WHERE community_id = ?
			   AND user_id = ?
		`

		err = db.Get(&communityUser, pq, community.ID, user.ID)

		if err != nil {
			slog.Error("Database problem 💀",
				slog.String("error", err.Error()))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		permissions = communityUser.Permissions
	}

	severOwner := false

	if userOk {
		severOwner = user.ID == community.OwnerID
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"id":             security_helpers.Encode(community.ID, model.COMMUNITIES_TYPE, community.Salt),
		"created_at":     community.CreatedAt.Format(time.RFC3339),
		"name":           community.Name,
		"handle":         community.Handle,
		"top_channels":   mtc,
		"channel_groups": mg,
		"user":           mu,
		"private":        community.Private,
		"show_can_join":  showCanJoin,
		"permissions":    permissions.ToFiberMap(),
		"server_owner":   severOwner,
	})
}
