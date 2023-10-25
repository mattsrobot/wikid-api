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

type KickUserInput struct {
	UserID string `json:"user_id" validate:"required,lte=255"`
}

func KickUser(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Kicking user ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(KickUserInput)

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
		slog.Info("Unable to kick user, input 💀",
			slog.String("error", err.Error()))

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

	handle := Truncate(strings.ToLower(c.Params("handle")), 255)

	community := model.Communities{}

	err = db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", strings.ToLower(handle))

	if err != nil {
		slog.Info("No community found 💀 "+handle,
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	hasPermission := HasCommunityPermission(user.ID, community.ID, model.KickMembers, db, wRdb, rRdb, ctx)

	if !hasPermission {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	kickedUserId, kickedUserOk := security_helpers.Decode(input.UserID)

	if kickedUserId == 0 || kickedUserOk != model.USERS_TYPE {
		slog.Error("No user found 💀 "+handle,
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	kickedUser := model.Users{}

	err = db.Get(&kickedUser, "SELECT * FROM users WHERE id = ?", kickedUserId)

	if err != nil {
		slog.Error("No user found 💀 "+handle,
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	handleError := func(err error) error {

		if err != nil {
			slog.Error("Unable to kick user. 💀",
				slog.String("error", err.Error()))
		}

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to kick user.",
			}},
		})
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	if err != nil {
		slog.Error("Couldn't begin tx, db error 💀")

		return handleError(err)
	}

	handleTxError := func(err error) error {
		tx.Rollback()

		return handleError(err)
	}

	_, err = tx.Exec("DELETE FROM communities_users WHERE user_id = ? AND community_id = ?", kickedUserId, community.ID)

	if err != nil {
		slog.Error("Couldn't kick user, db error 💀",
			slog.String("error", err.Error()))

		return handleTxError(err)
	}

	_, err = tx.Exec("DELETE FROM communiy_roles_users WHERE user_id = ? AND community_id = ?", kickedUserId, community.ID)

	if err != nil {
		slog.Error("Couldn't kick user, db error 💀",
			slog.String("error", err.Error()))

		return handleTxError(err)
	}

	_, err = tx.Exec("INSERT INTO communities_banned_users (created_at, community_id, user_id) VALUES (?, ?, ?)", time.Now(), community.ID, kickedUser.ID)

	if err != nil {
		tx.Rollback()

		slog.Error("Couldn't kick user, db error 💀",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to kick user.",
			}},
		})
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Couldn't commit channel")

		return handleError(err)
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"updated": true,
	})
}
