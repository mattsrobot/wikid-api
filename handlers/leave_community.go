package handlers

import (
	"context"
	"database/sql"
	"strings"

	"github.com/macwilko/exotic-auth/db/chat_users_db/model"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

func LeaveCommunity(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Leaving community ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
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
		slog.Info("No community found 💀 " + handle)
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	if user.ID == community.OwnerID {
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "You must delete the community to leave it.",
			}},
		})
	}

	handleError := func(err error) error {
		slog.Error("Unable to leave community. 💀")

		if err != nil {
			slog.Warn(err.Error())
		}

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to leave community.",
			}},
		})
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	if err != nil {
		slog.Error("Couldn't begin tx, db error 💀")

		return handleError(err)
	}

	handleTxError := func(err error) error {
		tx.Rollback()

		return handleError(err)
	}

	dcu := `
		DELETE FROM communities_users
		WHERE user_id = ?
	`

	_, err = tx.Exec(dcu, user.ID)

	if err != nil {
		return handleTxError(err)
	}

	dcr := `
	DELETE FROM communiy_roles_users
	WHERE user_id = ?
	AND community_id = ?
	`

	_, err = tx.Exec(dcr, user.ID, community.ID)

	if err != nil {
		return handleTxError(err)
	}

	dci := `
	DELETE FROM community_invites
	WHERE user_id = ?
	AND community_id = ?
	`

	_, err = tx.Exec(dci, user.ID, community.ID)

	if err != nil {
		return handleTxError(err)
	}

	rk := model.PermissionRedisKey(user.ID, community.ID)

	_, err = wRdb.Del(ctx, rk).Result()

	if err != nil {
		return handleTxError(err)
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Unable to leave community.")

		return handleError(err)
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"ok": true,
	})
}
