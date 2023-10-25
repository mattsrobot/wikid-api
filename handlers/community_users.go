package handlers

import (
	"context"
	"strings"

	"github.com/macwilko/exotic-auth/db/chat_users_db/model"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

func CommunityUsers(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	slog.Info("Starting fetch community users ✅")

	/* Fetch community */

	communityHandle := Truncate(strings.ToLower(c.Params("communityHandle")), 255)

	community := model.Communities{}

	err := db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", communityHandle)

	if err != nil {
		slog.Error("No community found 💀", slog.String("error", err.Error()), slog.String("area", "can't find  community"))

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	/* Got community  */

	communityUsers := []model.CommunitiesUsers{}

	err = db.Select(&communityUsers, "SELECT * FROM communities_users WHERE community_id = ?", community.ID)

	if err != nil {
		slog.Error("No channel found 💀", slog.String("error", err.Error()), slog.String("area", "can't find community users"))

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	var uIds = []uint64{}

	for _, cu := range communityUsers {
		uIds = append(uIds, cu.UserID)
	}

	users := []model.Users{}

	usersQuery, usersArgs, err := sqlx.In("SELECT * FROM users WHERE id IN (?) ORDER BY name ASC", uIds)

	if err != nil {
		slog.Error("Database problem 💀", slog.String("error", err.Error()), slog.String("area", "selecting users IN"))

		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	usersQuery = db.Rebind(usersQuery)

	err = db.Select(&users, usersQuery, usersArgs...)

	if err != nil {
		slog.Error("Database problem 💀", slog.String("error", err.Error()), slog.String("area", "after the bind to users query"))

		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	channelUsers := make([]fiber.Map, len(users))

	for i, cu := range users {
		channelUsers[i] = cu.ToFiberMap()
	}

	return c.Status(fiber.StatusOK).JSON(&channelUsers)
}
