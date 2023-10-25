package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cloudflare/cloudflare-go"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/imroc/req/v3"
	"github.com/jmoiron/sqlx"
	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/internal_handlers"
	"github.com/macwilko/exotic-auth/security_helpers"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type CreateMessageInput struct {
	Text      string  `json:"text" validate:"required,lte=2000"`
	ChannelID string  `json:"channel_id" validate:"required,lte=255"`
	ParentID  *string `json:"parent_id" validate:"omitempty,lte=255"`
}

func tempDir() string {
	dir := os.Getenv("TMPDIR")
	if dir == "" {
		dir = "/tmp"
	}
	return dir
}

func CreateMessage(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {

	slog.Info("Creating message ✅")

	user, ok := c.Locals("viewer").(model.Users)

	if !ok {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	cf, err := cloudflare.New(os.Getenv("CLOUDFLARE_API_KEY"), os.Getenv("CLOUDFLARE_API_EMAIL"))

	if err != nil {
		slog.Error("Couldn't create cf api, cf error 💀")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	form, err := c.MultipartForm()

	if err != nil {
		slog.Error("Couldn't validate multipart form create message",
			slog.String("error", err.Error()))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Invalid input.",
			}},
		})
	}

	if len(form.Value["channel_id"]) == 0 || len(form.Value["text"]) == 0 {
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Invalid input.",
			}},
		})
	}

	input := CreateMessageInput{
		ChannelID: form.Value["channel_id"][0],
		Text:      form.Value["text"][0],
	}

	if len(form.Value["parent_id"]) > 0 {
		input.ParentID = &form.Value["parent_id"][0]

	}

	var errors []fiber.Map

	if len(errors) > 0 {
		slog.Error("Unable to create message, input error 💀")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": errors,
		})
	}

	files := form.File["files"]

	if len(files) > 5 {
		slog.Error("Too many files")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "To many files.",
			}},
		})
	}

	hasFiles := len(files) > 0

	handle := Truncate(strings.ToLower(c.Params("handle")), 255)

	community := model.Communities{}

	err = db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", handle)

	if err != nil {
		slog.Info("No community found 💀 " + handle)
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	hasPermission := HasCommunityPermission(user.ID, community.ID, model.SendMessages, db, wRdb, rRdb, ctx)

	if !hasPermission {
		slog.Warn("Not allowed")

		return c.Status(fiber.StatusUnauthorized).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not allowed.",
			}},
		})
	}

	channel := model.Channels{}

	channelId, channelOk := security_helpers.Decode(input.ChannelID)

	if channelId == 0 || channelOk != model.CHANNELS_TYPE {
		slog.Info("Channel security ID failure 💀 ")

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	err = db.Get(&channel, "SELECT * FROM channels WHERE id = ? AND community_id = ? LIMIT 1", channelId, community.ID)

	if err != nil {
		slog.Info("No group found 💀 " + handle)
		slog.Error(err.Error())

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	salt := uuid.New().String()

	createdAt := time.Now()

	handleCantCreateError := func(err error) error {
		slog.Error("Unable to create message. 💀")

		if err != nil {
			slog.Error(err.Error())
		}

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Unable to create message.",
			}},
		})
	}

	tx, err := db.BeginTxx(ctx, &sql.TxOptions{ReadOnly: false})

	if err != nil {
		slog.Error("Couldn't begin tx, db error 💀")

		return handleCantCreateError(err)
	}

	handleTxError := func(err error) error {
		tx.Rollback()

		return handleCantCreateError(err)
	}

	var parentId uint64 = 0

	if input.ParentID != nil {

		pId, parentOk := security_helpers.Decode(*input.ParentID)

		if pId == 0 || parentOk != model.MESSAGES_TYPE {
			slog.Error("Channel security ID failure 💀 ")

			tx.Rollback()

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		parent := model.Messages{}

		err = tx.Get(&parent, "SELECT * FROM messages WHERE id = ? AND community_id = ? LIMIT 1", pId, community.ID)

		if err != nil {
			slog.Error("Couldn't find parent message, db error 💀")

			return handleTxError(err)
		}

		slog.Info("🔥 Replying to a parent")

		parentId = pId
	}

	message := model.Messages{}

	tx.Get(&message, "SELECT * FROM messages WHERE channel_id = ? ORDER BY id DESC LIMIT 1", channelId)

	var messageId uint64

	var prevCount uint64

	tx.Get(&prevCount, "SELECT count(*) FROM files WHERE message_id = ? AND user_id = ?", message.ID, user.ID)

	if message.UserID == user.ID && parentId == 0 && !hasFiles && prevCount == 0 {

		messageId = message.ID

		updatedAt := time.Now()

		text := message.Text + "\n" + input.Text

		_, err = tx.Exec("UPDATE messages SET text = ?, created_at = ?, updated_at = ? WHERE id = ?", text, updatedAt, updatedAt, messageId)

		if err != nil {
			slog.Error("Couldn't edit messages, db error 💀")

			return handleTxError(err)
		}

	} else {

		_, err = tx.Exec("INSERT INTO messages (created_at, object_salt, community_id, channel_id, user_id, text, parent_id) VALUES (?, ?, ?, ?, ?, ?, ?)", createdAt, salt, community.ID, channel.ID, user.ID, input.Text, parentId)

		if err != nil {
			slog.Error("Couldn't insert messages, db error 💀")

			return handleTxError(err)
		}

		err = tx.Get(&messageId, "SELECT LAST_INSERT_ID()")

		if err != nil {
			slog.Error("Couldn't get last insert for messages, db error 💀")

			return handleTxError(err)
		}
	}

	for _, file := range files {

		ext := filepath.Ext(file.Filename)
		salt := uuid.New().String()
		filename := salt + ext

		createdAt := time.Now()

		if len(file.Header["Content-Type"]) == 0 {
			slog.Error("Files length was zero")

			tx.Rollback()

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not an allowed type.",
				}},
			})
		}

		contentType := file.Header["Content-Type"][0]

		validType := len(contentType) > 0

		if !validType {
			slog.Error("Files length was zero")

			tx.Rollback()

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not an allowed type.",
				}},
			})
		}

		if strings.Contains(contentType, "image") {
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

			img, err := cf.UploadImage(ctx, cloudflare.AccountIdentifier(os.Getenv("CLOUDFLARE_ACCOUNT_IDENTIFIER")), cloudflare.UploadImageParams{
				File:              opener,
				Name:              filename,
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
			(created_at, object_salt, file_name, user_id, content_size, message_id, mime_type, cf_images_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

			_, err = tx.Exec(ic, createdAt, filename, file.Filename, user.ID, file.Size, messageId, contentType, img.ID)

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
		} else if strings.Contains(contentType, "video") {
			tempFile := fmt.Sprintf("%s/%s", tempDir(), filename)

			if err := c.SaveFile(file, tempFile); err != nil {

				slog.Error("Couldn't save file to tmp",
					slog.String("error", err.Error()))

				tx.Rollback()

				return c.Status(fiber.StatusOK).JSON(&fiber.Map{
					"errors": []fiber.Map{{
						"message": "Couldn't upload file.",
					}},
				})
			}

			video, err := cf.StreamUploadVideoFile(ctx, cloudflare.StreamUploadFileParameters{
				AccountID: os.Getenv("CLOUDFLARE_ACCOUNT_IDENTIFIER"),
				FilePath:  tempFile,
			})

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

			err = os.Remove(filename)

			if err != err {
				slog.Error("Couldn't remove temp file",
					slog.String("error", err.Error()))
			}

			ic := `
			INSERT INTO files
			(created_at, object_salt, file_name, user_id, content_size, message_id, mime_type, cf_video_stream_uid, cf_video_stream_thumbnail)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

			_, err = tx.Exec(ic, createdAt, filename, file.Filename, user.ID, file.Size, messageId, contentType, video.UID, video.Thumbnail)

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
		} else {
			bucketName := os.Getenv("CLOUDFLARE_BUCKET_NAME")
			accountId := os.Getenv("CLOUDFLARE_ACCOUNT_IDENTIFIER")
			accessKeyId := os.Getenv("CLOUDFLARE_R2_KEY_ID")
			accessKeySecret := os.Getenv("CLOUDFLARE_R2_ACCESS_SECRET")

			r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountId),
					HostnameImmutable: true,
					Source:            aws.EndpointSourceCustom,
				}, nil
			})

			cfg, err := config.LoadDefaultConfig(ctx,
				config.WithEndpointResolverWithOptions(r2Resolver),
				config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")),
				config.WithRegion("auto"),
			)

			if err != nil {
				slog.Error("Couldn't get S3 context 💀",
					slog.String("error", err.Error()))

				tx.Rollback()

				return c.Status(fiber.StatusOK).JSON(&fiber.Map{
					"errors": []fiber.Map{{
						"message": "Couldn't upload file.",
					}},
				})
			}

			client := s3.NewFromConfig(cfg)

			uploader := manager.NewUploader(client)

			body, err := file.Open()

			if err != nil {
				slog.Error("Couldn't open file 💀",
					slog.String("error", err.Error()))

				tx.Rollback()

				return c.Status(fiber.StatusOK).JSON(&fiber.Map{
					"errors": []fiber.Map{{
						"message": "Couldn't upload file.",
					}},
				})
			}

			result, err := uploader.Upload(ctx, &s3.PutObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(filename),
				Body:   body,
			})

			if err != nil {
				slog.Error("Couldn't upload file 💀",
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
			(created_at, object_salt, file_name, user_id, content_size, message_id, mime_type, cf_r2_uid)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

			_, err = tx.Exec(ic, createdAt, filename, file.Filename, user.ID, file.Size, messageId, contentType, result.Key)

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
		}
	}

	err = tx.Commit()

	if err != nil {
		slog.Error("Couldn't commit channel")

		return handleCantCreateError(err)
	}

	go func() {
		_, err = wRdb.Set(ctx, fmt.Sprintf("user-%d-channel-%d", user.ID, channel.ID), time.Now().Format(time.RFC3339), 0).Result()

		if err != nil {
			slog.Error("Couldn't update communities redis for channel 💀",
				slog.String("error", err.Error()),
				slog.String("area", "not sure"))
		}
	}()

	var avatarUrl *string = nil

	if user.CFAvatarImagesID.Valid {
		s := os.Getenv("CLOUDFLARE_IMAGES_PROXY") + user.CFAvatarImagesID.String + "/public"
		avatarUrl = &s
	}

	newMessage := model.Messages{}

	err = db.Get(&newMessage, "SELECT * FROM messages WHERE id = ? LIMIT 1", messageId)

	if err != nil {
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	rolesUsers := []model.CommuniyRolesUsers{}

	err = db.Select(&rolesUsers, "SELECT * FROM community_roles_users WHERE community_id = ? AND user_id = ? ", community.ID, user.ID)

	if err != nil {
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	var cids = []uint64{}

	for _, ru := range rolesUsers {
		cids = append(cids, ru.CommunityRoleID)
	}

	mappedUser := fiber.Map{
		"name":       user.Name.String,
		"handle":     user.Handle.String,
		"avatar_url": avatarUrl,
	}

	if len(cids) > 0 {

		rolesQuery, rolesArgs, err := sqlx.In("SELECT * FROM community_roles WHERE id IN (?) ORDER BY priority ASC LIMIT 1", cids)

		if err != nil {
			slog.Error("Database problem 💀",
				slog.String("error", err.Error()),
				slog.String("area", "community_roles_users"))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		rolesQuery = db.Rebind(rolesQuery)

		role := model.CommunityRoles{}

		err = db.Get(&role, rolesQuery, rolesArgs...)

		if err != nil {
			slog.Error("Database problem 💀",
				slog.String("error", err.Error()),
				slog.String("area", "after the bind selecting roles"))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		uhr := fiber.Map{
			"name":  role.Name,
			"color": role.Color,
		}

		maps.Copy(mappedUser, fiber.Map{
			"powerful_role": uhr,
		})
	}

	mappedMessage := fiber.Map{
		"id":         security_helpers.Encode(messageId, model.MESSAGES_TYPE, newMessage.Salt),
		"created_at": newMessage.CreatedAt.Format(time.RFC3339),
		"text":       newMessage.Text,
		"edited":     newMessage.Edited,
		"user":       mappedUser,
	}

	if newMessage.UpdatedAt.Valid {
		maps.Copy(mappedMessage, fiber.Map{
			"updated_at": newMessage.UpdatedAt.Time.Format(time.RFC3339),
		})
	}

	mfiles := []model.Files{}

	err = db.Select(&mfiles, "SELECT * FROM files WHERE message_id = ?", messageId)

	if err != nil {
		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	if len(mfiles) > 0 {
		mfs := make([]fiber.Map, len(mfiles))
		for i, f := range mfiles {
			mfs[i] = f.ToFiberMap()
		}
		maps.Copy(mappedMessage, fiber.Map{
			"files": mfs,
		})
	}

	if newMessage.ParentID > 0 {
		pMessage := model.Messages{}

		err = db.Get(&pMessage, "SELECT * FROM messages WHERE id = ? LIMIT 1", newMessage.ParentID)

		if err != nil {
			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		pUser := model.Users{}

		err = db.Get(&pUser, "SELECT * FROM users WHERE id = ? LIMIT 1", pMessage.UserID)

		if err != nil {
			pUser = model.GHOST_USER
		}

		var avatarUrl *string = nil

		if pUser.CFAvatarImagesID.Valid {
			s := os.Getenv("CLOUDFLARE_IMAGES_PROXY") + pUser.CFAvatarImagesID.String + "/public"
			avatarUrl = &s
		}

		maps.Copy(mappedMessage, fiber.Map{
			"parent": fiber.Map{
				"id":         security_helpers.Encode(pMessage.ID, model.MESSAGES_TYPE, pMessage.Salt),
				"created_at": pMessage.CreatedAt.Format(time.RFC3339),
				"text":       pMessage.Text,
				"edited":     pMessage.Edited,
				"user": fiber.Map{
					"name":   pUser.Name.String,
					"handle": pUser.Handle.String,
					// "powerful_role": uhr,
					"avatar_url": avatarUrl,
				},
			},
		})
	}

	go func() {
		client := req.C()

		marshalled, err := json.Marshal(mappedMessage)

		if err != nil {
			slog.Error("💀 Couldn't marshal message",
				slog.String("error", err.Error()))

			return
		}

		client.R().
			SetContentType("application/json").
			SetBody(&internal_handlers.BroadcastMessageInput{
				Topic:   security_helpers.Encode(channelId, model.CHANNELS_TYPE, channel.Salt),
				Message: string(marshalled),
			}).
			Post(os.Getenv("PRIVATE_WS_INTERNAL_API") + "/broadcast-message")

		slog.Info("✅ Broadcasted message event")
	}()

	return c.Status(fiber.StatusOK).JSON(&mappedMessage)
}
