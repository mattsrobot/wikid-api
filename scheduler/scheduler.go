package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/macwilko/exotic-auth/tasks"
	"github.com/redis/go-redis/v9"

	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
)

func main() {
	lg := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(lg)

	slog.Info("🚀 Starting scheduler ✅")

	if len(os.Getenv("PORT")) > 0 {
		time.Sleep(4 * time.Second)
	}

	godotenv.Load("../.env")

	db, err := sqlx.Connect("mysql", os.Getenv("DATABASE_URL"))

	if err != nil {
		slog.Error("Unable to connect to db",
			slog.String("error", err.Error()))

		panic(err)
	}

	defer db.Close()

	writeRedisOpts, err := redis.ParseURL(os.Getenv("WRITE_REDIS_URL"))

	if err != nil {
		slog.Error("Unable to read redis database",
			slog.String("error", err.Error()))

		panic(err)
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Network:  writeRedisOpts.Network,
			Addr:     writeRedisOpts.Addr,
			Username: writeRedisOpts.Username,
			Password: writeRedisOpts.Password,
			DB:       writeRedisOpts.DB,
		},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)

	mux := asynq.NewServeMux()

	mux.HandleFunc(tasks.TypeEmailDelivery, tasks.HandleEmailDeliveryTask)

	mux.HandleFunc(tasks.TypeCreateCommunity, func(ctx context.Context, t *asynq.Task) error {
		return tasks.HandleCreateCommunityTask(ctx, t, db)
	})

	mux.HandleFunc(tasks.TypeDeleteCommunity, func(ctx context.Context, t *asynq.Task) error {
		return tasks.HandleDeleteCommunityTask(ctx, t, db)
	})

	if err := srv.Run(mux); err != nil {
		slog.Error("Scheduler crashed",
			slog.String("error", err.Error()))
	}
}
