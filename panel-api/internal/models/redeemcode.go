package models

import "time"

// RedeemCode is an admin-minted code users can redeem once for coins.
// MaxUses of 0 means unlimited total redemptions (still capped to one per user).
type RedeemCode struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Coins     int64     `json:"coins"`
	MaxUses   int       `json:"max_uses"`
	Uses      int       `json:"uses"`
	CreatedAt time.Time `json:"created_at"`
}
