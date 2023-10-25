package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/google/uuid"
	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/security_helpers"
	"golang.org/x/exp/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func EditProfilePicture(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	form, err := c.MultipartForm()

	if err != nil {
		slog.Error("Couldn't validate multipart form",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Invalid input.",
			}},
		})
	}

	files := form.File["documents"]

	if len(files) == 0 {
		slog.Error("Files length was zero")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "No files found.",
			}},
		})
	}

	file := files[0]

	if len(file.Header["Content-Type"]) == 0 {
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "No files found.",
			}},
		})
	}

	contentType := file.Header["Content-Type"][0]
	allowedType := (contentType == "image/png" || contentType == "image/jpeg")

	if !allowedType {
		slog.Error("Files length was zero")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not an allowed type.",
			}},
		})
	}

	ext := filepath.Ext(file.Filename)

	salt := uuid.New().String() + ext

	createdAt := time.Now()

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	if err != nil {
		slog.Error("Couldn't begin transaction",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Invalid input.",
			}},
		})
	}

	opener, err := file.Open()

	if err != err {
		slog.Error("Couldn't open file",
			slog.String("error", err.Error()))

		tx.Rollback()

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Couldn't upload file.",
			}},
		})
	}

	cf, err := cloudflare.New(os.Getenv("CLOUDFLARE_API_KEY"), os.Getenv("CLOUDFLARE_API_EMAIL"))

	if err != nil {
		slog.Error("Couldn't create cf api, cf error 💀",
			slog.String("error", err.Error()))

		tx.Rollback()

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	img, err := cf.UploadImage(ctx, cloudflare.AccountIdentifier(os.Getenv("CLOUDFLARE_ACCOUNT_IDENTIFIER")), cloudflare.UploadImageParams{
		File:              opener,
		Name:              salt,
		RequireSignedURLs: false,
	})

	opener.Close()

	if err != err {
		slog.Error("Couldn't upload file",
			slog.String("error", err.Error()))

		tx.Rollback()

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Couldn't upload file.",
			}},
		})
	}

	ic := `
	INSERT INTO files
	(created_at, object_salt, file_name, user_id, content_size, cf_images_id)
	VALUES (?, ?, ?, ?, ?, ?)`

	_, err = tx.Exec(ic, createdAt, salt, file.Filename, user.ID, file.Size, img.ID)

	if err != nil {
		slog.Error("Couldn't insert files, db error 💀",
			slog.String("error", err.Error()))

		tx.Rollback()

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Invalid input.",
			}},
		})
	}

	var avatarFileId uint64

	err = tx.Get(&avatarFileId, "SELECT LAST_INSERT_ID()")

	if err != nil {
		slog.Error("Couldn't get avtar file id, db error 💀",
			slog.String("error", err.Error()))

		tx.Rollback()

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Invalid input.",
			}},
		})
	}

	_, err = tx.Exec("UPDATE users SET avatar_file_id = ?, cf_avatar_images_id = ? WHERE id = ?", avatarFileId, img.ID, user.ID)

	if err != nil {
		slog.Error("Couldn't update user, db error 💀",
			slog.String("error", err.Error()))

		tx.Rollback()

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Invalid input.",
			}},
		})
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Couldn't commit file",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Invalid input.",
			}},
		})
	}

	wRdb.Del(ctx, fmt.Sprintf("user-%d", user.ID))

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"id":         security_helpers.Encode(user.ID, model.USERS_TYPE, user.Salt),
		"created_at": user.CreatedAt.Format(time.RFC3339),
		"name":       user.Name.String,
		"handle":     user.Handle.String,
		"email":      user.Email,
		"user_count": user.ID,
		"about":      user.About.String,
		"avatar_url": os.Getenv("CLOUDFLARE_IMAGES_PROXY") + user.CFAvatarImagesID.String + "/public",
	})
}
