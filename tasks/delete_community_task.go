package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/cloudflare/cloudflare-go"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/macwilko/exotic-auth/railway"
)

const (
	TypeDeleteCommunity = "community:delete"
)

type DeleteCommunityPayload struct {
	CFRecordID       string
	RailwayServiceID string
}

func NewDeleteCommunityTask(cfRecordID string, railwayServiceID string) (*asynq.Task, error) {
	payload, err := json.Marshal(DeleteCommunityPayload{CFRecordID: cfRecordID, RailwayServiceID: railwayServiceID})

	slog.Info("Scheduling community for deletion")

	if err != nil {
		slog.Error("Unable to schedule community deletion")
		slog.Error(err.Error())

		return nil, err
	}

	return asynq.NewTask(TypeDeleteCommunity, payload), nil
}

func HandleDeleteCommunityTask(ctx context.Context, t *asynq.Task, db *sqlx.DB) error {
	slog.Info("Deleting community ✅")

	var p DeleteCommunityPayload

	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		slog.Error("Could not create community")
		slog.Error(err.Error())

		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	var (
		railwayToken  = os.Getenv("RAILWAY_TOKEN")
		environmentId = os.Getenv("RAILWAY_ENVIRONMENT_ID")
	)

	rw := railway.NewAuthedClient(railwayToken)

	slog.Error("Removing records")

	cf, _ := cloudflare.New(os.Getenv("CLOUDFLARE_API_KEY"), os.Getenv("CLOUDFLARE_API_EMAIL"))

	cf.DeleteDNSRecord(ctx, cloudflare.ZoneIdentifier(os.Getenv("CLOUDFLARE_ZONE_ID")), p.CFRecordID)

	railway.ServiceDelete(rw, p.RailwayServiceID, &environmentId)

	return nil
}
