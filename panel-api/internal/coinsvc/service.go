// Package coinsvc implements Sky Panel's coin economy: AFK-page accrual, a
// daily login reward, and admin balance adjustments. All crediting is
// server-authoritative — the client only ever sends a heartbeat or a claim
// request, never an amount.
package coinsvc

import (
	"errors"
	"strconv"
	"strings"
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

// Setting keys that let an admin tune the economy at runtime (via the generic
// settings store). Each falls back to the constant above when unset/invalid.
const (
	SettingAFKCoins           = "afk.coins_per_heartbeat"
	SettingAFKMinSeconds      = "afk.min_interval_seconds"
	SettingAFKMaxSeconds      = "afk.max_interval_seconds"
	SettingDailyAmount        = "daily_reward.amount"
	SettingDailyIntervalHours = "daily_reward.interval_hours"
)

var ErrDailyRewardAlreadyClaimed = errors.New("daily reward already claimed")

// ErrAFKSessionElsewhere means another browser tab currently owns this user's
// AFK session, so this heartbeat is not allowed to earn. The active session
// must go stale (no heartbeat for AFKHeartbeatMaxInterval) before another tab
// can take over — this is what stops a user farming coins in many tabs.
var ErrAFKSessionElsewhere = errors.New("afk session active in another tab")

type Service struct {
	Users        *repo.Users
	Ledger       *repo.Ledger
	AFK          *repo.AFKState
	DailyRewards *repo.DailyRewards
	Settings     *repo.Settings

	// now is overridden in tests; production code leaves it nil and gets
	// time.Now.
	now func() time.Time
}

func NewService(users *repo.Users, ledger *repo.Ledger, afk *repo.AFKState, dailyRewards *repo.DailyRewards, settings *repo.Settings) *Service {
	return &Service{Users: users, Ledger: ledger, AFK: afk, DailyRewards: dailyRewards, Settings: settings}
}

func (s *Service) clock() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now().UTC()
}

// Runtime-tunable economy parameters, each falling back to its constant when
// the corresponding setting is unset or invalid.
func (s *Service) afkCoins() int64 { return s.settingInt64(SettingAFKCoins, AFKCoinsPerHeartbeat) }
func (s *Service) afkMinInterval() time.Duration {
	return s.settingSeconds(SettingAFKMinSeconds, AFKHeartbeatMinInterval)
}
func (s *Service) afkMaxInterval() time.Duration {
	return s.settingSeconds(SettingAFKMaxSeconds, AFKHeartbeatMaxInterval)
}
func (s *Service) dailyAmount() int64 { return s.settingInt64(SettingDailyAmount, DailyRewardAmount) }
func (s *Service) dailyInterval() time.Duration {
	return time.Duration(s.settingInt64(SettingDailyIntervalHours, int64(DailyRewardInterval/time.Hour))) * time.Hour
}

func (s *Service) settingSeconds(key string, fallback time.Duration) time.Duration {
	return time.Duration(s.settingInt64(key, int64(fallback/time.Second))) * time.Second
}

func (s *Service) settingInt64(key string, fallback int64) int64 {
	if s.Settings == nil {
		return fallback
	}
	v, found, err := s.Settings.Get(key)
	if err != nil || !found {
		return fallback
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

// HeartbeatResult is what a single AFK heartbeat produced.
type HeartbeatResult struct {
	Credited       int64
	Balance        int64
	SessionStarted time.Time
}

// Heartbeat records one AFK-page heartbeat for userID from a specific browser
// tab (sessionID) and returns how many coins (if any) were credited, the
// resulting balance, and when the current earning session began.
//
// Only one session earns at a time. If a different session is still fresh
// (has beaten within AFKHeartbeatMaxInterval), this heartbeat is rejected with
// ErrAFKSessionElsewhere and earns nothing — that's the multi-tab guard. Once
// the active session goes stale, the next heartbeat (from any tab) takes over
// and starts a fresh session. A credit of 0 is not an error: it just means
// this beat was too soon after the last, or the session had just started.
func (s *Service) Heartbeat(userID, sessionID string) (HeartbeatResult, error) {
	now := s.clock()

	prev, found, err := s.AFK.Get(userID)
	if err != nil {
		return HeartbeatResult{}, err
	}

	minInterval, maxInterval := s.afkMinInterval(), s.afkMaxInterval()
	// Guard against a misconfigured window (min >= max) that would otherwise
	// make the credit condition impossible and silently stop all earning —
	// fall back to the built-in defaults instead.
	if minInterval >= maxInterval {
		minInterval, maxInterval = AFKHeartbeatMinInterval, AFKHeartbeatMaxInterval
	}

	// Is a session currently live (someone beat recently)?
	sessionLive := found && prev.SessionID != "" && now.Sub(prev.LastHeartbeat) <= maxInterval

	if sessionLive && prev.SessionID != sessionID {
		return HeartbeatResult{}, ErrAFKSessionElsewhere
	}

	// This tab continues the live session, or (re)starts one if none is live.
	continuing := sessionLive && prev.SessionID == sessionID
	sessionStarted := now
	var credited int64
	if continuing {
		sessionStarted = prev.SessionStartedAt
		if elapsed := now.Sub(prev.LastHeartbeat); elapsed >= minInterval && elapsed <= maxInterval {
			credited = s.afkCoins()
		}
	}

	if err := s.AFK.Set(userID, sessionID, now, sessionStarted); err != nil {
		return HeartbeatResult{}, err
	}

	balance := int64(0)
	if credited > 0 {
		balance, err = s.Ledger.AddEntry(userID, credited, models.ReasonAFKAccrual, "")
		if err != nil {
			return HeartbeatResult{}, err
		}
	} else {
		balance, err = s.balance(userID)
		if err != nil {
			return HeartbeatResult{}, err
		}
	}

	return HeartbeatResult{Credited: credited, Balance: balance, SessionStarted: sessionStarted}, nil
}

// ClaimDailyReward credits DailyRewardAmount coins if at least
// DailyRewardInterval has passed since the user's last claim.
func (s *Service) ClaimDailyReward(userID string) (credited, balance int64, err error) {
	now := s.clock()

	last, found, err := s.DailyRewards.LastClaimed(userID)
	if err != nil {
		return 0, 0, err
	}
	if found && now.Sub(last) < s.dailyInterval() {
		return 0, 0, ErrDailyRewardAlreadyClaimed
	}

	if err := s.DailyRewards.SetLastClaimed(userID, now); err != nil {
		return 0, 0, err
	}

	amount := s.dailyAmount()
	balance, err = s.Ledger.AddEntry(userID, amount, models.ReasonDailyReward, "")
	if err != nil {
		return 0, 0, err
	}
	return amount, balance, nil
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
