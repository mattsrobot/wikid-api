package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/mail"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/security_helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type SignUpInput struct {
	Email    string  `json:"email" validate:"required,email,lte=255"`
	Password string  `json:"password" validate:"required,gte=6,lte=50"`
	Handle   string  `json:"handle" validate:"required,gte=3,lte=30"`
	Code     *string `json:"code" validate:"omitempty,lte=30"`
}

func SignUp(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	slog.Info("Starting user sign_up ✅")

	input := new(SignUpInput)

	if err := c.BodyParser(input); err != nil {
		slog.Error("💀 Invalid input 💀",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "email",
				"message": "Unable to sign up currently.",
			}},
		})
	}

	validate := validator.New()

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
		slog.Error("💀 Unable to sign up 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": errors,
		})
	}

	lowerHandle := strings.ToLower(input.Handle)

	var handleCount int
	err = db.Get(&handleCount, "SELECT count(*) FROM users WHERE handle = ?", lowerHandle)

	if err != nil {
		slog.Error("💀 Unable to sign_up, db issue 💀",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "email",
				"message": "Unable to sign up currently.",
			}},
		})
	}

	if handleCount > 0 {
		slog.Error("💀 Unable to signup, handle was already taken 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "handle",
				"message": "Already taken.",
			}},
		})
	}

	match, _ := regexp.MatchString("[a-z0-9-]+", lowerHandle)

	if !match {
		slog.Error("Lower handle didn't match regex, db error 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "handle",
				"message": "Handle must be must be letters, numbers and _ only.",
			}},
		})
	}

	addr, err := mail.ParseAddress(input.Email)

	if err != nil {
		slog.Error("💀 Unable to sign_up, address issue 💀",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "email",
				"message": "Not a not valid email.",
			}},
		})
	}

	lowerEmail := strings.ToLower(addr.Address)

	var userCount int
	err = db.Get(&userCount, "SELECT count(*) FROM users WHERE email = ?", lowerEmail)

	if err != nil {
		slog.Error("💀 Unable to sign_up, db issue 💀",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "email",
				"message": "Unable to signup currently.",
			}},
		})
	}

	newUser := (userCount == 0)

	if !newUser {
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "email",
				"message": "Already signed up.",
			}},
		})
	}

	validInviteCode := false
	invite := model.CommunityInvites{}

	if input.Code != nil {

		err := db.Get(&invite, "SELECT * FROM community_invites WHERE code = ? LIMIT 1", *input.Code)

		if err != nil {
			slog.Error("No invite found 💀",
				slog.String("error", err.Error()),
				slog.String("area", "can't find invite"))

			return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		validInviteCode = true
	}

	if !validInviteCode {
		var betaNotEnabledCount int

		err = db.Get(&betaNotEnabledCount, "SELECT count(*) FROM beta_users WHERE enabled = ? AND email = ? ", false, lowerEmail)

		if err != nil {
			slog.Error("💀 Unable to sign_up, db issue 💀",
				slog.String("error", err.Error()))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"field":   "email",
					"message": "Unable to signup currently.",
				}},
			})
		}

		if betaNotEnabledCount > 0 {
			slog.Error("💀 Beta not enabled 💀")

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"field":   "email",
					"message": "Hang tight, you are part of the waitlist, but your invitation has not been approved yet.",
				}},
			})
		}
	}

	var betaCount int

	err = db.Get(&betaCount, "SELECT count(*) FROM beta_users WHERE redeemed = ? AND enabled = ? AND email = ? ", false, true, lowerEmail)

	if err != nil {
		slog.Error("💀 Unable to sign_up, db issue 💀",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "email",
				"message": "Unable to signup currently.",
			}},
		})
	}

	canBetaTest := betaCount == 1 || validInviteCode

	if !canBetaTest {
		slog.Warn("💀 User tried to sign_up but was not part of beta test 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "email",
				"message": "You are not part of the beta test. Reach out to the team to join.",
			}},
		})
	}

	passwordHash, err := security_helpers.HashPassword(input.Password)

	if err != nil {
		slog.Error("💀 Unable to sign_up, db issue 💀",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"field":   "password",
				"message": "Unable to signup currently.",
			}},
		})
	}

	var user model.Users

	tx, _ := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	_, err = tx.Exec("UPDATE beta_users SET redeemed = ? WHERE email = ?", true, lowerEmail)

	if err != nil {
		tx.Rollback()

		slog.Error("💀 Unable to sign_up, db issue 💀",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to signup currently.",
			}},
		})
	}

	_, err = tx.Exec("INSERT INTO users (created_at, email, handle, object_salt, password_hash) VALUES (?, ?, ?, ?, ?)", time.Now(), lowerEmail, lowerHandle, uuid.New().String(), passwordHash)

	if err != nil {
		tx.Rollback()

		slog.Error("💀 Unable to sign_up, db issue 💀",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to signup currently.",
			}},
		})
	}

	err = tx.Get(&user, "SELECT * FROM users WHERE email = ? LIMIT 1", lowerEmail)

	if err != nil {
		tx.Rollback()

		slog.Error("💀 Unable to sign_up, db issue 💀",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to signup currently.",
			}},
		})
	}

	if validInviteCode {
		_, err = tx.Exec("INSERT INTO communities_users (created_at, community_id, user_id, selected_channel_id) VALUES (?, ?, ?, ?)", time.Now(), invite.CommunityID, user.ID, 0)

		if err != nil {
			tx.Rollback()

			slog.Error("💀 Unable to sign_up, db issue 💀",
				slog.String("error", err.Error()))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Unable to signup currently.",
				}},
			})
		}

		community := model.Communities{}

		err := db.Get(&community, "SELECT * FROM communities WHERE id = ? LIMIT 1", invite.CommunityID)

		if err != nil {
			tx.Rollback()

			slog.Error("💀 Unable to sign_up, db issue 💀",
				slog.String("error", err.Error()))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Unable to signup currently.",
				}},
			})
		}

		RecalculateAndUpdatePermissionsForUser(user.ID, community, tx, wRdb, rRdb, ctx)
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("💀 Unable to sign_up, db issue 💀",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to signup currently.",
			}},
		})
	}

	p, err := json.Marshal(user)

	if err != nil {
		slog.Error("💀 Unable to sign_up, json marshal issue 💀",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to signup.",
			}},
		})
	}

	go func() {
		_, err = wRdb.Set(ctx, fmt.Sprintf("user-%d", user.ID), p, 1*time.Hour).Result()

		if err != nil {
			slog.Error("💀 Unable to sign_up, redis issue 💀",
				slog.String("error", err.Error()))
		}
	}()

	slog.Info(fmt.Sprintf("Issuing claims for user id %d", user.ID))

	claims := jwt.MapClaims{
		"id":  security_helpers.Encode(user.ID, model.USERS_TYPE, user.Salt),
		"exp": time.Now().Add((time.Hour * 24) * 31).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	t, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))

	if err != nil {
		slog.Error("💀 Unable to sign_up 💀")
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to signup.",
			}},
		})
	}

	slog.Info("Issued sign_up token ✅")

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"token": t,
	})
}
