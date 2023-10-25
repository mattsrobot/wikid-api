package model

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

// Stored in redis. This means we can quickly fetch message reactions without hitting the DB.

// Each message can have a number of reactions (emojis, etc).
// We store a map of unicode strings to users.
type MessagesReactions struct {
	MessageID uint64                    `json:"message_id"`
	Reactions map[string][]ReactionUser `json:"reactions"`
}

// A user can react to a message.
// We store the basic user information here.
type ReactionUser struct {
	UserID     uint64 `json:"user_id"`
	UserHandle string `json:"user_handle"`
}

func (f *ReactionUser) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		UserID     uint64 `json:"user_id"`
		UserHandle string `json:"user_handle"`
	}{
		UserID:     f.UserID,
		UserHandle: f.UserHandle,
	})
}

// Helpers for ReactionUsers
func RemoveReactionUser(slice []ReactionUser, s int) []ReactionUser {
	return append(slice[:s], slice[s+1:]...)
}

func (c MessagesReactions) ToFiberMap() fiber.Map {

	mm := fiber.Map{}

	for i, m := range c.Reactions {

		ru := make([]fiber.Map, len(m))

		for i, m := range m {
			ru[i] = fiber.Map{
				"user_id":     m.UserID,
				"user_handle": m.UserHandle,
			}
		}

		mm[i] = ru
	}

	return mm
}
