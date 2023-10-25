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

type EditCommunityDefaultPermissionsInput struct {
	ViewChannels    *bool `json:"view_channels" validate:"required"`
	ManageChannels  *bool `json:"manage_channels" validate:"required"`
	ManageCommunity *bool `json:"manage_community" validate:"required"`
	CreateInvite    *bool `json:"create_invite" validate:"required"`
	KickMembers     *bool `json:"kick_members" validate:"required"`
	BanMembers      *bool `json:"ban_members" validate:"required"`
	SendMessages    *bool `json:"send_messages" validate:"required"`
	AttachMedia     *bool `json:"attach_media" validate:"required"`
}

func EditCommunityDefaultPermissions(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Editing community default permissions ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(EditCommunityDefaultPermissionsInput)

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
		slog.Error("Unable to edit role 💀",
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
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": errors,
		})
	}

	handle := Truncate(strings.ToLower(c.Params("handle")), 255)

	community := model.Communities{}

	err = db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", handle)

	if err != nil {
		slog.Error("No community found 💀", slog.String("error", err.Error()), slog.String("area", "can't find this community"))

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

	updatedAt := time.Now()

	handleCantEditError := func(err error, reason string) error {

		if err != nil {
			slog.Error("Can't edit community role 💀",
				slog.String("error", err.Error()),
				slog.String("area", reason))
		}

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to edit role.",
			}},
		})
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	if err != nil {
		return handleCantEditError(err, "Couldn't begin tx, db error 💀")
	}

	handleTxError := func(err error, reason string) error {
		tx.Rollback()

		return handleCantEditError(err, reason)
	}

	uq := `
		UPDATE communities
		SET updated_at = ?,
			view_channels = ?,
			manage_channels = ?,
			manage_community = ?,
			create_invite = ?,
			kick_members = ?,
			ban_members = ?,
			send_messages = ?,
			attach_media = ?
		WHERE id = ?
	`

	_, err = tx.Exec(uq, updatedAt, input.ViewChannels, input.ManageChannels, input.ManageCommunity,
		input.CreateInvite, input.KickMembers, input.BanMembers, input.SendMessages, input.AttachMedia, community.ID)

	if err != nil {
		return handleTxError(err, "Couldn't insert roles, db error 💀")
	}

	var rolesUserIds []uint64

	err = tx.Select(&rolesUserIds, "SELECT user_id FROM communities_users WHERE community_id = ?", community.ID)

	if err != nil {
		return handleTxError(err, "Couldn't find roles, db error 💀")
	}

	err = db.Get(&community, "SELECT * FROM communities WHERE id = ? LIMIT 1", community.ID)

	if err != nil {
		return handleTxError(err, "Couldn't get latest community info 💀")
	}

	err = RecalculateAndUpdatePermissionsForUsers(rolesUserIds, community, tx, wRdb, wRdb, ctx)

	if err != nil {
		return handleTxError(err, "Couldn't recalculate user permissions, db error 💀")
	}

	err = tx.Commit()

	if err != nil {
		return handleCantEditError(err, "Couldn't commit role edit")
	}

	return c.Status(fiber.StatusOK).JSON(community.Permissions.ToFiberMap())
}
