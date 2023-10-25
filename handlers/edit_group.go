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

type EditGroupInput struct {
	GroupID string `json:"group_id" validate:"required,gte=3,lte=255"`
	Name    string `json:"name" validate:"required,gte=3,lte=32"`
}

func EditGroup(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Creating group ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(EditGroupInput)

	if err := c.BodyParser(input); err != nil {
		slog.Warn("Invalid input 💀")

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
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
		slog.Error("Unable to create group, input error 💀")

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"errors": errors,
		})
	}

	handle := Truncate(strings.ToLower(c.Params("handle")), 255)

	community := model.Communities{}

	err = db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", handle)

	if err != nil {
		slog.Error("Community not found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "community not found editing a group"))

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

	groupId, groupOk := security_helpers.Decode(input.GroupID)

	if groupId == 0 || groupOk != model.CHANNEL_GROUPS_TYPE {
		slog.Error("Can't find group 💀",
			slog.String("error", err.Error()),
			slog.String("area", "security id decode failed"))

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	group := model.ChannelGroups{}

	err = db.Get(&group, "SELECT * FROM channel_groups WHERE id = ? AND community_id = ? LIMIT 1", groupId, community.ID)

	if err != nil {
		slog.Error("Group not found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "group not found editing a group"))

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	handleCantCreateError := func(err error) error {
		slog.Error("Unable to create channel. 💀")

		if err != nil {
			slog.Error(err.Error())
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
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
	UPDATE channel_groups
	SET name = ?,
		updated_at = ?
	WHERE id = ?
	`

	updatedAt := time.Now()

	_, err = tx.Exec(iq, input.Name, updatedAt, groupId)

	if err != nil {
		slog.Error("Couldn't edit goup, db error 💀")

		return handleTxError(err)
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Couldn't commit group")

		return handleCantCreateError(err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"id":         input.GroupID,
		"created_at": group.CreatedAt.Format(time.RFC3339),
		"update_at":  updatedAt.Format(time.RFC3339),
		"name":       input.Name,
	})
}
