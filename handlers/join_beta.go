package handlers

import (
	"context"
	"database/sql"
	"net/mail"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/jmoiron/sqlx"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type JoinBetaInput struct {
	Email string `json:"email" validate:"required,email"`
}

func JoinBeta(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	slog.Info("Starting beta join ✅")

	input := new(JoinBetaInput)

	if err := c.BodyParser(input); err != nil {
		slog.Error("Invalid input")
		slog.Error(err.Error())

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "email",
				"message": "Unable to join beta currently.",
			}},
		})
	}

	validate := validator.New()
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	err := validate.Struct(input)

	if err != nil {
		slog.Error(err.Error())
	}

	en := en.New()
	uni := ut.New(en, en)
	trans, _ := uni.GetTranslator("en")
	en_translations.RegisterDefaultTranslations(validate, trans)
	err = validate.Struct(input)

	var errors []fiber.Map

	if err != nil {
		slog.Error(err.Error())

		errs := err.(validator.ValidationErrors)

		for _, v := range errs {
			errors = append(errors, fiber.Map{
				"field":   v.Field(),
				"message": v.Translate(trans),
			})
		}
	}

	if len(errors) > 0 {
		slog.Error("💀 Unable to join beta 💀")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": errors,
		})
	}

	addr, err := mail.ParseAddress(input.Email)

	if err != nil {
		slog.Warn("💀 Email not valid 💀")
		slog.Warn(err.Error())

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "email",
				"message": "Not a not valid email.",
			}},
		})
	}

	email := strings.ToLower(addr.Address)

	var userCount int
	err = db.Get(&userCount, "SELECT count(*) FROM beta_users WHERE email = ?", email)

	if err != nil {
		slog.Error("💀 Unable to join beta, db issue 💀")
		slog.Error(err.Error())

		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "email",
				"message": "Unable to join beta currently.",
			}},
		})
	}

	newUser := (userCount == 0)

	if !newUser {
		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "email",
				"message": "Already signed up to beta test.",
			}},
		})
	}

	tx, _ := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	_, err = tx.Exec("INSERT INTO beta_users (created_at, email) VALUES (?, ?)", time.Now(), email)

	if err != nil {
		tx.Rollback()
		slog.Error("💀 Unable to join beta, db issue 💀")
		slog.Error(err.Error())

		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to join beta currently.",
			}},
		})
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("💀 Unable to join beta, db issue 💀")
		slog.Error(err.Error())

		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to join beta currently.",
			}},
		})
	}

	slog.Info("User joined beta ✅")

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"joined": true,
	})
}
