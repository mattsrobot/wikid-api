package model

import (
	"database/sql"
	"time"
)

type BetaUsers struct {
	ID        uint64       `db:"id"`
	CreatedAt time.Time    `db:"created_at"`
	UpdatedAt sql.NullTime `db:"updated_at"`
	Email     string       `db:"email"`
	Redeemed  bool         `db:"redeemed"`
	Enabled   bool         `db:"enabled"`
}
