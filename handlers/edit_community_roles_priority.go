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

type EditCommunityRolesPriorityInput struct {
	RoleIDs []string `json:"role_ids" validate:"required"`
}

func EditCommunityRolesPriority(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Editing community roles priority ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(EditCommunityRolesPriorityInput)

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
		slog.Error("Unable to edit roles 💀",
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

	if len(input.RoleIDs) == 0 {
		errors = append(errors, fiber.Map{
			"field":   "roles",
			"message": "You need a role to sort",
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

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	var mappedRoleIds = []uint64{}

	for _, roleIdStr := range input.RoleIDs {
		if rid, ok := security_helpers.Decode(roleIdStr); ok == model.COMMUNITY_ROLES_TYPE {
			mappedRoleIds = append(mappedRoleIds, rid)
		}
	}

	if len(mappedRoleIds) == 0 {
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	handleCantEditError := func(err error, reason string) error {

		if err != nil {
			slog.Error("Can't edit roles 💀",
				slog.String("error", err.Error()),
				slog.String("area", reason))
		}

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to edit roles.",
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

	for i, rid := range mappedRoleIds {
		uq := `
		UPDATE community_roles
		SET priority = ?
		WHERE id = ?
		AND community_id = ?`

		_, err = tx.Exec(uq, i, rid, community.ID)

		if err != nil {
			return handleTxError(err, "Can't reset roles")
		}
	}

	err = tx.Commit()

	if err != nil {
		return handleCantEditError(err, "Couldn't commit role edit")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"updated": true})
}
