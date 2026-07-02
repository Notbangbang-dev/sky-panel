package models

import "time"

const (
	ReasonDailyReward     = "daily_reward"
	ReasonAFKAccrual      = "afk_accrual"
	ReasonAdminAdjustment = "admin_adjustment"
	ReasonStorePurchase   = "store_purchase"
	ReasonStoreRefund     = "store_refund"
)

type LedgerEntry struct {
	ID        string
	UserID    string
	Amount    int64
	Reason    string
	Metadata  string
	CreatedAt time.Time
}
