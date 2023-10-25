package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/security_helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

func AuthorizationWS(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, rdb *redis.Client, queue *asynq.Client) error {

	jwtToken := c.Query("token")

	if len(jwtToken) == 0 {
		slog.Error("💀 Unauthorized, missing jwt")

		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		return c.Status(fiber.StatusUnauthorized).SendString("Missing authorization token")
	}

	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {

		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			slog.Error("💀 Unauthorized, HMAC error in jwt")

			return nil, errors.New("unexpected signing method")
		}

		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		slog.Error("💀 Unauthorized, HMAC signature did not validate")

		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		return c.Status(fiber.StatusUnauthorized).SendString(err.Error())
	}

	claims, ok := token.Claims.(jwt.MapClaims)

	if !ok {
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		return c.Status(fiber.StatusUnauthorized).SendString("Missing authorization token")
	}

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

	val, err := rdb.Get(ctx, fmt.Sprintf("user-%d", dbId)).Result()

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

		_, err = rdb.Set(ctx, fmt.Sprintf("user-%d", user.ID), p, 1*time.Hour).Result()

		if err != nil {
			slog.Error("Unable to authorize",
				slog.String("error", err.Error()))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Unable to authorize",
				}},
			})
		}

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
