package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/security_helpers"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type UpdateProfileInput struct {
	Name   string `json:"name" validate:"lte=255"`
	Handle string `json:"handle" validate:"required,gte=3,lte=255"`
	About  string `json:"about" validate:"lte=3000"`
}

func UpdateProfile(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(UpdateProfileInput)
	slog.Info("Updating profile")

	if err := c.BodyParser(input); err != nil {
		slog.Error("Unable to update profile.", slog.String("error", err.Error()), slog.String("area", "Input errors"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Invalid input.",
			}},
		})
	}

	validate := validator.New()
	en := en.New()
	uni := ut.New(en, en)
	trans, _ := uni.GetTranslator("en")
	en_translations.RegisterDefaultTranslations(validate, trans)
	err := validate.Struct(input)

	var errors []fiber.Map

	if err != nil {
		slog.Error("Unable to update profile.", slog.String("error", err.Error()), slog.String("area", "Validation errors"))

		errs := err.(validator.ValidationErrors)

		for _, v := range errs {
			errors = append(errors, fiber.Map{
				"field":   v.Field(),
				"message": v.Translate(trans),
			})
		}
	}

	if len(errors) > 0 {
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": errors,
		})
	}

	lowerHandle := strings.ToLower(input.Handle)

	var handleCount int
	err = db.Get(&handleCount, "SELECT count(*) FROM users WHERE handle = ? AND id != ?", lowerHandle, user.ID)

	if err != nil {
		slog.Error("Unable to update profile.", slog.String("error", err.Error()), slog.String("area", "handle selection"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to update profile.",
			}},
		})
	}

	if handleCount > 0 {
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "handle",
				"message": "Already taken.",
			}},
		})
	}

	var updatedUser model.Users

	tx := db.MustBegin()

	_, err = tx.Exec("UPDATE users SET updated_at = ?, name = ?, handle = ?, about = ? WHERE id = ?", time.Now(), input.Name, lowerHandle, input.About, user.ID)

	if err != nil {
		tx.Rollback()

		slog.Error("Unable to update profile.", slog.String("error", err.Error()), slog.String("area", "db insert"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to update profile.",
			}},
		})
	}

	err = tx.Get(&updatedUser, "SELECT * FROM users WHERE id = ? LIMIT 1", user.ID)

	if err != nil {
		tx.Rollback()

		slog.Error("Unable to update profile.", slog.String("error", err.Error()), slog.String("area", "db select id"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to update profile.",
			}},
		})
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Unable to update profile.", slog.String("error", err.Error()), slog.String("area", "db commit"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to update profile.",
			}},
		})
	}

	p, err := json.Marshal(updatedUser)

	if err != nil {
		slog.Error("Unable to update profile.")
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to update profile.",
			}},
		})
	}

	_, err = wRdb.Set(ctx, fmt.Sprintf("user-%d", user.ID), p, 1*time.Hour).Result()

	if err != nil {
		slog.Error("Unable to update profile.")
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to update profile.",
			}},
		})
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"id":         security_helpers.Encode(user.ID, model.USERS_TYPE, user.Salt),
		"created_at": user.CreatedAt.Format(time.RFC3339),
		"name":       user.Name.String,
		"handle":     user.Handle.String,
		"email":      user.Email,
		"user_count": user.ID,
		"about":      user.About.String,
	})
}
