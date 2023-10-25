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

type DeleteCommunityRoleInput struct {
	ID string `json:"id" validate:"required,gte=3,lte=255"`
}

func DeleteCommunityRole(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Deleteing community role ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(DeleteCommunityRoleInput)

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
		slog.Error("Unable to delete role 💀",
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

	handleCantDeleteError := func(err error, reason string) error {

		if err != nil {
			slog.Error("Can't delete community role 💀",
				slog.String("error", err.Error()),
				slog.String("area", reason))
		}

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to delete role.",
			}},
		})
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	if err != nil {
		return handleCantDeleteError(err, "Couldn't begin tx, db error 💀")
	}

	handleTxError := func(err error, reason string) error {
		tx.Rollback()

		return handleCantDeleteError(err, reason)
	}

	uq := `
		DELETE FROM community_roles
		WHERE id = ?
	`

	_, err = tx.Exec(uq, roleId)

	if err != nil {
		return handleTxError(err, "Couldn't delete roles, db error 💀")
	}

	var rolesUserIds []uint64

	err = tx.Select(&rolesUserIds, "SELECT user_id FROM community_roles_users WHERE community_id = ?", community.ID)

	if err != nil {
		return handleTxError(err, "Couldn't find roles, db error 💀")
	}

	if len(rolesUserIds) > 0 {
		deleteUsersQuery, deleteUsersArgs, err := sqlx.In("DELETE FROM community_roles_users WHERE community_role_id = ? AND user_id IN (?) ", roleId, rolesUserIds)

		if err != nil {
			return handleTxError(err, "Couldn't delete user roles 💀")
		}

		deleteUsersQuery = tx.Rebind(deleteUsersQuery)

		_, err = tx.Exec(deleteUsersQuery, deleteUsersArgs...)

		if err != nil {
			return handleTxError(err, "Couldn't delete user roles 💀")
		}

		err = RecalculateAndUpdatePermissionsForUsers(rolesUserIds, community, tx, wRdb, rRdb, ctx)

		if err != nil {
			return handleTxError(err, "Couldn't recalculate user permissions 💀")
		}

	}

	err = tx.Commit()

	if err != nil {
		return handleCantDeleteError(err, "Couldn't commit role delete")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true})
}
