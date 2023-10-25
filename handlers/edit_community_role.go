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
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type EditCommunityRoleInput struct {
	ID                    string   `json:"id" validate:"required,gte=3,lte=255"`
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

func EditCommunityRole(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Editing community role ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(EditCommunityRoleInput)

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

	err = db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", strings.ToLower(handle))

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

	roleId, roleOk := security_helpers.Decode(input.ID)

	if roleId == 0 || roleOk != model.COMMUNITY_ROLES_TYPE {
		slog.Info("No channel found 💀 " + handle)
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
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

	if communityRole.CommunityID != community.ID {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	go func() {
		updatedAt := time.Now()

		handleCantEditError := func(err error, reason string) {

			if err != nil {
				slog.Error("Can't edit community role 💀",
					slog.String("error", err.Error()),
					slog.String("area", reason))
			}
		}

		tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

		if err != nil {
			handleCantEditError(err, "Couldn't begin tx, db error 💀")

			return
		}

		handleTxError := func(err error, reason string) {
			tx.Rollback()

			handleCantEditError(err, reason)
		}

		/* Update the community role with the new permission */

		uq := `UPDATE community_roles
			   SET updated_at = ?,
				 name = ?,
				 color = ?,
				 show_online_differently = ?,
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

		_, err = tx.Exec(uq, updatedAt, input.Name, input.Color, input.ShowOnlineDifferently,
			input.ViewChannels, input.ManageChannels, input.ManageCommunity, input.CreateInvite,
			input.KickMembers, input.BanMembers, input.SendMessages, input.AttachMedia, roleId)

		if err != nil {
			handleTxError(err, "Couldn't insert roles, db error 💀")

			return
		}

		err = tx.Commit()

		if err != nil {
			handleCantEditError(err, "Couldn't commit role edit")

			return
		}

		tx, err = db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

		if err != nil {
			handleCantEditError(err, "Couldn't begin tx, db error 💀")

			return
		}

		/* Find the current users who have this role */

		var beforeChangeUsersIds []uint64

		err = tx.Select(&beforeChangeUsersIds, "SELECT user_id FROM community_roles_users WHERE community_id = ?", community.ID)

		if err != nil {
			handleTxError(err, "Couldn't find roles, db error 💀")

			return
		}

		/* Delete community role user users for this role */

		deleteUsersQuery, deleteUsersArgs, err := sqlx.In("DELETE FROM community_roles_users WHERE community_role_id = ?", roleId)

		if err != nil {
			handleTxError(err, "Couldn't delete user roles 💀")

			return
		}

		deleteUsersQuery = tx.Rebind(deleteUsersQuery)

		_, err = tx.Exec(deleteUsersQuery, deleteUsersArgs...)

		if err != nil {
			handleTxError(err, "Couldn't delete user roles 💀")

			return
		}

		/* Reset these users permissions */

		err = RecalculateAndUpdatePermissionsForUsers(beforeChangeUsersIds, community, tx, wRdb, rRdb, ctx)

		if err != nil {
			handleTxError(err, "Couldn't recalculate users permissions 💀")

			return
		}

		/* Find the new users who should have the role */

		var afterChangeUsersIds = []uint64{}

		for _, uIdStr := range input.Members {
			if uid, object := security_helpers.Decode(uIdStr); object == model.USERS_TYPE {
				afterChangeUsersIds = append(afterChangeUsersIds, uid)
			}
		}

		/* If there's users to add with this role, add the role and recalculate their permissions */

		if len(afterChangeUsersIds) > 0 {

			rolesUsersQuery, rolesUsersArgs, err := sqlx.In("SELECT * FROM communities_users WHERE community_id = ? AND user_id IN (?) ", community.ID, afterChangeUsersIds)

			if err != nil {
				handleTxError(err, "Couldn't get user roles 💀")

				return
			}

			rolesUsersQuery = tx.Rebind(rolesUsersQuery)

			rolesUsers := []model.CommunitiesUsers{}

			err = tx.Select(&rolesUsers, rolesUsersQuery, rolesUsersArgs...)

			if err != nil {
				handleTxError(err, "After the bind selecting roles 💀")

				return
			}

			createdAt := time.Now()

			for _, roleUser := range rolesUsers {

				insertRoleStmt := `INSERT INTO community_roles_users (created_at, community_role_id, user_id, community_id)
								   VALUES (?, ?, ?, ?)`

				_, err = tx.Exec(insertRoleStmt, createdAt, roleId, roleUser.UserID, community.ID)

				if err != nil {
					handleTxError(err, "Couldn't insert roles, db error 💀")

					return
				}
			}

			err = RecalculateAndUpdatePermissionsForUsers(beforeChangeUsersIds, community, tx, wRdb, rRdb, ctx)

			if err != nil {
				handleTxError(err, "Couldn't recalculate users permissions 💀")

				return
			}
		}

		err = tx.Commit()

		if err != nil {
			handleCantEditError(err, "Couldn't commit role edit")

			return
		}
	}()

	communityRole = model.CommunityRoles{}

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
