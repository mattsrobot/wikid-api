package handlers

import (
	"context"
	"encoding/json"
	"fmt"
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

type ReactToMessageInput struct {
	MessageID string `json:"message_id" validate:"required,gte=3,lte=255"`
	Reaction  string `json:"reaction" validate:"required,lte=255"`
}

func ReactToMessage(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Reacting to message ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	input := new(ReactToMessageInput)

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
		slog.Error("Unable to react to message, input 💀",
			slog.String("error", err.Error()))

		errs := err.(validator.ValidationErrors)

		for _, v := range errs {
			errors = append(errors, fiber.Map{
				"field":   v.Field(),
				"message": v.Translate(trans),
			})
		}
	}

	if len(errors) > 0 {
		slog.Error("Unable to react to message, input error 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": errors,
		})
	}

	handle := Truncate(strings.ToLower(c.Params("handle")), 255)

	community := model.Communities{}

	err = db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", handle)

	if err != nil {
		slog.Error("No community found 💀 "+handle,
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	hasPermission := HasCommunityPermission(user.ID, community.ID, model.SendMessages, db, wRdb, rRdb, ctx)

	if !hasPermission {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	messageId, messageOk := security_helpers.Decode(input.MessageID)

	if messageId == 0 || messageOk != model.MESSAGES_TYPE {
		slog.Error("No message found 💀 ",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	message := model.Messages{}

	err = db.Get(&message, "SELECT * FROM messages WHERE id = ? AND community_id = ?", messageId, community.ID)

	if err != nil {
		slog.Error("No message found 💀 ",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	reactions := model.MessagesReactions{}

	rkey := fmt.Sprintf("message-reactions-%d", messageId)

	val, err := rRdb.Get(ctx, rkey).Result()

	if err == nil {
		json.Unmarshal([]byte(val), &reactions)
	} else {
		reactions.MessageID = messageId
		reactions.Reactions = make(map[string][]model.ReactionUser)
	}

	r, rf := reactions.Reactions[input.Reaction]

	if rf {
		adding := true
		idx := 0

		for i, ru := range r {
			if ru.UserID == user.ID {
				adding = false
				idx = i
			}
		}

		if adding {
			r = append(r, model.ReactionUser{
				UserID:     user.ID,
				UserHandle: user.Handle.String,
			})
		} else {
			r = model.RemoveReactionUser(r, idx)
		}
	} else {
		r = []model.ReactionUser{{
			UserID:     user.ID,
			UserHandle: user.Handle.String,
		}}
	}

	reactions.MessageID = messageId

	if len(r) > 0 {
		reactions.Reactions[input.Reaction] = r
	} else {
		delete(reactions.Reactions, input.Reaction)
	}

	p, err := json.Marshal(reactions)

	if err != nil {
		slog.Error("Unable to react to message 💀",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to react to message.",
			}},
		})
	}

	go func() {
		_, err = wRdb.Set(ctx, rkey, p, 0).Result()

		if err != nil {
			slog.Error("Unable to react to message 💀",
				slog.String("error", err.Error()))
		}
	}()

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"updated": true,
	})
}
