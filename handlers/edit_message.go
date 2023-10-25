package handlers

import (
	"context"
	"database/sql"
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

type EditMessageInput struct {
	MessageID string `json:"message_id" validate:"required,gte=3,lte=255"`
	Text      string `json:"text" validate:"required,lte=2000"`
}

func EditMessage(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Editing message ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(EditMessageInput)

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
		slog.Info("Unable to edit message, input 💀")
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
		slog.Error("Unable to edit message, input error 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": errors,
		})
	}

	handle := Truncate(strings.ToLower(c.Params("handle")), 255)

	community := model.Communities{}

	err = db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", handle)

	if err != nil {
		slog.Info("No community found 💀 " + handle)
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	hasPermission := HasCommunityPermission(user.ID, community.ID, model.SendMessages, db, wRdb, rRdb, ctx)

	if !hasPermission {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	messageId, messageOk := security_helpers.Decode(input.MessageID)

	if messageId == 0 || messageOk != model.MESSAGES_TYPE {
		slog.Info("No channel found 💀 " + handle)
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	message := model.Messages{}

	err = db.Get(&message, "SELECT * FROM messages WHERE id = ? AND user_id = ? AND community_id = ?", messageId, user.ID, community.ID)

	if err != nil {
		slog.Error("No message found 💀 "+handle,
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	handleCantEditError := func(err error) error {
		if err != nil {
			slog.Error("Can't edit message 💀 "+handle,
				slog.String("error", err.Error()))
		} else {
			slog.Error("Can't edit message 💀")
		}

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to edit message.",
			}},
		})
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	if err != nil {
		slog.Error("Couldn't begin tx, db error 💀")

		return handleCantEditError(err)
	}

	handleTxError := func(err error) error {
		tx.Rollback()

		return handleCantEditError(err)
	}

	updatedAt := time.Now()

	_, err = tx.Exec("UPDATE messages SET text = ?, updated_at = ?, edited = ? WHERE id = ?", input.Text, updatedAt, true, messageId)

	if err != nil {
		slog.Error("Couldn't edit messages, db error 💀")

		return handleTxError(err)
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Couldn't commit message")

		return handleCantEditError(err)
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"id":         input.MessageID,
		"created_at": message.CreatedAt.Format(time.RFC3339),
		"update_at":  updatedAt.Format(time.RFC3339),
		"text":       input.Text,
		"edited":     true,
	})
}
