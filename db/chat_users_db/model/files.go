package model

import (
	"database/sql"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/macwilko/exotic-auth/security_helpers"
)

type Files struct {
	ID                     uint64         `db:"id"`
	CreatedAt              time.Time      `db:"created_at"`
	Salt                   string         `db:"object_salt"`
	FileName               string         `db:"file_name"`
	UserID                 uint64         `db:"user_id"`
	ContentSize            uint64         `db:"content_size"`
	MessageID              sql.NullInt64  `db:"message_id"`
	MimeType               sql.NullString `db:"mime_type"`
	CFImagesID             sql.NullString `db:"cf_images_id"`
	CFVideoStreamUID       sql.NullString `db:"cf_video_stream_uid"`
	CFVideoStreamThumbnail sql.NullString `db:"cf_video_stream_thumbnail"`
	CFF2ID                 sql.NullString `db:"cf_r2_uid"`
}

func (c Files) ToFiberMap() fiber.Map {

	if c.CFImagesID.Valid {
		return fiber.Map{
			"id":            security_helpers.Encode(c.ID, FILES_TYPE, c.Salt),
			"created_at":    c.CreatedAt.Format(time.RFC3339),
			"name":          c.FileName,
			"content_size":  c.ContentSize,
			"mime_type":     c.MimeType.String,
			"url":           os.Getenv("CLOUDFLARE_IMAGES_PROXY") + c.CFImagesID.String + "/public",
			"thumbnail_url": os.Getenv("CLOUDFLARE_IMAGES_PROXY") + c.CFImagesID.String + "/public",
		}
	}

	if c.CFVideoStreamUID.Valid {
		return fiber.Map{
			"id":            security_helpers.Encode(c.ID, FILES_TYPE, c.Salt),
			"created_at":    c.CreatedAt.Format(time.RFC3339),
			"name":          c.FileName,
			"content_size":  c.ContentSize,
			"mime_type":     c.MimeType.String,
			"dash_url":      os.Getenv("CLOUDFLARE_VIDEOS_PROXY") + c.CFVideoStreamUID.String + "/manifest/video.mpd",
			"hls_url":       os.Getenv("CLOUDFLARE_VIDEOS_PROXY") + c.CFVideoStreamUID.String + "/manifest/video.m3u8",
			"thumbnail_url": c.CFVideoStreamThumbnail.String,
		}
	}

	return fiber.Map{
		"id":           security_helpers.Encode(c.ID, FILES_TYPE, c.Salt),
		"created_at":   c.CreatedAt.Format(time.RFC3339),
		"name":         c.FileName,
		"content_size": c.ContentSize,
		"mime_type":    c.MimeType.String,
		"url":          os.Getenv("CLOUDFLARE_FILES_PROXY") + c.CFF2ID.String,
	}
}

var FILES_TYPE = "Files"
