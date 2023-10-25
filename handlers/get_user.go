package handlers

import (
	"context"

	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/security_helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

func GetUser(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	handleNotFound := func(err error) error {
		slog.Error("not found")

		if err != nil {
			slog.Error(err.Error())
		}

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	id := Truncate(c.Params("id"), 255)

	dbId, objectType := security_helpers.Decode(id)

	if objectType != model.USERS_TYPE {
		return handleNotFound(nil)
	}

	user := model.Users{}

	err := db.Get(&user, "SELECT id, handle, name, object_salt FROM users WHERE id = ?", dbId)

	if err != nil || user.ID == 0 {
		return handleNotFound(err)
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"id":     security_helpers.Encode(user.ID, model.USERS_TYPE, user.Salt),
		"name":   user.Name.String,
		"handle": user.Handle.String,
	})
}
