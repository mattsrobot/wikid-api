package handlers

import (
	"context"
	"database/sql"
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
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type EditChannelInput struct {
	ChannelID string  `json:"channel_id" validate:"required,gte=3,lte=255"`
	Name      string  `json:"name" validate:"required,gte=3,lte=32"`
	GroupID   *string `json:"group_id" validate:"omitempty,lte=255"`
}

func EditChannel(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Editing channel ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(EditChannelInput)

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

	handle := Truncate(strings.ToLower(c.Params("handle")), 255)

	community := model.Communities{}

	err = db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", strings.ToLower(handle))

	if err != nil {
		slog.Info("No community found 💀 " + handle)
		slog.Error(err.Error())

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

	channel := model.Channels{}

	channelId, channelOk := security_helpers.Decode(input.ChannelID)

	if channelId == 0 || channelOk != model.CHANNELS_TYPE {
		slog.Info("No channel found 💀 " + handle)
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	err = db.Get(&channel, "SELECT * FROM channels WHERE id = ?", channelId)

	if err != nil {
		slog.Info("No channel found 💀 " + handle)
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	if channel.CommunityID != community.ID {
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	group := model.ChannelGroups{}

	if input.GroupID != nil && *input.GroupID != "" {

		groupId, groupOk := security_helpers.Decode(*input.GroupID)

		if groupId == 0 || groupOk != model.CHANNEL_GROUPS_TYPE {
			slog.Info("No group found 💀 " + handle)
			slog.Error(err.Error())

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		err = db.Get(&group, "SELECT * FROM channel_groups WHERE id = ? AND community_id = ?", groupId, community.ID)

		if err != nil {
			slog.Info("No group found 💀 " + handle)
			slog.Error(err.Error())

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}
	}

	if group.ID > 0 && group.CommunityID != community.ID {
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
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

	m1 := regexp.MustCompile(`\s`)
	channelHandle := m1.ReplaceAllString(strings.ToLower(input.Name), "-")
	m2 := regexp.MustCompile(`[^a-z0-9-]`)
	channelHandle = m2.ReplaceAllString(channelHandle, "")

	updatedAt := time.Now()

	_, err = tx.Exec("UPDATE channels SET updated_at = ?, name = ?, handle = ?, group_id = ? WHERE id = ?", updatedAt, input.Name, channelHandle, group.ID, channel.ID)

	if err != nil {
		slog.Error("Couldn't insert channels, db error 💀")

		return handleTxError(err)
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Couldn't commit channel")

		return handleCantEditError(err)
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"id":         input.ChannelID,
		"created_at": channel.CreatedAt.Format(time.RFC3339),
		"update_at":  updatedAt.Format(time.RFC3339),
		"name":       input.Name,
		"handle":     channelHandle,
	})
}
