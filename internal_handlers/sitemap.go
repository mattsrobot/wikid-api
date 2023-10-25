package internal_handlers

import (
	"cmp"
	"context"
	"slices"

	"github.com/macwilko/exotic-auth/db/chat_users_db/model"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

func Sitemap(c *fiber.Ctx, ctx context.Context, db *sqlx.DB, wRdb *redis.Client, rRdb *redis.Client, queue *asynq.Client) error {
	slog.Info("Starting site map ✅")

	var communities []model.Communities

	err := db.Select(&communities, "SELECT * FROM communities WHERE private = ?", false)

	if err != nil {
		slog.Error("Database problem 💀",
			slog.String("error", err.Error()),
			slog.String("area", "selecting the communities"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Sitemap error",
			}},
		})
	}

	var communitiesIds = []uint64{}

	for _, community := range communities {
		communitiesIds = append(communitiesIds, community.ID)
	}

	channelsQuery, channelsArgs, err := sqlx.In("SELECT * FROM channels WHERE community_id IN (?)", communitiesIds)

	if err != nil {
		slog.Error("Database problem 💀",
			slog.String("error", err.Error()),
			slog.String("area", "creating the query for channels"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Sitemap error",
			}},
		})
	}

	channelsQuery = db.Rebind(channelsQuery)

	channels := []model.Channels{}

	err = db.Select(&channels, channelsQuery, channelsArgs...)

	if err != nil {
		slog.Error("Database problem 💀",
			slog.String("error", err.Error()),
			slog.String("area", "after the bind to community_id query"))

		return c.Status(fiber.StatusOK).JSON(&fiber.Map{
			"errors": []fiber.Map{{
				"message": "Sitemap error",
			}},
		})
	}

	channelsMap := make(map[uint64][]model.Channels)

	for _, channel := range channels {
		list, ok := channelsMap[channel.CommunityID]

		if ok {
			channelsMap[channel.CommunityID] = append(list, channel)
		} else {
			channelsMap[channel.CommunityID] = []model.Channels{channel}
		}
	}

	var mappedCommunities = []fiber.Map{}

	sortCommunities := func(a, b model.Communities) int {
		return cmp.Compare(a.Handle, b.Handle)
	}

	sortChannels := func(a, b model.Channels) int {
		return cmp.Compare(a.Handle, b.Handle)
	}

	slices.SortFunc(communities, sortCommunities)

	for _, community := range communities {

		communityChannels, ok := channelsMap[community.ID]

		if !ok || len(communityChannels) == 0 {
			continue
		}

		slices.SortFunc(communityChannels, sortChannels)

		var mappedChannels = []fiber.Map{}

		for _, channel := range communityChannels {
			mappedChannels = append(mappedChannels, fiber.Map{
				"handle": channel.Handle,
			})
		}

		mappedCommunities = append(mappedCommunities, fiber.Map{
			"handle":   community.Handle,
			"channels": mappedChannels,
		})
	}

	return c.Status(fiber.StatusOK).JSON(&fiber.Map{
		"communities": mappedCommunities,
	})
}
