package handlers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/security_helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

func Me(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, rdb *redis.Client, queue *asynq.Client) error {
	slog.Info("Starting me ✅")

	user, userOk := c.Locals("viewer").(model.Users)

	if !userOk {
		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	handleDbProblem := func(err error) error {

		if err != nil {
			slog.Error("Internal server error 💀",
				slog.String("error", err.Error()))
		} else {
			slog.Error("Internal server error 💀")
		}

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Cannot fetch user.",
			}},
		})
	}

	communityUsers := []model.CommunitiesUsers{}

	err := db.Select(&communityUsers, "SELECT community_id, selected_channel_id FROM communities_users WHERE user_id = ?", user.ID)

	if err != nil {
		return handleDbProblem(err)
	}

	communities := []model.Communities{}

	// A map of community IDs to selected channel handles
	cIdChId := make(map[uint64]string)

	if len(communityUsers) > 0 {
		communityIDs := make([]uint64, len(communityUsers))
		selectedChannelIds := []uint64{}

		for i, v := range communityUsers {
			communityIDs[i] = v.CommunityID

			if v.SelectedChannelID > 0 {
				selectedChannelIds = append(selectedChannelIds, v.SelectedChannelID)
			}
		}

		query, args, err := sqlx.In("SELECT * FROM communities WHERE id IN (?)", communityIDs)

		if err != nil {
			return handleDbProblem(err)
		}

		query = db.Rebind(query)

		err = db.Select(&communities, query, args...)

		if err != nil {
			return handleDbProblem(err)
		}

		channels := []model.Channels{}

		if len(selectedChannelIds) > 0 {
			cQuery, cArgs, err := sqlx.In("SELECT id, handle FROM channels WHERE id IN (?)", selectedChannelIds)

			if err != nil {
				return handleDbProblem(err)
			}

			cQuery = db.Rebind(cQuery)

			err = db.Select(&channels, cQuery, cArgs...)

			if err != nil {
				return handleDbProblem(err)
			}

			for _, v := range communityUsers {
				for _, ch := range channels {
					if ch.ID == v.SelectedChannelID {
						cIdChId[v.CommunityID] = ch.Handle
						break
					}
				}
			}
		}
	}

	mappedCommunities := make([]fiber.Map, len(communities))

	for i, community := range communities {
		severOwner := user.ID == community.OwnerID

		permissions := community.Permissions

		pq := `SELECT view_channels, manage_channels, manage_community, create_invite, kick_members,
		ban_members, send_messages, attach_media FROM communities_users WHERE community_id = ? AND user_id = ?
		`

		err = db.Get(&permissions, pq, community.ID, user.ID)

		if err != nil {
			slog.Error("Database problem 💀",
				slog.String("error", err.Error()))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		defaultChannel, ok := cIdChId[community.ID]

		if !ok {
			var handle string

			pq := `
			SELECT handle
			FROM channels
			WHERE community_id = ?
			LIMIT 1
			`

			err = db.Get(&handle, pq, community.ID)

			if err != nil {
				slog.Error("Database problem 💀",
					slog.String("error", err.Error()))

				handle = ""
			}

			defaultChannel = handle
		}

		channels := []uint64{}

		err = db.Select(&channels, "SELECT id FROM channels WHERE community_id = ?", community.ID)

		if err != nil {
			slog.Info("Database problem 💀")

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		var unreadCount uint64 = 0

		for _, ch := range channels {

			var msgCount uint64 = 0

			rk := fmt.Sprintf("user-%d-channel-%d", user.ID, ch)

			if val, err := rdb.Get(ctx, rk).Result(); err == nil {
				if tm, err := time.Parse(time.RFC3339, val); err == nil {
					q := `SELECT count(*)
					FROM messages
					WHERE channel_id = ?
					AND NOT user_id = ?
					AND created_at > ?`

					err = db.Get(&msgCount, q, ch, user.ID, tm)

					if err != nil {
						slog.Error("Database problem 💀",
							slog.String("err", err.Error()))
					}
				}
			}

			unreadCount = unreadCount + msgCount
		}

		mappedCommunities[i] = fiber.Map{
			"id":              security_helpers.Encode(community.ID, model.COMMUNITIES_TYPE, community.Salt),
			"name":            community.Name,
			"handle":          community.Handle,
			"permissions":     permissions.ToFiberMap(),
			"server_owner":    severOwner,
			"default_channel": defaultChannel,
			"unread_count":    unreadCount,
		}
	}

	var avatarUrl *string = nil

	if user.CFAvatarImagesID.Valid {
		s := os.Getenv("CLOUDFLARE_IMAGES_PROXY") + user.CFAvatarImagesID.String + "/public"
		avatarUrl = &s
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"id":          security_helpers.Encode(user.ID, model.USERS_TYPE, user.Salt),
		"created_at":  user.CreatedAt.Format(time.RFC3339),
		"name":        user.Name.String,
		"handle":      user.Handle.String,
		"email":       user.Email,
		"user_count":  user.ID,
		"about":       user.About.String,
		"communities": mappedCommunities,
		"avatar_url":  avatarUrl,
	})
}
