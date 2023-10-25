package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/security_helpers"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type CreateCommunityInput struct {
	Name    string `json:"name" validate:"required,gte=3,lte=50"`
	Handle  string `json:"handle" validate:"required,gte=3,lte=32"`
	Private *bool  `json:"private" validate:"required"`
}

func CreateCommunity(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, rdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Creating community ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(CreateCommunityInput)

	if err := c.BodyParser(input); err != nil {
		slog.Warn("Invalid input 💀")

		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
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
		slog.Info("Unable to create community, input 💀")
		slog.Info(err.Error())

		errs := err.(validator.ValidationErrors)

		for _, v := range errs {
			errors = append(errors, fiber.Map{
				"field":   v.Field(),
				"message": v.Translate(trans),
			})
		}
	}

	lowerHandle := strings.ToLower(input.Handle)

	match, _ := regexp.MatchString("[a-z-]+", lowerHandle)

	if !match {
		slog.Error("Lower handle didn't match regex, db error 💀")

		errors = append(errors, fiber.Map{
			"field":   "handle",
			"message": "Unique name must be letters and _ only.",
		})
	}

	var handleCount int
	err = db.Get(&handleCount, "SELECT count(*) FROM communities WHERE handle = ?", lowerHandle)

	if handleCount > 0 {
		slog.Error("Handle already taken 💀")

		errors = append(errors, fiber.Map{
			"field":   "handle",
			"message": "Unique name already taken",
		})
	}

	if err != nil {
		slog.Error("Unable to create community, db error 💀")
		slog.Error(err.Error())

		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "handle",
				"message": "Unable to update profile.",
			}},
		})
	}

	if len(errors) > 0 {
		slog.Error("Unable to create community, input error 💀")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": errors,
		})
	}

	salt := uuid.New().String()
	createdAt := time.Now()

	handleCantCreateError := func(err error) error {
		slog.Error("Unable to create community. 💀")

		if err != nil {
			slog.Error(err.Error())
		}

		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to create community.",
			}},
		})
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	if err != nil {
		slog.Error("Couldn't being tx, db error 💀")

		return handleCantCreateError(err)
	}

	handleTxError := func(err error) error {
		tx.Rollback()

		return handleCantCreateError(err)
	}

	fqn := fmt.Sprintf("%s.%s", lowerHandle, os.Getenv("EXOTIC_FQN"))

	ic := `INSERT INTO communities
	(created_at, object_salt, owner_id, name, self_hosted, fqn, private, handle)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(ic, createdAt, salt, user.ID, input.Name, false, fqn, input.Private, lowerHandle)

	if err != nil {
		slog.Error("Couldn't insert communities, db error 💀")

		return handleTxError(err)
	}

	var communityId uint64

	err = tx.Get(&communityId, "SELECT LAST_INSERT_ID()")

	if err != nil {
		slog.Error("Couldn't get last insert for communities, db error 💀")

		return handleTxError(err)
	}

	_, err = tx.Exec("INSERT INTO channel_groups (created_at, object_salt, community_id, name) VALUES (?, ?, ?, ?)", createdAt, salt, communityId, "Text channels")

	if err != nil {
		slog.Error("Couldn't insert into channel_groups, db error 💀")

		return handleTxError(err)
	}

	var groupId uint64

	err = tx.Get(&groupId, "SELECT LAST_INSERT_ID()")

	if err != nil {
		slog.Error("Couldn't get last insert for groups, db error 💀")

		return handleTxError(err)
	}

	_, err = tx.Exec("INSERT INTO channels (created_at, object_salt, community_id, group_id, name, handle) VALUES (?, ?, ?, ?, ?, ?)", createdAt, salt, communityId, groupId, "🤭 chit chat", "chit-chat")

	if err != nil {
		slog.Error("Couldn't insert into channels, db error 💀")

		return handleTxError(err)
	}

	var channelId uint64

	err = tx.Get(&channelId, "SELECT LAST_INSERT_ID()")

	if err != nil {
		slog.Error("Couldn't get last insert for channels, db error 💀")

		return handleTxError(err)
	}

	ud := `INSERT INTO communities_users
	(created_at, user_id, community_id, view_channels, manage_channels, manage_community, create_invite,
	kick_members, ban_members, send_messages, attach_media, selected_channel_id)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.Exec(ud, createdAt, user.ID, communityId, true, true, true, true, true, true, true, true, channelId)

	if err != nil {
		slog.Error("Couldn't insert into communities users, db error 💀")

		return handleTxError(err)
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Couldn't commit community")

		return handleCantCreateError(err)
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"id":         security_helpers.Encode(communityId, model.COMMUNITIES_TYPE, salt),
		"created_at": createdAt.Format(time.RFC3339),
		"name":       input.Name,
		"handle":     lowerHandle,
	})
}
