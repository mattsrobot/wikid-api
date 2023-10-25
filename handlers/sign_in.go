package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/mail"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/jmoiron/sqlx"
	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/security_helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type SignInInput struct {
	Email    string  `json:"email" validate:"required,email,lte=200"`
	Password string  `json:"password" validate:"required,gte=6,lte=50"`
	Code     *string `json:"code" validate:"omitempty,lte=30"`
}

func SignIn(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	slog.Info("Starting user sign_in ✅")

	input := new(SignInInput)

	if err := c.BodyParser(input); err != nil {
		slog.Error("💀 Invalid input 💀")
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to login",
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

	en := en.New()
	uni := ut.New(en, en)
	trans, _ := uni.GetTranslator("en")
	en_translations.RegisterDefaultTranslations(validate, trans)
	err := validate.Struct(input)

	var errors []fiber.Map

	if err != nil {
		slog.Warn("💀 Invalid input 💀")
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
		slog.Error("💀 Unable to sign_in 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": errors,
		})
	}

	addr, err := mail.ParseAddress(input.Email)

	if err != nil {
		slog.Error("💀 Email not valid 💀")
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "email",
				"message": "Not a not valid email",
			}},
		})
	}

	email := strings.ToLower(addr.Address)

	var userCount int
	err = db.Get(&userCount, "SELECT count(*) FROM users WHERE email = ?", email)

	if err != nil {
		slog.Error("💀 Unable to sign_in 💀")
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to login",
			}},
		})
	}

	newUser := (userCount == 0)

	if newUser {
		slog.Warn("💀 User does not exist 💀 ")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "User does not exist",
			}},
		})
	}

	var user model.Users

	err = db.Get(&user, "SELECT * FROM users WHERE email = ? LIMIT 1", email)

	if err != nil {
		slog.Error("Unable to sign_in")
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to login.",
			}},
		})
	}

	if user.PasswordHash == "" {
		slog.Error("💀 Unable to sign_in 💀 ")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "User does not exist.",
			}},
		})
	}

	password_checks_out := security_helpers.CheckPasswordHash(input.Password, user.PasswordHash)

	if !password_checks_out {
		slog.Warn("💀 Unable to sign_in 💀 ")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "password",
				"message": "Password invalid.",
			}},
		})
	}

	p, err := json.Marshal(user)

	if err != nil {
		slog.Error("Unable to login")
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to login.",
			}},
		})
	}

	go func() {
		_, err = wRdb.Set(ctx, fmt.Sprintf("user-%d", user.ID), p, 1*time.Hour).Result()

		if err != nil {
			slog.Error("💀 Unable to login 💀",
				slog.String("error", err.Error()))
		}
	}()

	claims := jwt.MapClaims{
		"id":  security_helpers.Encode(user.ID, model.USERS_TYPE, user.Salt),
		"exp": time.Now().Add((time.Hour * 24) * 31).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	t, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))

	if err != nil {
		slog.Error("💀 Unable to login 💀")
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to login.",
			}},
		})
	}

	if input.Code != nil {

		invite := model.CommunityInvites{}

		err := db.Get(&invite, "SELECT * FROM community_invites WHERE code = ? LIMIT 1", *input.Code)

		if err != nil {
			slog.Error("No invite found 💀",
				slog.String("error", err.Error()),
				slog.String("area", "can't find invite"))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		tx, _ := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

		_, err = tx.Exec("INSERT INTO communities_users (created_at, community_id, user_id, selected_channel_id) VALUES (?, ?, ?, ?)", time.Now(), invite.CommunityID, user.ID, 0)

		if err != nil {
			tx.Rollback()

			slog.Error("💀 Unable to sign_in, db issue 💀",
				slog.String("error", err.Error()))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Unable to sign in currently.",
				}},
			})
		}

		community := model.Communities{}

		err = db.Get(&community, "SELECT * FROM communities WHERE id = ? LIMIT 1", invite.CommunityID)

		if err != nil {
			tx.Rollback()

			slog.Error("💀 Unable to sign_in, db issue 💀",
				slog.String("error", err.Error()))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Unable to sign in currently.",
				}},
			})
		}

		var banned uint64

		err = db.Get(&banned, "SELECT count(*) FROM communities_banned_users WHERE user_id = ? AND community_id", user.ID, community.ID)

		if err != nil {
			tx.Rollback()

			slog.Error("Database problem 💀",
				slog.String("error", err.Error()),
				slog.String("area", "can't select count of messages"))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		if banned > 0 {
			tx.Rollback()

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "You are banned from this community.",
				}},
			})
		}

		RecalculateAndUpdatePermissionsForUser(user.ID, community, tx, wRdb, rRdb, ctx)

		err = tx.Commit()

		if err != nil {
			slog.Error("💀 Unable to sign_in, db issue 💀",
				slog.String("error", err.Error()))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Unable to sign in currently.",
				}},
			})
		}
	}

	slog.Info("Issued login token ✅")

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"token": t,
	})
}
