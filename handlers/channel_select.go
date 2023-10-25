package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/macwilko/exotic-auth/db/chat_users_db/model"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type ChannelSelectInput struct {
	ChannelHandle string `json:"channel_handle" validate:"required,lte=255"`
}

func ChannelSelect(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	slog.Info("Starting channel select ✅")

	user, userOk := c.Locals("viewer").(model.Users)

	if !userOk {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(ChannelSelectInput)

	if err := c.BodyParser(input); err != nil {
		slog.Warn("Invalid input 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"error": "Invalid input",
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
		slog.Info("Unable to edit community, input 💀")
		slog.Info(err.Error())

		errs := err.(validator.ValidationErrors)

		for _, v := range errs {
			errors = append(errors, fiber.Map{
				"field":   v.Field(),
				"message": v.Translate(trans),
			})
		}
	}

	if len(errors) > 0 {
		slog.Error("Unable to edit channel, input error 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": errors,
		})
	}

	/* Fetch community */

	communityHandle := Truncate(strings.ToLower(c.Params("handle")), 255)

	community := model.Communities{}

	err = db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", communityHandle)

	if err != nil {
		slog.Error("No community found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "can't find  community"))

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	/* Got community  */

	/* Fetch channel */

	channelHandle := Truncate(strings.ToLower(input.ChannelHandle), 255)

	channel := model.Channels{}

	err = db.Get(&channel, "SELECT * FROM channels WHERE handle = ? AND community_id = ? LIMIT 1", channelHandle, community.ID)

	if err != nil {
		slog.Error("No channel found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "can't find  channel"))

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	hasPermission := HasChannelPermission(user.ID, channel.ID, model.ViewChannels, db, wRdb, rRdb, ctx)

	if !hasPermission {
		slog.Warn("Not allowed",
			slog.String("area", "No permission bit"))

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	go func() {
		_, err = wRdb.Set(ctx, fmt.Sprintf("user-%d-channel-%d", user.ID, channel.ID), time.Now().Format(time.RFC3339), 0).Result()

		if err != nil {
			slog.Error("Couldn't update communities redis for channel 💀",
				slog.String("error", err.Error()),
				slog.String("area", "not sure"))
		}
	}()

	_, err = db.Exec("UPDATE communities_users SET selected_channel_id = ? WHERE user_id = ? AND community_id = ?", channel.ID, user.ID, community.ID)

	if err != nil {
		slog.Error("Couldn't update communities_users 💀",
			slog.String("error", err.Error()),
			slog.String("area", "not sure"))

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{"updated": true})
}
