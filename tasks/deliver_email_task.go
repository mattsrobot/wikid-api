package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/hibiken/asynq"
	"github.com/mrz1836/postmark"
)

const (
	TypeEmailDelivery = "email:deliver"
)

type EmailDeliveryPayload struct {
	TemplateId        string
	TemplateVariables map[string]interface{}
	From              string
	To                string
}

func NewEmailDeliveryTask(templateId string, from string, to string, templateVariables map[string]interface{}) (*asynq.Task, error) {
	payload, err := json.Marshal(EmailDeliveryPayload{TemplateId: templateId, TemplateVariables: templateVariables, From: from, To: to})

	slog.Info("Scheduling email for delivery")

	if err != nil {
		slog.Error("Unable to schedule email")
		slog.Error(err.Error())

		return nil, err
	}

	return asynq.NewTask(TypeEmailDelivery, payload), nil
}

func HandleEmailDeliveryTask(ctx context.Context, t *asynq.Task) error {
	slog.Info("Sending email")

	var p EmailDeliveryPayload

	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		slog.Error("Could not send email")
		slog.Error(err.Error())

		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	client := postmark.NewClient(os.Getenv("POSTMARK_SERVER_TOKEN"), os.Getenv("POSTMARK_ACCOUNT_TOKEN"))

	templatedEmail := postmark.TemplatedEmail{
		TemplateAlias: p.TemplateId,
		TemplateModel: p.TemplateVariables,
		From:          p.From,
		To:            p.To,
		TrackOpens:    true,
		TrackLinks:    "HtmlAndText",
	}

	_, err := client.SendTemplatedEmail(context.Background(), templatedEmail)

	if err != nil {
		slog.Error("Could not send email")
		slog.Error(err.Error())

		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	return nil
}
