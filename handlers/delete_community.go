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

func DeleteCommunity(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, rdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Deleting community ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	lowerHandle := Truncate(strings.ToLower(c.Params("handle")), 255)

	var community model.Communities

	err := db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", lowerHandle)

	if err != nil {
		slog.Error("Unable to delete community, db error 💀")
		slog.Error(err.Error())

		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "handle",
				"message": "Unable to delete.",
			}},
		})
	}

	if user.ID != community.OwnerID {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	handleCantDeleteError := func(err error) error {
		slog.Error("Unable to delete community. 💀")

		if err != nil {
			slog.Error(err.Error())
		}

		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to delete community.",
			}},
		})
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	if err != nil {
		slog.Error("Couldn't create tx, db error 💀")

		return handleCantDeleteError(err)
	}

	handleTxError := func(err error, reason string) error {
		tx.Rollback()

		return handleCantDeleteError(err)
	}

	_, err = tx.Exec("DELETE FROM communities_users WHERE community_id = ?", community.ID)

	if err != nil {
		return handleTxError(err, "Couldn't delete communities, db error 💀")
	}

	_, err = tx.Exec("DELETE FROM messages WHERE community_id = ?", community.ID)

	if err != nil {
		return handleTxError(err, "Couldn't delete communities, db error 💀")
	}

	_, err = tx.Exec("DELETE FROM community_roles_users WHERE community_id = ?", community.ID)

	if err != nil {
		return handleTxError(err, "Couldn't delete communities, db error 💀")
	}

	_, err = tx.Exec("DELETE FROM community_roles WHERE community_id = ?", community.ID)

	if err != nil {
		return handleTxError(err, "Couldn't delete communities, db error 💀")
	}

	_, err = tx.Exec("DELETE FROM channels WHERE community_id = ?", community.ID)

	if err != nil {
		return handleTxError(err, "Couldn't delete communities, db error 💀")
	}

	_, err = tx.Exec("DELETE FROM channel_groups WHERE community_id = ?", community.ID)

	if err != nil {
		return handleTxError(err, "Couldn't delete communities, db error 💀")
	}

	_, err = tx.Exec("DELETE FROM communities WHERE id = ?", community.ID)

	if err != nil {
		return handleTxError(err, "Couldn't delete communities, db error 💀")
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Couldn't delete community")
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{"deleted": true})
}
