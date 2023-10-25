package handlers

import (
	"context"
	"maps"
	"strings"

	"github.com/macwilko/exotic-auth/db/chat_users_db/model"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

func CommunityRoles(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	slog.Info("Starting fetch community roles ✅")

	user, userOk := c.Locals("viewer").(model.Users)

	if !userOk {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	handle := Truncate(strings.ToLower(c.Params("handle")), 255)

	community := model.Communities{}

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
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	roles := []model.CommunityRoles{}

	err = db.Select(&roles, "SELECT * FROM community_roles WHERE community_id = ? ORDER BY priority ASC", community.ID)

	if err != nil {
		slog.Error("Database issue 💀",
			slog.String("error", err.Error()),
			slog.String("area", "cant read roles"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	mr := make([]fiber.Map, len(roles))

	for i, r := range roles {
		mRole := r.ToFiberMap(true)

		var rolesCount int
		err = db.Get(&rolesCount, "SELECT count(*) FROM community_roles_users WHERE community_role_id = ?", r.ID)

		if err != nil {
			slog.Error("Database issue 💀",
				slog.String("error", err.Error()),
				slog.String("area", "cant read member count"))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		withCount := fiber.Map{
			"members": rolesCount,
		}

		maps.Copy(mRole, withCount)

		mr[i] = mRole
	}

	return c.Status(fiber.StatusOK).JSON(mr)
}
