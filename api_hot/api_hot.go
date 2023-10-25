package main

import (
	"context"
	"fmt"
	"time"

	"log/slog"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/idempotency"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/handlers"
	"github.com/macwilko/exotic-auth/internal_handlers"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	jwtware "github.com/gofiber/contrib/jwt"
)

func main() {
	lg := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(lg)

	slog.Info("🚀 Starting hot api ✅")

	if len(os.Getenv("PORT")) > 0 {
		time.Sleep(4 * time.Second)
	}

	ctx := context.Background()

	godotenv.Load("../.env")

	readRedisOpts, err := redis.ParseURL(os.Getenv("READ_REDIS_URL"))

	if err != nil {
		slog.Error("Unable to read redis database",
			slog.String("error", err.Error()))

		panic(err)
	}

	writeRedisOpts, err := redis.ParseURL(os.Getenv("WRITE_REDIS_URL"))

	if err != nil {
		slog.Error("Unable to read redis database",
			slog.String("error", err.Error()))

		panic(err)
	}

	queue := asynq.NewClient(asynq.RedisClientOpt{
		Network:  writeRedisOpts.Network,
		Addr:     writeRedisOpts.Addr,
		Username: writeRedisOpts.Username,
		Password: writeRedisOpts.Password,
		DB:       writeRedisOpts.DB,
	})

	defer queue.Close()

	db, err := sqlx.Connect("mysql", os.Getenv("DATABASE_URL"))

	if err != nil {
		slog.Error("Unable to connect to db",
			slog.String("error", err.Error()))

		panic(err)
	}

	defer db.Close()

	rRdb := redis.NewClient(&redis.Options{
		Addr:     readRedisOpts.Addr,
		Username: readRedisOpts.Username,
		Password: readRedisOpts.Password,
		DB:       readRedisOpts.DB,
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			slog.Info("Read Connected")
			return nil
		},
	})

	if err := rRdb.Ping(context.Background()).Err(); err != nil {
		slog.Error("Read Redis Error",
			slog.String("error", err.Error()))
	}

	wRdb := redis.NewClient(&redis.Options{
		Addr:     writeRedisOpts.Addr,
		Username: writeRedisOpts.Username,
		Password: writeRedisOpts.Password,
		DB:       writeRedisOpts.DB,
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			slog.Info("Write Connected")
			return nil
		},
	})

	if err := wRdb.Ping(context.Background()).Err(); err != nil {
		slog.Error("Write Redis Error",
			slog.String("error", err.Error()))
	}

	app := fiber.New(fiber.Config{
		Network:   "tcp",
		BodyLimit: 209715200,
	})

	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	app.Use(logger.New())
	// app.Use(helmet.New())
	app.Use(idempotency.New())
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		DisableColors: false,
		Format:        "${pid} ${locals:requestid} ${status} - ${method} ${path}\u200b",
	}))
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	app.Use(cors.New())

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString(fmt.Sprintf("So exotic! %s", os.Getenv("RAILWAY_REPLICA_ID")))
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("I'm healthy!")
	})

	app.Get("/metrics", monitor.New(monitor.Config{Title: "Metrics"}))

	v1 := fiber.New()
	app.Mount("/v1", v1)

	v1.Use(func(c *fiber.Ctx) error {
		c.Accepts("application/json")
		return c.Next()
	})

	internal := fiber.New()

	v1.Mount("/internal", internal)

	internal.Get("/sitemap", func(c *fiber.Ctx) error {
		return internal_handlers.Sitemap(c, ctx, db, wRdb, rRdb, queue)
	})

	auth := fiber.New()

	auth.Use(limiter.New(limiter.Config{
		Max:               30,
		Expiration:        1 * time.Hour,
		LimiterMiddleware: limiter.SlidingWindow{},
	}))

	v1.Mount("/auth", auth)

	auth.Post("/sign_in", func(c *fiber.Ctx) error {
		return handlers.SignIn(c, ctx, db, wRdb, rRdb, queue)
	})

	auth.Post("/sign_up", func(c *fiber.Ctx) error {
		return handlers.SignUp(c, ctx, db, wRdb, rRdb, queue)
	})

	auth.Post("/join_beta", func(c *fiber.Ctx) error {
		return handlers.JoinBeta(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Use(jwtware.New(jwtware.Config{
		SuccessHandler: func(c *fiber.Ctx) error {
			lg.Info("jwt authorized ✅")
			return c.Next()
		},
		ErrorHandler: func(c *fiber.Ctx, h error) error {
			lg.Info("jwt unauthorized 👀")
			return c.Next()
		},
		SigningKey: jwtware.SigningKey{Key: []byte(os.Getenv("JWT_SECRET"))},
	}))

	v1.Use(func(c *fiber.Ctx) error {
		return handlers.AuthorizationREST(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Get("/community/:handle", func(c *fiber.Ctx) error {
		return handlers.Community(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Get("/users/:id", func(c *fiber.Ctx) error {
		return handlers.GetUser(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Get("/communities/:communityHandle/users", func(c *fiber.Ctx) error {
		return handlers.CommunityUsers(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Get("/communities/:communityHandle/channels/:channelHandle", func(c *fiber.Ctx) error {
		return handlers.Channel(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Get("/communities/:handle", func(c *fiber.Ctx) error {
		return handlers.Community(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Get("/invite", func(c *fiber.Ctx) error {
		return handlers.CommunityInvite(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Use(func(c *fiber.Ctx) error {
		_, ok := c.Locals("viewer").(model.Users)

		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not allowed.",
				}},
			})
		}

		return c.Next()
	})

	v1.Post("/communities/create", func(c *fiber.Ctx) error {
		return handlers.CreateCommunity(c, ctx, db, rRdb, queue)
	})

	v1.Post("/communities/:handle/delete", func(c *fiber.Ctx) error {
		return handlers.DeleteCommunity(c, ctx, db, rRdb, queue)
	})

	v1.Get("/me", func(c *fiber.Ctx) error {
		return handlers.Me(c, ctx, db, rRdb, queue)
	})

	v1.Post("/me/update", func(c *fiber.Ctx) error {
		return handlers.UpdateProfile(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/refresh_token", func(c *fiber.Ctx) error {
		return handlers.RefreshToken(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/me/edit-profile-picture", func(c *fiber.Ctx) error {
		return handlers.EditProfilePicture(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/edit", func(c *fiber.Ctx) error {
		return handlers.EditCommunity(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Get("/communities/:handle/roles", func(c *fiber.Ctx) error {
		return handlers.CommunityRoles(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Get("/communities/:handle/default-permissions", func(c *fiber.Ctx) error {
		return handlers.CommunityDefaultPermissions(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/kick-user", func(c *fiber.Ctx) error {
		return handlers.KickUser(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/ban-user", func(c *fiber.Ctx) error {
		return handlers.BanUser(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/invites/create", func(c *fiber.Ctx) error {
		return handlers.CreateCommunityInvite(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/leave", func(c *fiber.Ctx) error {
		return handlers.LeaveCommunity(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/join", func(c *fiber.Ctx) error {
		return handlers.JoinCommunity(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/default-permissions/edit", func(c *fiber.Ctx) error {
		return handlers.EditCommunityDefaultPermissions(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Get("/communities/:handle/roles/:roleId", func(c *fiber.Ctx) error {
		return handlers.CommunityRole(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/roles/edit", func(c *fiber.Ctx) error {
		return handlers.EditCommunityRole(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/roles/edit-priority", func(c *fiber.Ctx) error {
		return handlers.EditCommunityRolesPriority(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/roles/delete", func(c *fiber.Ctx) error {
		return handlers.DeleteCommunityRole(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/roles/create", func(c *fiber.Ctx) error {
		return handlers.CreateCommunityRole(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/channels/edit", func(c *fiber.Ctx) error {
		return handlers.EditChannel(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/channels/select", func(c *fiber.Ctx) error {
		return handlers.ChannelSelect(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/channels/create", func(c *fiber.Ctx) error {
		return handlers.CreateChannel(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/channels/delete", func(c *fiber.Ctx) error {
		return handlers.DeleteChannel(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/groups/create", func(c *fiber.Ctx) error {
		return handlers.CreateGroup(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/groups/edit", func(c *fiber.Ctx) error {
		return handlers.EditGroup(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/groups/delete", func(c *fiber.Ctx) error {
		return handlers.DeleteGroup(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/messages/create", func(c *fiber.Ctx) error {
		return handlers.CreateMessage(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/messages/edit", func(c *fiber.Ctx) error {
		return handlers.EditMessage(c, ctx, db, wRdb, rRdb, queue)
	})

	v1.Post("/communities/:handle/messages/react", func(c *fiber.Ctx) error {
		return handlers.ReactToMessage(c, ctx, db, wRdb, rRdb, queue)
	})

	port := ":3001"

	if envPort := os.Getenv("PORT"); envPort != "" {
		port = ":" + envPort
	}

	app.Listen(port)
}
