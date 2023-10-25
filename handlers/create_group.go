package handlers

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"
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

type CreateGroupInput struct {
	Name string `json:"name" validate:"required,gte=3,lte=32"`
}

func CreateGroup(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Creating group ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(CreateGroupInput)

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
		slog.Warn("Unable to create group, input 💀",
			slog.String("error", err.Error()),
			slog.String("area", "input validation"))

		errs := err.(validator.ValidationErrors)

		for _, v := range errs {
			errors = append(errors, fiber.Map{
				"field":   v.Field(),
				"message": v.Translate(trans),
			})
		}
	}

	if len(errors) > 0 {
		slog.Error("Unable to create channel, input error 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": errors,
		})
	}

	handle := Truncate(strings.ToLower(c.Params("handle")), 255)

	community := model.Communities{}

	err = db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", handle)

	if err != nil {
		slog.Error("Community not found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "community not found creating a group"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	hasPermission := HasCommunityPermission(user.ID, community.ID, model.ManageChannels, db, wRdb, rRdb, ctx)

	if !hasPermission {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	handleCantCreateError := func(err error) error {
		slog.Error("Unable to create channel. 💀")

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
				"message": "Unable to create channel.",
			}},
		})
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	if err != nil {
		slog.Error("Couldn't begin tx, db error 💀")

		return handleCantCreateError(err)
	}

	handleTxError := func(err error) error {
		tx.Rollback()

		return handleCantCreateError(err)
	}

	iq := `
	INSERT INTO channel_groups
	(created_at, object_salt, community_id, name)
	VALUES (?, ?, ?, ?)
	`

	salt := uuid.New().String()

	createdAt := time.Now()

	_, err = tx.Exec(iq, createdAt, salt, community.ID, input.Name)

	if err != nil {
		slog.Error("Couldn't insert channels, db error 💀")

		return handleTxError(err)
	}

	var groupId uint64

	err = tx.Get(&groupId, "SELECT LAST_INSERT_ID()")

	if err != nil {
		slog.Error("Couldn't get last insert for groups, db error 💀")

		return handleTxError(err)
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Couldn't commit group")

		return handleCantCreateError(err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"id":         security_helpers.Encode(groupId, model.CHANNEL_GROUPS_TYPE, salt),
		"created_at": createdAt.Format(time.RFC3339),
		"name":       input.Name,
	})
}
