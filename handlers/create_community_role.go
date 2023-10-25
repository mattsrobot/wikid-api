package handlers

import (
	"context"
	"database/sql"
	"slices"
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

type CreateCommunityRoleInput struct {
	Name                  string   `json:"name" validate:"required,gte=3,lte=255"`
	Color                 string   `json:"color" validate:"required,gte=3,lte=255"`
	ShowOnlineDifferently *bool    `json:"show_online_differently" validate:"required"`
	ViewChannels          *bool    `json:"view_channels" validate:"required"`
	ManageChannels        *bool    `json:"manage_channels" validate:"required"`
	ManageCommunity       *bool    `json:"manage_community" validate:"required"`
	CreateInvite          *bool    `json:"create_invite" validate:"required"`
	KickMembers           *bool    `json:"kick_members" validate:"required"`
	BanMembers            *bool    `json:"ban_members" validate:"required"`
	SendMessages          *bool    `json:"send_messages" validate:"required"`
	AttachMedia           *bool    `json:"attach_media" validate:"required"`
	Members               []string `json:"members" validate:"required"`
}

func CreateCommunityRole(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Creating community role ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(CreateCommunityRoleInput)

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
		slog.Error("Unable to create role 💀",
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

	if !slices.Contains(ValidColors, input.Color) {
		errors = append(errors, fiber.Map{
			"field":   "color",
			"message": "Not an acceptable color value",
		})
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
		slog.Error("No community found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "can't find this community"))

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

	salt := uuid.New().String()

	createdAt := time.Now()

	handleCantCreateError := func(err error, reason string) error {

		if err != nil {
			slog.Error("Can't create community role 💀",
				slog.String("error", err.Error()),
				slog.String("area", reason))
		}

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to create role.",
			}},
		})
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	if err != nil {
		return handleCantCreateError(err, "Couldn't begin tx, db error 💀")
	}

	handleTxError := func(err error, reason string) error {
		tx.Rollback()

		return handleCantCreateError(err, reason)
	}

	var roleCount int
	err = tx.Get(&roleCount, "SELECT count(*) FROM community_roles WHERE community_id = ?", community.ID)

	if err != nil {
		return handleTxError(err, "Couldn't insert roles, db error 💀")
	}

	insertStmt := `
		INSERT INTO community_roles
		(created_at, object_salt, community_id, show_online_differently, priority, name, view_channels, manage_channels,
		manage_community, create_invite, kick_members, ban_members, send_messages, attach_media, color)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.Exec(insertStmt, createdAt, salt, community.ID, input.ShowOnlineDifferently, roleCount, input.Name,
		input.ViewChannels, input.ManageChannels, input.ManageCommunity, input.CreateInvite, input.KickMembers,
		input.BanMembers, input.SendMessages, input.AttachMedia, input.Color)

	if err != nil {
		return handleTxError(err, "Couldn't insert roles, db error 💀")
	}

	var roleId uint64

	err = tx.Get(&roleId, "SELECT LAST_INSERT_ID()")

	if err != nil {
		return handleTxError(err, "Couldn't get last insert for messages, db error 💀")
	}

	var uIds = []uint64{}

	for _, uIdStr := range input.Members {
		if uid, ok := security_helpers.Decode(uIdStr); ok == model.USERS_TYPE {
			uIds = append(uIds, uid)
		}
	}

	if len(uIds) > 0 {

		rolesUsersQuery, rolesUsersArgs, err := sqlx.In("SELECT * FROM communities_users WHERE community_id = ? AND user_id IN (?) ", community.ID, uIds)

		if err != nil {
			return handleTxError(err, "Couldn't get user roles 💀")
		}

		rolesUsersQuery = tx.Rebind(rolesUsersQuery)

		rolesUsers := []model.CommunitiesUsers{}

		err = tx.Select(&rolesUsers, rolesUsersQuery, rolesUsersArgs...)

		if err != nil {
			return handleTxError(err, "After the bind selecting roles 💀")
		}

		for _, roleUser := range rolesUsers {
			rk := model.PermissionRedisKey(roleUser.UserID, community.ID)

			_, err = wRdb.Del(ctx, rk).Result()

			if err != nil {
				return handleTxError(err, "Couldn't delete roles, redis error 💀")
			}

			insertRoleStmt := `
			INSERT INTO community_roles_users
			(created_at, community_role_id, user_id, community_id)
			VALUES (?, ?, ?, ?)`

			_, err = tx.Exec(insertRoleStmt, createdAt, roleId, roleUser.UserID, community.ID)

			if input.ViewChannels != nil && *input.ViewChannels && !roleUser.ViewChannels {
				_, err = tx.Exec("UPDATE communities_users SET view_channels = ? WHERE user_id = ? AND community_id =?", true, roleUser.UserID, roleUser.CommunityID)
			}

			if input.ManageChannels != nil && *input.ManageChannels && !roleUser.ManageChannels {
				_, err = tx.Exec("UPDATE communities_users SET manage_channels = ? WHERE user_id = ? AND community_id =?", true, roleUser.UserID, roleUser.CommunityID)
			}

			if input.ManageCommunity != nil && *input.ManageCommunity && !roleUser.ManageCommunity {
				_, err = tx.Exec("UPDATE communities_users SET manage_community = ? WHERE user_id = ? AND community_id =?", true, roleUser.UserID, roleUser.CommunityID)
			}

			if input.CreateInvite != nil && *input.CreateInvite && !roleUser.CreateInvite {
				_, err = tx.Exec("UPDATE communities_users SET create_invite = ? WHERE user_id = ? AND community_id =?", true, roleUser.UserID, roleUser.CommunityID)
			}

			if input.KickMembers != nil && *input.KickMembers && !roleUser.KickMembers {
				_, err = tx.Exec("UPDATE communities_users SET kick_members = ? WHERE user_id = ? AND community_id =?", true, roleUser.UserID, roleUser.CommunityID)
			}

			if input.BanMembers != nil && *input.BanMembers && !roleUser.BanMembers {
				_, err = tx.Exec("UPDATE communities_users SET ban_members = ? WHERE user_id = ? AND community_id =?", true, roleUser.UserID, roleUser.CommunityID)
			}

			if input.SendMessages != nil && *input.SendMessages && !roleUser.SendMessages {
				_, err = tx.Exec("UPDATE communities_users SET send_messages = ? WHERE user_id = ? AND community_id =?", true, roleUser.UserID, roleUser.CommunityID)
			}

			if input.AttachMedia != nil && *input.AttachMedia && !roleUser.AttachMedia {
				_, err = tx.Exec("UPDATE communities_users SET attach_media = ? WHERE user_id = ? AND community_id =?", true, roleUser.UserID, roleUser.CommunityID)
			}

			if err != nil {
				return handleTxError(err, "Couldn't insert roles, db error 💀")
			}
		}
	}

	err = tx.Commit()

	if err != nil {
		return handleCantCreateError(err, "Couldn't commit role")
	}

	communityRole := model.CommunityRoles{}

	err = db.Get(&communityRole, "SELECT * FROM community_roles WHERE id = ?", roleId)

	if err != nil {
		slog.Error("No community found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "can't find this community"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	return c.Status(fiber.StatusOK).JSON(communityRole.ToFiberMap(true))
}
