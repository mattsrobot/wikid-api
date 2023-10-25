package handlers

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/macwilko/exotic-auth/db/chat_users_db/model"

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

type EditCommunityInput struct {
	Name string `json:"name" validate:"required,gte=3,lte=32"`
}

func EditCommunity(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Editing community ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(EditCommunityInput)

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
		slog.Error("Unable to edit community 💀",
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
		slog.Error("Unable to edit community, input error 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": errors,
		})
	}

	handle := Truncate(strings.ToLower(c.Params("handle")), 255)

	community := model.Communities{}

	err = db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", strings.ToLower(handle))

	if err != nil {
		slog.Error("No community found 💀 ",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	hasPermission := HasCommunityPermission(user.ID, community.ID, model.ManageCommunity, db, wRdb, rRdb, ctx)

	if !hasPermission {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	handleCantEditError := func(err error) error {
		slog.Error("Unable to edit channel. 💀")

		if err != nil {

			es := strings.ToLower(err.Error())

			slog.Error(es)

			if strings.Contains(es, "duplicate") {
				return c.Status(fiber.StatusOK).JSON(&fiber.Map{
					"errors": []fiber.Map{{
						"message": "Channel name must be unique.",
					}},
				})
			}
		}

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to edit channel.",
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

	_, err = tx.Exec("UPDATE communities SET updated_at = ?, name = ? WHERE id = ?", updatedAt, input.Name, community.ID)

	if err != nil {
		slog.Error("Couldn't insert channels, db error 💀")

		return handleTxError(err)
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Couldn't commit channel")

		return handleCantEditError(err)
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{})
}
