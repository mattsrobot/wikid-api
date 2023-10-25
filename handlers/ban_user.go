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

type BanUserInput struct {
	UserID string `json:"user_id" validate:"required,lte=255"`
}

func BanUser(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Banning user ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(BanUserInput)

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
		slog.Info("Unable to ban user, input 💀",
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

	hasPermission := HasCommunityPermission(user.ID, community.ID, model.BanMembers, db, wRdb, rRdb, ctx)

	if !hasPermission {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	banedUserId, banedUserOk := security_helpers.Decode(input.UserID)

	if banedUserId == 0 || banedUserOk != model.USERS_TYPE {
		slog.Error("No user found 💀 "+handle,
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	banedUser := model.Users{}

	err = db.Get(&banedUser, "SELECT * FROM users WHERE id = ?", banedUserId)

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
			slog.Error("Unable to ban user. 💀",
				slog.String("error", err.Error()))
		}

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to ban user.",
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

	_, err = tx.Exec("DELETE FROM communities_users WHERE user_id = ? AND community_id = ?", banedUserId, community.ID)

	if err != nil {
		slog.Error("Couldn't ban user, db error 💀",
			slog.String("error", err.Error()))

		return handleTxError(err)
	}

	_, err = tx.Exec("DELETE FROM communiy_roles_users WHERE user_id = ? AND community_id = ?", banedUserId, community.ID)

	if err != nil {
		slog.Error("Couldn't ban user, db error 💀",
			slog.String("error", err.Error()))

		return handleTxError(err)
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
