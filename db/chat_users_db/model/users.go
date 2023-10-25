package model

import (
	"database/sql"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/macwilko/exotic-auth/security_helpers"
)

type Users struct {
	ID                        uint64         `db:"id"`
	CreatedAt                 time.Time      `db:"created_at"`
	UpdatedAt                 sql.NullTime   `db:"updated_at"`
	Salt                      string         `db:"object_salt"`
	PasswordHash              string         `db:"password_hash"`
	Email                     string         `db:"email"`
	Name                      sql.NullString `db:"name"`
	Handle                    sql.NullString `db:"handle"`
	DateOfBirth               sql.NullTime   `db:"dob"`
	About                     sql.NullString `db:"about"`
	CommunityOwnerCount       uint64         `db:"community_owner_count"`
	CommunityParticipantCount uint64         `db:"community_participant_count"`
	Verified                  bool           `db:"verified"`
	LastActiveAt              time.Time      `db:"last_active_at"`
	CFAvatarImagesID          sql.NullString `db:"cf_avatar_images_id"`
	AvatarFileID              sql.NullInt64  `db:"avatar_file_id"`
}

func (c Users) ToFiberMap() fiber.Map {
	var avatarUrl *string = nil

	if c.CFAvatarImagesID.Valid {
		s := os.Getenv("CLOUDFLARE_IMAGES_PROXY") + c.CFAvatarImagesID.String + "/public"
		avatarUrl = &s
	}

	return fiber.Map{
		"id":         security_helpers.Encode(c.ID, USERS_TYPE, c.Salt),
		"created_at": c.CreatedAt.Format(time.RFC3339),
		"name":       c.Name.String,
		"handle":     c.Handle.String,
		"about":      c.About.String,
		"avatar_url": avatarUrl,
	}
}

var GHOST_USER = Users{
	ID:                        0,
	CreatedAt:                 time.Now(),
	Salt:                      "ghost",
	PasswordHash:              "beepbeep",
	Email:                     "ghost@wikid.app",
	Handle:                    sql.NullString{String: "Ghost"},
	Name:                      sql.NullString{String: "Ghost"},
	CommunityOwnerCount:       0,
	CommunityParticipantCount: 0,
}

var USERS_TYPE = "Users"
