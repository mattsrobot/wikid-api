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

type JoinCommunityInput struct {
	Code *string `json:"code" validate:"omitempty,lte=255"`
}

func JoinCommunity(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Joining community ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(JoinCommunityInput)

	if err := c.BodyParser(input); err != nil {
		slog.Warn("Invalid input 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"error": "Invalid input.",
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
		slog.Info("Unable to join community, input 💀")
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
		slog.Error("Unable to join community, input error 💀")

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
				"message": "Not found.",
			}},
		})
	}

	if community.Private {
		if input.Code == nil {
			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Please provide a valid invite code to join.",
				}},
			})
		}

		invite := model.CommunityInvites{}

		err = db.Get(&invite, "SELECT * FROM community_invites WHERE code = ? LIMIT 1", *input.Code)

		if err != nil {
			slog.Info("No community found 💀 " + handle)
			slog.Error(err.Error())

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		if invite.CommunityID != community.ID {
			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Please provide a valid invite code to join.",
				}},
			})
		}
	}

	createdAt := time.Now()

	handleError := func(err error) error {
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

		return handleError(err)
	}

	var banned uint64

	err = db.Get(&banned, "SELECT count(*) FROM communities_banned_users WHERE user_id = ? AND community_id", user.ID, community.ID)

	if err != nil {
		slog.Error("Database problem 💀",
			slog.String("error", err.Error()),
			slog.String("area", "can't select count of messages"))

		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	if banned > 0 {
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "You are banned from this community.",
			}},
		})
	}

	handleTxError := func(err error) error {
		tx.Rollback()

		return handleError(err)
	}

	icu := `
	INSERT INTO communities_users
	(created_at, community_id, user_id, view_channels, manage_channels, manage_community, create_invite,
	kick_members, ban_members, send_messages, attach_media)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(icu, createdAt, community.ID, user.ID, community.ViewChannels, community.ManageChannels,
		community.ManageCommunity, community.CreateInvite, community.KickMembers, community.BanMembers,
		community.SendMessages, community.AttachMedia)

	if err != nil {
		slog.Error("Couldn't insert channels, db error 💀")

		return handleTxError(err)
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Couldn't commit channel")

		return handleError(err)
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"ok": true,
	})
}
