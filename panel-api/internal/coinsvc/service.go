// Package coinsvc implements Sky Panel's coin economy: AFK-page accrual, a
// daily login reward, and admin balance adjustments. All crediting is
// server-authoritative — the client only ever sends a heartbeat or a claim
// request, never an amount.
package coinsvc

import (
	"errors"
	"time"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

const (
	// A heartbeat faster than this is ignored (client retry/double-click,
	// or an attempt to spam heartbeats for extra credit).
	AFKHeartbeatMinInterval = 20 * time.Second
	// A gap longer than this since the last heartbeat means the AFK session
	// lapsed (tab closed, laptop slept, etc.) — no credit for that gap, the
	// session simply restarts from now.
	AFKHeartbeatMaxInterval = 90 * time.Second
	AFKCoinsPerHeartbeat    = 1

	DailyRewardAmount   = 100
	DailyRewardInterval = 24 * time.Hour
)

var ErrDailyRewardAlreadyClaimed = errors.New("daily reward already claimed")

type Service struct {
	Users        *repo.Users
	Ledger       *repo.Ledger
	AFK          *repo.AFKState
	DailyRewards *repo.DailyRewards

	// now is overridden in tests; production code leaves it nil and gets
	// time.Now.
	now func() time.Time
}

func NewService(users *repo.Users, ledger *repo.Ledger, afk *repo.AFKState, dailyRewards *repo.DailyRewards) *Service {
	return &Service{Users: users, Ledger: ledger, AFK: afk, DailyRewards: dailyRewards}
}

func (s *Service) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now().UTC()
}

// Heartbeat records one AFK-page heartbeat for userID and returns how many
// coins (if any) were credited plus the resulting balance. A credit of 0 is
// not an error — it just means this particular heartbeat was too soon after
// the last one, or came after too long a gap to count as continuous.
func (s *Service) Heartbeat(userID string) (credited, balance int64, err error) {
	now := s.clock()

	last, found, err := s.AFK.LastHeartbeat(userID)
	if err != nil {
		return 0, 0, err
	}

	if found {
		elapsed := now.Sub(last)
		if elapsed >= AFKHeartbeatMinInterval && elapsed <= AFKHeartbeatMaxInterval {
			credited = AFKCoinsPerHeartbeat
		}
	}

	if err := s.AFK.SetLastHeartbeat(userID, now); err != nil {
		return 0, 0, err
	}

	if credited == 0 {
		balance, err = s.balance(userID)
		return 0, balance, err
	}

	balance, err = s.Ledger.AddEntry(userID, credited, models.ReasonAFKAccrual, "")
	if err != nil {
		return 0, 0, err
	}
	return credited, balance, nil
}

// ClaimDailyReward credits DailyRewardAmount coins if at least
// DailyRewardInterval has passed since the user's last claim.
func (s *Service) ClaimDailyReward(userID string) (credited, balance int64, err error) {
	now := s.clock()

	last, found, err := s.DailyRewards.LastClaimed(userID)
	if err != nil {
		return 0, 0, err
	}
	if found && now.Sub(last) < DailyRewardInterval {
		return 0, 0, ErrDailyRewardAlreadyClaimed
	}

	if err := s.DailyRewards.SetLastClaimed(userID, now); err != nil {
		return 0, 0, err
	}

	balance, err = s.Ledger.AddEntry(userID, DailyRewardAmount, models.ReasonDailyReward, "")
	if err != nil {
		return 0, 0, err
	}
	return DailyRewardAmount, balance, nil
}

// AdminAdjust applies an arbitrary (positive or negative) coin adjustment,
// e.g. from the admin console.
func (s *Service) AdminAdjust(userID string, amount int64, note string) (balance int64, err error) {
	return s.Ledger.AddEntry(userID, amount, models.ReasonAdminAdjustment, note)
}

func (s *Service) balance(userID string) (int64, error) {
	u, err := s.Users.GetByID(userID)
	if err != nil {
		return 0, err
	}
	return u.Coins, nil
}
