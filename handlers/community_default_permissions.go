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

func CommunityDefaultPermissions(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	slog.Info("Starting fetch community default permissions ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	community := model.Communities{}

	lowerHandle := Truncate(strings.ToLower(c.Params("handle")), 255)

	err := db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", lowerHandle)

	if err != nil {
		slog.Error("Community not found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "getting default permissions"))

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	hasPermission := HasCommunityPermission(user.ID, community.ID, model.ManageCommunity, db, wRdb, rRdb, ctx)

	if !hasPermission {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	return c.Status(fiber.StatusOK).JSON(community.Permissions.ToFiberMap())
}
