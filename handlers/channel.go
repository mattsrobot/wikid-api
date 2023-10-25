package handlers

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/macwilko/exotic-auth/db/chat_users_db/model"
	"github.com/macwilko/exotic-auth/security_helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

func Channel(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	slog.Info("Starting fetch channel ✅")

	pageNumber := c.QueryInt("page", 0)

	user, userOk := c.Locals("viewer").(model.Users)

	/* Fetch community */

	communityHandle := Truncate(strings.ToLower(c.Params("communityHandle")), 255)

	community := model.Communities{}

	err := db.Get(&community, "SELECT * FROM communities WHERE handle = ? LIMIT 1", communityHandle)

	if err != nil {
		slog.Error("No community found 💀",
			slog.String("error", err.Error()),
			slog.String("area", "can't find  community"))

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	permissions := community.Permissions

	var defaultChannel string

	var mu *fiber.Map

	if userOk {

		var avatarUrl *string = nil

		if user.CFAvatarImagesID.Valid {
			s := os.Getenv("CLOUDFLARE_IMAGES_PROXY") + user.CFAvatarImagesID.String + "/public"
			avatarUrl = &s
		}

		mu = &fiber.Map{
			"id":         security_helpers.Encode(user.ID, model.USERS_TYPE, user.Salt),
			"handle":     user.Handle.String,
			"name":       user.Name.String,
			"avatar_url": avatarUrl,
		}

		pq := `
		SELECT view_channels, manage_channels, manage_community, create_invite, kick_members,
		ban_members, send_messages, attach_media, selected_channel_id
		FROM communities_users
		WHERE community_id = ?
		AND user_id = ?
		`

		var cu model.CommunitiesUsers

		err = db.Get(&cu, pq, community.ID, user.ID)

		if err != nil {
			slog.Error("Database problem 💀",
				slog.String("error", err.Error()))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		permissions = cu.Permissions

		if cu.SelectedChannelID > 0 {
			pcq := `
			SELECT handle
			FROM channels
			WHERE id = ?
			LIMIT 1
			`

			err = db.Get(&defaultChannel, pcq, cu.SelectedChannelID)

			if err != nil {
				return c.Status(fiber.StatusOK).JSON(&fiber.Map{
					"errors": []fiber.Map{{
						"message": "Not found",
					}},
				})
			}
		} else {
			pq := `
			SELECT handle
			FROM channels
			WHERE community_id = ?
			LIMIT 1
			`

			err = db.Get(&defaultChannel, pq, community.ID)

			if err != nil {
				return c.Status(fiber.StatusOK).JSON(&fiber.Map{
					"errors": []fiber.Map{{
						"message": "Not found",
					}},
				})
			}
		}

	} else {
		pq := `
			SELECT handle
			FROM channels
			WHERE community_id = ?
			LIMIT 1
			`

		err = db.Get(&defaultChannel, pq, community.ID)

		if err != nil {
			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}
	}

	severOwner := false

	if userOk {
		severOwner = user.ID == community.OwnerID
	}

	/* Got community  */

	/* Fetch channel */

	channelHandle := Truncate(strings.ToLower(c.Params("channelHandle")), 255)

	channel := model.Channels{}

	err = db.Get(&channel, "SELECT * FROM channels WHERE handle = ? AND community_id = ? LIMIT 1", channelHandle, community.ID)

	if err != nil {
		slog.Error("No channel found 💀", slog.String("error", err.Error()), slog.String("area", "can't find  channel"))

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	if channel.CommunityID != community.ID {
		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	communityUsers := []model.CommunitiesUsers{}

	err = db.Select(&communityUsers, "SELECT * FROM communities_users WHERE community_id = ?", community.ID)

	if err != nil {
		slog.Error("No channel found 💀", slog.String("error", err.Error()), slog.String("area", "can't find community users"))

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	/* Got channel  */

	/* Fetch community roles */

	cr := []model.CommunityRoles{}

	/* These roles are shown prominently */
	var vcr = []model.CommunityRoles{}

	err = db.Select(&cr, "SELECT * FROM community_roles WHERE community_id = ? ORDER BY priority ASC", community.ID)

	if err != nil {
		slog.Error("Database problem 💀", slog.String("error", err.Error()), slog.String("area", "can't select roles"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	crIdMap := make(map[uint64]model.CommunityRoles)

	for _, r := range cr {
		if r.ShowOnlineDifferently {
			vcr = append(vcr, r)
		}
		crIdMap[r.ID] = r
	}

	/* Got community roles */

	/* Fetch messages */

	messages := []model.Messages{}

	offset := 0

	if pageNumber > 0 {
		offset = pageNumber * 50
	}

	err = db.Select(&messages, "SELECT * FROM messages WHERE channel_id = ? ORDER BY id DESC LIMIT 50 OFFSET ?", channel.ID, offset)

	if err != nil {
		slog.Error("Database problem 💀", slog.String("error", err.Error()), slog.String("area", "can't select messages"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	slices.Reverse(messages)

	var messageIds = []uint64{}

	// map of message ids to files
	filesMap := make(map[uint64][]model.Files)

	var parentIDs = []uint64{}
	parentsMap := make(map[uint64]model.Messages)

	for _, m := range messages {
		messageIds = append(messageIds, m.ID)

		if m.ParentID > 0 {
			parentIDs = append(parentIDs, m.ParentID)
		}
	}

	if len(messageIds) > 0 {
		pmq, mpqArgs, err := sqlx.In("SELECT * FROM files WHERE message_id IN (?)", messageIds)

		if err != nil {
			slog.Error("Database problem 💀",
				slog.String("error", err.Error()),
				slog.String("area", "selecting message ids IN"))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		pmq = db.Rebind(pmq)

		files := []model.Files{}

		err = db.Select(&files, pmq, mpqArgs...)

		if err != nil {
			slog.Error("Database problem 💀",
				slog.String("error", err.Error()),
				slog.String("area", "after the bind to parentIds query"))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		for _, m := range files {
			if m.MessageID.Valid {
				fs, ok := filesMap[uint64(m.MessageID.Int64)]

				if !ok {
					filesMap[uint64(m.MessageID.Int64)] = []model.Files{m}
				} else {
					filesMap[uint64(m.MessageID.Int64)] = append(fs, m)
				}

			}
		}
	}

	if len(parentIDs) > 0 {
		pmq, mpqArgs, err := sqlx.In("SELECT * FROM messages WHERE id IN (?)", parentIDs)

		if err != nil {
			slog.Error("Database problem 💀",
				slog.String("error", err.Error()),
				slog.String("area", "selecting parent ids IN"))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		pmq = db.Rebind(pmq)

		parents := []model.Messages{}

		err = db.Select(&parents, pmq, mpqArgs...)

		if err != nil {
			slog.Error("Database problem 💀",
				slog.String("error", err.Error()),
				slog.String("area", "after the bind to parentIds query"))

			return c.Status(fiber.StatusOK).JSON(&fiber.Map{
				"errors": []fiber.Map{{
					"message": "Not found",
				}},
			})
		}

		for _, m := range parents {
			parentsMap[m.ID] = m
		}
	}

	var remaining uint64

	err = db.Get(&remaining, "SELECT count(*) FROM messages WHERE channel_id = ? ORDER BY id DESC", channel.ID)

	if err != nil {
		slog.Error("Database problem 💀", slog.String("error", err.Error()), slog.String("area", "can't select count of messages"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	nextPageAmount := (uint64(pageNumber) + 1) * 50

	if remaining > nextPageAmount {
		remaining = remaining - nextPageAmount
	} else {
		remaining = 0
	}

	/* Got messages */

	/* Fetch users from messages */

	uIdsMap := make(map[uint64]bool)
	var uIds = []uint64{}

	for _, m := range messages {
		if uIdsMap[m.UserID] {
			continue
		}
		uIdsMap[m.UserID] = true
		uIds = append(uIds, m.UserID)
	}

	for _, cu := range communityUsers {
		if uIdsMap[cu.UserID] {
			continue
		}
		uIdsMap[cu.UserID] = true
		uIds = append(uIds, cu.UserID)
	}

	users := []model.Users{}
	rolesUsers := []model.CommuniyRolesUsers{}

	usersQuery, usersArgs, err := sqlx.In("SELECT * FROM users WHERE id IN (?)", uIds)

	if err != nil {
		slog.Error("Database problem 💀", slog.String("error", err.Error()), slog.String("area", "selecting users IN"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	usersQuery = db.Rebind(usersQuery)

	err = db.Select(&users, usersQuery, usersArgs...)

	if err != nil {
		slog.Error("Database problem 💀", slog.String("error", err.Error()), slog.String("area", "after the bind to users query"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	/* Got users from messages */

	/* Fetch communiuty roles for users */

	rolesUsersQuery, rolesUsersArgs, err := sqlx.In("SELECT * FROM community_roles_users WHERE community_id = ? AND user_id IN (?) ", community.ID, uIds)

	if err != nil {
		slog.Error("Database problem 💀", slog.String("error", err.Error()), slog.String("area", "community_roles_users"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}

	rolesUsersQuery = db.Rebind(rolesUsersQuery)

	err = db.Select(&rolesUsers, rolesUsersQuery, rolesUsersArgs...)

	if err != nil {
		slog.Error("Database problem 💀", slog.String("error", err.Error()), slog.String("area", "after the bind selecting roles"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Not found",
			}},
		})
	}
	// Map of user id to their powerful community role
	urhMap := make(map[uint64]*model.CommunityRoles)
	// Map of user id to their community roles (most powerful first)
	urMap := make(map[uint64][]model.CommunityRoles)
	// Map of user id to their powerful prominent community role
	urVMap := make(map[uint64]*model.CommunityRoles)
	// Map of prominent community roles to users
	vcruMap := make(map[uint64][]model.Users)
	// Users with no prominent roles
	var unvr = []model.Users{}
	// Sort the roles by their displayed priority
	pc := func(a, b model.CommunityRoles) int {
		return cmp.Compare(a.Priority, b.Priority)
	}
	// Sort by handle
	ph := func(a, b model.Users) int {
		return cmp.Compare(a.Handle.String, b.Handle.String)
	}
	// Discover which users have prominent roles to be displayed
	for _, u := range users {
		// Find which roles this user has
		var userRoles = []model.CommunityRoles{}
		// Loop through each role/users join table
		for _, ru := range rolesUsers {
			// Add the role to the users roles if the user ID matches
			if r, ok := crIdMap[ru.CommunityRoleID]; ok && ru.UserID == u.ID {
				userRoles = append(userRoles, r)
			}
		}
		slices.SortFunc(userRoles, pc)
		// Save it in the map
		urMap[u.ID] = userRoles
		// Save the most powerful role also
		if len(userRoles) > 0 {
			urhMap[u.ID] = &userRoles[0]
			for _, ur := range userRoles {
				// Save the most powerful role that's displayed prominently
				if ur.ShowOnlineDifferently {
					urVMap[u.ID] = &ur
					break
				}
			}
		}
	}
	// Users that have no prominent role to display
	for _, u := range users {
		if _, found := urVMap[u.ID]; !found {
			unvr = append(unvr, u)
		}
	}

	slices.SortFunc(unvr, ph)
	// Users that belong to a prominent role
	for _, cr := range vcr {
		var uicr = []model.Users{}

		for _, u := range users {
			if mpvr, found := urVMap[u.ID]; found && mpvr.ID == cr.ID {
				uicr = append(uicr, u)
			}
		}

		vcruMap[cr.ID] = uicr
	}

	/* Assemble final response */

	monline := make([]fiber.Map, len(unvr))

	for i, nvr := range unvr {
		var avatarUrl *string = nil

		if nvr.CFAvatarImagesID.Valid {
			s := os.Getenv("CLOUDFLARE_IMAGES_PROXY") + nvr.CFAvatarImagesID.String + "/public"
			avatarUrl = &s
		}

		monline[i] = fiber.Map{
			"handle":     nvr.Handle.String,
			"name":       nvr.Name.String,
			"avatar_url": avatarUrl,
		}
	}

	moffline := make([]fiber.Map, 0)

	mvcr := make([]fiber.Map, len(vcr))

	for i, cr := range vcr {

		fu := []fiber.Map{}

		if rs, found := vcruMap[cr.ID]; found {
			for _, rss := range rs {

				var uhr *fiber.Map

				if hr, found := urhMap[rss.ID]; found && hr != nil {
					uhr = &fiber.Map{
						"name": hr.Name,
					}
				}

				urs := []fiber.Map{}

				if rs, found := urMap[rss.ID]; found {
					for _, rss := range rs {
						ur := fiber.Map{
							"name": rss.Name,
						}
						urs = append(urs, ur)
					}
				}

				var avatarUrl *string = nil

				if rss.CFAvatarImagesID.Valid {
					s := os.Getenv("CLOUDFLARE_IMAGES_PROXY") + rss.CFAvatarImagesID.String + "/public"
					avatarUrl = &s
				}

				u := fiber.Map{
					"handle":        rss.Handle.String,
					"name":          rss.Name.String,
					"powerful_role": uhr,
					"all_roles":     urs,
					"avatar_url":    avatarUrl,
				}

				fu = append(fu, u)
			}
		}

		mvcr[i] = fiber.Map{
			"id":    security_helpers.Encode(cr.ID, model.COMMUNITY_ROLES_TYPE, cr.Salt),
			"name":  cr.Name,
			"color": cr.Color,
			"users": fu,
		}
	}

	usersMap := make(map[uint64]model.Users)

	for _, u := range users {
		if _, found := usersMap[u.ID]; found {
			continue
		}
		usersMap[u.ID] = u
	}

	mm := make([]fiber.Map, len(messages))

	for i, m := range messages {

		mu := model.GHOST_USER

		if fu, found := usersMap[m.UserID]; found {
			mu = fu
		}

		var uhr *fiber.Map

		if hr, found := urhMap[m.UserID]; found && hr != nil {
			uhr = &fiber.Map{
				"name":  hr.Name,
				"color": hr.Color,
			}
		}

		reactions := model.MessagesReactions{}

		rkey := fmt.Sprintf("message-reactions-%d", m.ID)

		val, err := rRdb.Get(ctx, rkey).Result()

		if err == nil {
			json.Unmarshal([]byte(val), &reactions)
		}

		var avatarUrl *string = nil

		if mu.CFAvatarImagesID.Valid {
			s := os.Getenv("CLOUDFLARE_IMAGES_PROXY") + mu.CFAvatarImagesID.String + "/public"
			avatarUrl = &s
		}

		mappedMessage := fiber.Map{
			"id":         security_helpers.Encode(m.ID, model.MESSAGES_TYPE, m.Salt),
			"created_at": m.CreatedAt.Format(time.RFC3339),
			"text":       m.Text,
			"edited":     m.Edited,
			"user": fiber.Map{
				"name":          mu.Name.String,
				"handle":        mu.Handle.String,
				"powerful_role": uhr,
				"avatar_url":    avatarUrl,
			},
		}

		if m.UpdatedAt.Valid {
			maps.Copy(mappedMessage, fiber.Map{
				"updated_at": m.UpdatedAt.Time.Format(time.RFC3339),
			})
		}

		if reactions.MessageID == m.ID {
			maps.Copy(mappedMessage, fiber.Map{
				"reactions": reactions.ToFiberMap(),
			})
		}

		files, fok := filesMap[m.ID]

		if fok && len(files) > 0 {
			mfs := make([]fiber.Map, len(files))
			for i, f := range files {
				mfs[i] = f.ToFiberMap()
			}
			maps.Copy(mappedMessage, fiber.Map{
				"files": mfs,
			})
		}

		if p, pok := parentsMap[m.ParentID]; pok {

			mu := model.GHOST_USER

			if fu, found := usersMap[p.UserID]; found {
				mu = fu
			}

			var uhr *fiber.Map

			if hr, found := urhMap[p.UserID]; found && hr != nil {
				uhr = &fiber.Map{
					"name":  hr.Name,
					"color": hr.Color,
				}
			}

			var avatarUrl *string = nil

			if mu.CFAvatarImagesID.Valid {
				s := os.Getenv("CLOUDFLARE_IMAGES_PROXY") + mu.CFAvatarImagesID.String + "/public"
				avatarUrl = &s
			}

			maps.Copy(mappedMessage, fiber.Map{
				"parent": fiber.Map{
					"id":         security_helpers.Encode(p.ID, model.MESSAGES_TYPE, p.Salt),
					"created_at": p.CreatedAt.Format(time.RFC3339),
					"text":       p.Text,
					"edited":     p.Edited,
					"user": fiber.Map{
						"name":          mu.Name.String,
						"handle":        mu.Handle.String,
						"powerful_role": uhr,
						"avatar_url":    avatarUrl,
					},
				},
			})
		}

		mm[i] = mappedMessage
	}

	if userOk {
		if hr, found := urhMap[user.ID]; found && hr != nil && mu != nil {
			uhr := fiber.Map{
				"name":  hr.Name,
				"color": hr.Color,
			}

			maps.Copy(*mu, fiber.Map{
				"powerful_role": uhr,
			})
		}
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"id":         security_helpers.Encode(channel.ID, model.CHANNELS_TYPE, channel.Salt),
		"created_at": channel.CreatedAt.Format(time.RFC3339),
		"name":       channel.Name,
		"handle":     channel.Handle,
		"community": fiber.Map{
			"id":              security_helpers.Encode(community.ID, model.COMMUNITIES_TYPE, community.Salt),
			"created_at":      community.CreatedAt.Format(time.RFC3339),
			"name":            community.Name,
			"handle":          community.Handle,
			"permissions":     permissions.ToFiberMap(),
			"server_owner":    severOwner,
			"default_channel": defaultChannel,
		},
		"user":               mu,
		"prominent_roles":    mvcr,
		"others_online":      monline,
		"others_offline":     moffline,
		"messages":           mm,
		"message_count":      len(mm),
		"remaining_messages": remaining,
	})
}
