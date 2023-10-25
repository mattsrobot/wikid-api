package handlers

import (
	"context"
	"os"
	"time"

	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/security_helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

func RefreshToken(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	user := c.Locals("viewer").(model.Users)

	claims := jwt.MapClaims{
		"id":  security_helpers.Encode(user.ID, model.USERS_TYPE, user.Salt),
		"exp": time.Now().Add((time.Hour * 24) * 31).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	t, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))

	if err != nil {
		slog.Error("Unable to refresh token")
		slog.Error(err.Error())

		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to refresh token",
			}},
		})
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"token": t,
	})
}
