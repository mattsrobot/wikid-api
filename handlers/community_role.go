package handlers

import (
	"context"
	"maps"
	"strings"

	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/security_helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

func CommunityRole(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	slog.Info("Starting fetch community role ✅")

	user, userOk := c.Locals("viewer").(model.Users)

	if !userOk {
		slog.Warn("Not allowed",
			slog.String("area", "User didn't parse right"))

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	community := model.Communities{}

	handle := Truncate(strings.ToLower(c.Params("handle")), 255)

	err := db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", handle)

	if err != nil {
		slog.Error("No community found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "can't find this community"))

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	hasPermission := HasCommunityPermission(user.ID, community.ID, model.ManageCommunity, db, wRdb, rRdb, ctx)

	if !hasPermission {
		slog.Warn("Not allowed",
			slog.String("area", "No permission bit"))

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	roleId, roleOk := security_helpers.Decode(Truncate(c.Params("roleId"), 255))

	if roleId == 0 || roleOk != model.COMMUNITY_ROLES_TYPE {
		slog.Error("Can't find community role 💀",
			slog.String("error", err.Error()),
			slog.String("area", "security id decode failed"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	role := model.CommunityRoles{}

	err = db.Get(&role, "SELECT * FROM community_roles WHERE id = ? LIMIT 1", roleId)

	if err != nil {
		slog.Error("No channel role found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "can't find community role by id"))

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	if role.CommunityID != community.ID {
		slog.Warn("Not allowed",
			slog.String("area", "community ID didn't match"))

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	roleUsers := []model.CommuniyRolesUsers{}

	err = db.Select(&roleUsers, "SELECT * FROM community_roles_users WHERE community_role_id = ?", role.ID)

	if err != nil {
		slog.Error("No channel role found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "can't find community user roles"))

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	mIds := make([]uint64, len(roleUsers))

	for i, ru := range roleUsers {
		mIds[i] = ru.UserID
	}

	sIds := make([]string, len(roleUsers))

	if len(mIds) > 0 {
		users := []model.Users{}

		usersQuery, usersArgs, err := sqlx.In("SELECT * FROM users WHERE id IN (?) ORDER BY name ASC", mIds)

		if err != nil {
			slog.Error("Database problem 💀",
				slog.String("error", err.Error()),
				slog.String("area", "selecting users IN"))

			return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		usersQuery = db.Rebind(usersQuery)

		err = db.Select(&users, usersQuery, usersArgs...)

		if err != nil {
			slog.Error("Database problem 💀",
				slog.String("error", err.Error()),
				slog.String("area", "after the bind to users query"))

			return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		for i, ru := range users {
			sIds[i] = security_helpers.Encode(ru.ID, model.USERS_TYPE, ru.Salt)
		}
	}

	mRole := role.ToFiberMap(true)

	withMembers := fiber.Map{
		"members": sIds,
	}

	maps.Copy(mRole, withMembers)

	return c.Status(fiber.StatusOK).JSON(mRole)
}
