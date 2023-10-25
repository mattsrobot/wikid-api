package handlers

import (
	"context"
	"encoding/json"
	"fmt"
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

func AuthorizationREST(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	jwtToken, ok := c.Locals("user").(*jwt.Token)

	if !ok {
		slog.Info("Guest user request")
		return c.Next()
	}

	claims := jwtToken.Claims.(jwt.MapClaims)
	id := claims["id"].(string)
	dbId, _ := security_helpers.Decode(id)

	if dbId == 0 {
		slog.Error("💀 Unauthorized user attempt 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to authorize",
			}},
		})
	}

	val, err := rRdb.Get(ctx, fmt.Sprintf("user-%d", dbId)).Result()

	if err != nil {
		slog.Warn(fmt.Sprintf("Couldn't fetch user from Redis, going to database user-%d", dbId))

		user := model.Users{}

		err = db.Get(&user, "SELECT * FROM users WHERE id = ?", dbId)

		if err != nil || user.ID == 0 {

			if err != nil {
				slog.Error("💀 User doesn't exist 💀",
					slog.String("error", err.Error()))
			} else {
				slog.Error("💀 User doesn't exist 💀")
			}

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "System error",
				}},
			})
		}

		p, err := json.Marshal(user)

		if err != nil {
			slog.Error("Unable to authorize",
				slog.String("error", err.Error()))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Unable to authorize",
				}},
			})
		}

		go func() {
			_, err = wRdb.Set(ctx, fmt.Sprintf("user-%d", user.ID), p, 1*time.Hour).Result()

			if err != nil {
				slog.Error("Unable cache user in redis to authorize",
					slog.String("error", err.Error()))
			}
		}()

		c.Locals("viewer", user)

		return c.Next()
	}

	viewer := model.Users{}

	json.Unmarshal([]byte(val), &viewer)

	if viewer.ID == 0 {
		slog.Error("No user found")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "No user found",
			}},
		})
	}

	go func() {
		db.Exec("UPDATE users SET last_active_at = ? WHERE id = ?", time.Now(), viewer.ID)
	}()

	slog.Info("Attached viewer")

	c.Locals("viewer", viewer)

	return c.Next()
}
