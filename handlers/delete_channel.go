package handlers

import (
	"context"
	"database/sql"
	"strings"

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

type DeleteChannelInput struct {
	ChannelID string `json:"channel_id" validate:"required,gte=3,lte=255"`
}

func DeleteChannel(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Deleteing channel ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(DeleteChannelInput)

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
		slog.Error("Unable to delete group 💀",
			slog.String("error", err.Error()),
			slog.String("area", "input doesnt validate"))

		errs := err.(validator.ValidationErrors)

		for _, v := range errs {
			errors = append(errors, fiber.Map{
				"field":   v.Field(),
				"message": v.Translate(trans),
			})
		}
	}

	if len(errors) > 0 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"errors": errors,
		})
	}

	handle := Truncate(strings.ToLower(c.Params("handle")), 255)

	community := model.Communities{}

	err = db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", handle)

	if err != nil {
		slog.Error("No community found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "can't find this community"))

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	hasPermission := HasCommunityPermission(user.ID, community.ID, model.ManageChannels, db, wRdb, rRdb, ctx)

	if !hasPermission {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	channelId, channelOk := security_helpers.Decode(input.ChannelID)

	if channelId == 0 || channelOk != model.CHANNELS_TYPE {
		slog.Error("No channel found 💀",
			slog.String("error", err.Error()),
			slog.Uint64("cid", channelId),
			slog.String("ctype", channelOk),
			slog.String("area", "can't find this channel"))

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	handleCantDeleteError := func(err error, reason string) error {

		if err != nil {
			slog.Error("Can't delete channel 💀",
				slog.String("error", err.Error()),
				slog.String("area", reason))
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to delete channel.",
			}},
		})
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	if err != nil {
		return handleCantDeleteError(err, "Couldn't begin tx, db error 💀")
	}

	handleTxError := func(err error, reason string) error {
		tx.Rollback()

		return handleCantDeleteError(err, reason)
	}

	uq := `
		DELETE FROM channels
		WHERE id = ?
		AND community_id = ?
	`

	_, err = tx.Exec(uq, channelId, community.ID)

	if err != nil {
		return handleTxError(err, "Couldn't delete channel, db error 💀")
	}

	uq = `
		DELETE FROM messages
		WHERE channel_id = ?
		AND community_id = ?
	`

	_, err = tx.Exec(uq, channelId, community.ID)

	if err != nil {
		return handleTxError(err, "Couldn't delete messages, db error 💀")
	}

	err = tx.Commit()

	if err != nil {
		return handleCantDeleteError(err, "Couldn't commit channel delete")
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{"ok": true})
}
