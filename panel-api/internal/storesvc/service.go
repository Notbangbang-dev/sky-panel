// Package storesvc is Sky Panel's coin store: users spend the coins they earn
// (AFK accrual, daily reward) on permanent quota upgrades. The catalog is
// fixed in code — every item raises one quota dimension by a set amount for a
// set price. Purchases are server-authoritative: the client only ever names
// an item, never an amount or a price.
package storesvc

import (
	"errors"
	"fmt"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

// Dimension names the quota a store item raises.
const (
	DimensionMemory    = "memory"
	DimensionCPU       = "cpu"
	DimensionDisk      = "disk"
	DimensionDatabases = "databases"
)

// Item is one purchasable quota pack. Amount is in bytes for memory/disk and
// in percent-of-one-core for CPU; Price is in coins.
type Item struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Dimension   string `json:"dimension"`
	Amount      int64  `json:"amount"`
	Price       int64  `json:"price"`
}

const (
	mb = 1024 * 1024
	gb = 1024 * mb
)

// Catalog is the fixed store. Prices are tuned against the earn rates in
// coinsvc (1 coin per ~25s AFK, 100 per daily claim).
var Catalog = []Item{
	{ID: "mem-512", Name: "+512 MB RAM", Description: "Add 512 MB to your memory quota, permanently.", Dimension: DimensionMemory, Amount: 512 * mb, Price: 250},
	{ID: "mem-1024", Name: "+1 GB RAM", Description: "Add 1 GB to your memory quota, permanently.", Dimension: DimensionMemory, Amount: 1 * gb, Price: 450},
	{ID: "cpu-50", Name: "+50% CPU", Description: "Add half a core to your CPU quota, permanently.", Dimension: DimensionCPU, Amount: 50, Price: 300},
	{ID: "cpu-100", Name: "+100% CPU", Description: "Add a full core to your CPU quota, permanently.", Dimension: DimensionCPU, Amount: 100, Price: 550},
	{ID: "disk-5", Name: "+5 GB disk", Description: "Add 5 GB to your disk quota, permanently.", Dimension: DimensionDisk, Amount: 5 * gb, Price: 200},
	{ID: "disk-10", Name: "+10 GB disk", Description: "Add 10 GB to your disk quota, permanently.", Dimension: DimensionDisk, Amount: 10 * gb, Price: 350},
	{ID: "db-1", Name: "+1 Database", Description: "Unlock one MariaDB database slot, permanently.", Dimension: DimensionDatabases, Amount: 1, Price: 300},
	{ID: "db-3", Name: "+3 Databases", Description: "Unlock three MariaDB database slots, permanently.", Dimension: DimensionDatabases, Amount: 3, Price: 750},
}

// ErrItemNotFound is returned when a purchase names an item not in the catalog.
var ErrItemNotFound = errors.New("store item not found")

type Service struct {
	Ledger *repo.Ledger
	Quotas *repo.Quotas
}

func NewService(ledger *repo.Ledger, quotas *repo.Quotas) *Service {
	return &Service{Ledger: ledger, Quotas: quotas}
}

func lookup(itemID string) (Item, bool) {
	for _, it := range Catalog {
		if it.ID == itemID {
			return it, true
		}
	}
	return Item{}, false
}

// Purchase debits the item's price and raises the matching quota dimension.
// The debit is atomic and guarded against a negative balance (repo.Ledger),
// so an unaffordable purchase fails cleanly with repo.ErrInsufficientBalance
// and no quota is granted. If the quota grant somehow fails after the debit,
// the coins are refunded so the two never drift apart.
func (s *Service) Purchase(userID, itemID string) (item Item, balance int64, err error) {
	item, ok := lookup(itemID)
	if !ok {
		return Item{}, 0, ErrItemNotFound
	}

	balance, err = s.Ledger.AddEntry(userID, -item.Price, models.ReasonStorePurchase, item.ID)
	if err != nil {
		return Item{}, 0, err
	}

	var grantErr error
	switch item.Dimension {
	case DimensionMemory:
		grantErr = s.Quotas.Add(userID, item.Amount, 0, 0)
	case DimensionCPU:
		grantErr = s.Quotas.Add(userID, 0, int(item.Amount), 0)
	case DimensionDisk:
		grantErr = s.Quotas.Add(userID, 0, 0, item.Amount)
	case DimensionDatabases:
		grantErr = s.Quotas.AddDatabases(userID, int(item.Amount))
	default:
		// A catalog item with an unknown dimension would otherwise debit the
		// coins and grant nothing — refuse so the refund path runs.
		grantErr = fmt.Errorf("unknown store dimension %q", item.Dimension)
	}
	if grantErr != nil {
		// Refund so coins and quota never diverge on a partial failure.
		if refunded, refundErr := s.Ledger.AddEntry(userID, item.Price, models.ReasonStoreRefund, item.ID); refundErr == nil {
			balance = refunded
		}
		return Item{}, balance, grantErr
	}

	return item, balance, nil
}
