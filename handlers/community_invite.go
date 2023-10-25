package handlers

import (
	"context"
	"time"

	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/security_helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

func CommunityInvite(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	slog.Info("Starting fetch community invite ✅")

	code := Truncate(c.Query("code"), 255)

	if len(code) == 0 {
		slog.Error("No invite found 💀",
			slog.String("area", "code was blank"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Please provide a valid invite code.",
			}},
		})
	}

	/* Fetch community */

	invite := model.CommunityInvites{}

	err := db.Get(&invite, "SELECT * FROM community_invites WHERE code = ? LIMIT 1", code)

	if err != nil {
		slog.Error("No invite found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "can't find invite"))

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	community := model.Communities{}

	err = db.Get(&community, "SELECT * FROM communities WHERE id = ? LIMIT 1", invite.CommunityID)

	if err != nil {
		slog.Error("No community found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "can't find community"))

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"id":         security_helpers.Encode(community.ID, model.COMMUNITIES_TYPE, community.Salt),
		"created_at": community.CreatedAt.Format(time.RFC3339),
		"name":       community.Name,
		"handle":     community.Handle,
	})
}
