package handlers

import (
	"context"
	"database/sql"
	"strings"

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

type RailwayServiceInput struct {
	ID string `json:"id" validate:"required,gte=3"`
}

type RailwayEnvironmentInput struct {
	Name string `json:"name" validate:"required,gte=3"`
}

type RailwayServiceWebhookInput struct {
	Type        string                  `json:"type" validate:"required"`
	Status      string                  `json:"status" validate:"required"`
	Service     RailwayServiceInput     `json:"service" validate:"required"`
	Environment RailwayEnvironmentInput `json:"environment" validate:"required"`
}

func RailwayServiceWebhook(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, rdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Got webook data from Railway ✅")

	input := new(RailwayServiceWebhookInput)

	if err := c.BodyParser(input); err != nil {
		slog.Info("Didn't recognise this railway webhook, aborting ✅")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
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
		slog.Info(err.Error())

		errs := err.(validator.ValidationErrors)

		for _, v := range errs {
			errors = append(errors, fiber.Map{
				"field":   v.Field(),
				"message": v.Translate(trans),
			})
		}
	}

	if len(errors) > 0 {
		slog.Info("Unable to validate webhook, input 💀")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": errors,
		})
	}

	rwServiceId := input.Service.ID
	rwStatus := strings.ToLower(input.Status)
	rwDsUpdate := len(rwServiceId) > 0 && len(rwStatus) > 0 && rwStatus != "removing"

	if !rwDsUpdate {
		slog.Info("Ignoring rw status changes ✅")
		slog.Info("Done handling webook data from Railway ✅")
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{})
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	handleTxError := func(err error) error {
		tx.Rollback()

		if err != nil {
			slog.Error(err.Error())
		}

		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to handle webhook.",
			}},
		})
	}

	if err != nil {
		slog.Error("Couldn't create tx, db error 💀")

		return handleTxError(err)
	}

	if rwStatus == "success" {
		_, err = tx.Exec("UPDATE communities SET railway_deploy_status = ?, ready = ? WHERE railway_service_id = ?", rwStatus, true, rwServiceId)
	} else {
		_, err = tx.Exec("UPDATE communities SET railway_deploy_status = ? WHERE railway_service_id = ?", rwStatus, rwServiceId)

	}

	if err != nil {
		slog.Error("Couldn't update communitity deploy status, db error 💀")

		return handleTxError(err)
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Couldn't commit to db, db error 💀")
		slog.Error(err.Error())

		return c.Status(fiber.StatusInternalServerError).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to handle webhook.",
			}},
		})
	}

	slog.Info("Done handling webook data from Railway ✅")

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{})
}
