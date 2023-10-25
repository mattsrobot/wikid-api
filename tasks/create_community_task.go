package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/cloudflare/cloudflare-go"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/railway"
	"github.com/macwilko/exotic-auth/security_helpers"
)

const (
	TypeCreateCommunity = "community:create"
)

type CreateCommunityPayload struct {
	CommunityID uint64
}

func NewCreateCommunityTask(communityId uint64) (*asynq.Task, error) {
	payload, err := json.Marshal(CreateCommunityPayload{CommunityID: communityId})

	slog.Info("Scheduling community for creation")

	if err != nil {
		slog.Error("Unable to schedule community creation")
		slog.Error(err.Error())

		return nil, err
	}

	return asynq.NewTask(TypeCreateCommunity, payload), nil
}

func HandleCreateCommunityTask(ctx context.Context, t *asynq.Task, db *sqlx.DB) error {
	slog.Info("Creating community ✅")

	var p CreateCommunityPayload

	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		slog.Error("Could not create community")
		slog.Error(err.Error())

		return fmt.Errorf("json.Unmarshal failed: %v: %w", err, asynq.SkipRetry)
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	if err != nil {
		slog.Error("Couldn't tx, db error 💀")

		return fmt.Errorf("TX error: %v: %w", err, asynq.SkipRetry)
	}

	handleTxError := func(err error) error {
		tx.Rollback()

		slog.Error("Unable to create community. 💀")

		if err != nil {
			slog.Error(err.Error())
		}

		return fmt.Errorf("TX error: %v: %w", err, asynq.SkipRetry)
	}

	var community model.Communities

	err = tx.Get(&community, "SELECT * FROM communities WHERE id = ? LIMIT 1", p.CommunityID)

	if err != nil {
		slog.Error("Couldn't create cf api, cf error 💀")

		return handleTxError(err)
	}

	cf, err := cloudflare.New(os.Getenv("CLOUDFLARE_API_KEY"), os.Getenv("CLOUDFLARE_API_EMAIL"))

	if err != nil {
		slog.Error("Couldn't create cf api, cf error 💀")

		return handleTxError(err)
	}

	var (
		railwayToken  = os.Getenv("RAILWAY_TOKEN")
		projectId     = os.Getenv("RAILWAY_PROJECT_ID")
		environmentId = os.Getenv("RAILWAY_ENVIRONMENT_ID")
	)

	rw := railway.NewAuthedClient(railwayToken)

	cleanupAndReturn := func(err error, rollback bool, removeRecordId *string, removeServiceId *string) error {
		if rollback {
			slog.Error("Rollback")

			tx.Rollback()
		}

		if removeRecordId != nil {
			slog.Error("Removing records")

			cf.DeleteDNSRecord(ctx, nil, *removeRecordId)
		}

		if removeServiceId != nil {
			slog.Error("Removing railway service")

			railway.ServiceDelete(rw, *removeServiceId, &environmentId)
		}

		return handleTxError(err)
	}

	sc, _, err := railway.ServiceCreate(rw, &railway.ServiceCreateInput{
		Branch:        nil,
		Name:          &community.Handle,
		EnvironmentId: &environmentId,
		ProjectId:     projectId,
		Source:        nil,
		Variables:     nil,
	})

	if err != nil {
		slog.Error("Couldn't create railway service")

		return cleanupAndReturn(err, true, nil, nil)
	}

	rwServiceId := sc.ServiceCreate.Id

	cd, _, err := railway.CustomDomainCreate(rw, &railway.CustomDomainCreateInput{Domain: community.FQN, EnvironmentId: environmentId, ServiceId: rwServiceId})

	if err != nil {
		slog.Error("Couldn't create custom domain for railway")

		return cleanupAndReturn(err, true, nil, &sc.ServiceCreate.Id)
	}

	communityID := security_helpers.Encode(community.ID, model.COMMUNITIES_TYPE, community.Salt)

	rv := map[string]string{
		"RAILWAY_DOCKERFILE_PATH": "Dockerfile.hot",
		"COMMUNITY_ID":            communityID,
		"FQN":                     community.FQN,
		"HANDLE":                  community.Handle,
		"DATABASE_URL":            os.Getenv("DATABASE_URL"),
		"WRITE_REDIS_URL":         os.Getenv("WRITE_REDIS_URL"),
		"AES_IV":                  os.Getenv("AES_IV"),
		"AES_KEY":                 os.Getenv("AES_KEY"),
		"JWT_SECRET":              os.Getenv("JWT_SECRET"),
		"SALT":                    os.Getenv("SALT"),
		"PORT":                    "3000",
	}

	_, _, err = railway.VariableCollectionUpsert(rw, &railway.VariableCollectionUpsertInput{EnvironmentId: environmentId, ServiceId: &rwServiceId, Variables: rv, ProjectId: projectId})

	if err != nil {
		slog.Error("Couldn't upsert railway variables")

		return cleanupAndReturn(err, true, nil, &sc.ServiceCreate.Id)
	}

	rwr := cd.CustomDomainCreate.Status.DnsRecords

	if len(rwr) == 0 {
		slog.Error("Couldn't find railway DNS records to update")

		return cleanupAndReturn(err, true, nil, &sc.ServiceCreate.Id)
	}

	rwAddress := ""

	for _, v := range rwr {
		if v.RecordType == "DNS_RECORD_TYPE_CNAME" {
			rwAddress = v.RequiredValue
			break
		}
	}

	if len(rwAddress) == 0 {
		slog.Error("Couldn't find railway DNS records to update")

		return cleanupAndReturn(err, true, nil, &sc.ServiceCreate.Id)
	}

	proxied := true

	zr, err := cf.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(os.Getenv("CLOUDFLARE_ZONE_ID")), cloudflare.CreateDNSRecordParams{
		Name:    community.Handle,
		Content: rwAddress,
		Type:    "cname",
		Proxied: &proxied,
	})

	if err != nil {
		slog.Error("Couldn't create zone record")

		return cleanupAndReturn(err, true, nil, &sc.ServiceCreate.Id)
	}

	_, err = tx.Exec("UPDATE communities SET cf_fqn_zone_id = ?, railway_service_id = ? WHERE id = ?", zr.ID, rwServiceId, community.ID)

	if err != nil {
		slog.Error("Couldn't update database for railway records")

		return cleanupAndReturn(err, true, nil, &sc.ServiceCreate.Id)
	}

	healthCheckPath := "/health"

	_, _, err = railway.ServiceInstanceUpdate(rw, rwServiceId, &environmentId, &railway.ServiceInstanceUpdateInput{HealthcheckPath: &healthCheckPath})

	if err != nil {
		slog.Error("Couldn't update railway service instance records")

		return cleanupAndReturn(err, true, nil, &sc.ServiceCreate.Id)
	}

	prw := railway.NewAuthedClient(os.Getenv("RAILWAY_PERSONAL_TOKEN"))

	repo := "mattsrobot/exotic-auth"
	branch := "main"

	_, _, err = railway.ServiceConnect(prw, rwServiceId, &railway.ServiceConnectInput{Repo: &repo, Branch: &branch})

	if err != nil {
		slog.Error("Couldn't update railway service instance records")

		return cleanupAndReturn(err, true, nil, &sc.ServiceCreate.Id)
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Couldn't commit community")

		return cleanupAndReturn(err, false, &zr.ID, &sc.ServiceCreate.Id)
	}

	return nil
}
