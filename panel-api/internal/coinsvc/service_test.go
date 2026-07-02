package coinsvc

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/models"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/store"
)

// fakeClock lets tests advance time deterministically without sleeping.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestServiceAndUser(t *testing.T) (*Service, *fakeClock, string) {
	t.Helper()

	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", name)

	db, err := store.Open(dsn)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	users := repo.NewUsers(db)
	now := time.Now().UTC()
	u := &models.User{ID: uuid.NewString(), Email: uuid.NewString() + "@example.com", Username: "u-" + uuid.NewString(), PasswordHash: "hash", Role: models.RoleUser, CreatedAt: now, UpdatedAt: now}
	if err := users.Create(u); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	svc := NewService(users, repo.NewLedger(db), repo.NewAFKState(db), repo.NewDailyRewards(db))
	clock := &fakeClock{t: now}
	svc.now = clock.now

	return svc, clock, u.ID
}

const testSession = "tab-1"

func TestHeartbeatFirstCallCreditsNothingButRecordsState(t *testing.T) {
	svc, _, userID := newTestServiceAndUser(t)

	res, err := svc.Heartbeat(userID, testSession)
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if res.Credited != 0 {
		t.Errorf("expected 0 credited on first heartbeat, got %d", res.Credited)
	}
	if res.Balance != 0 {
		t.Errorf("expected balance 0, got %d", res.Balance)
	}
}

func TestHeartbeatWithinWindowCredits(t *testing.T) {
	svc, clock, userID := newTestServiceAndUser(t)

	svc.Heartbeat(userID, testSession) // establishes baseline
	clock.advance(30 * time.Second)

	res, err := svc.Heartbeat(userID, testSession)
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if res.Credited != AFKCoinsPerHeartbeat {
		t.Errorf("expected %d credited, got %d", AFKCoinsPerHeartbeat, res.Credited)
	}
	if res.Balance != AFKCoinsPerHeartbeat {
		t.Errorf("expected balance %d, got %d", AFKCoinsPerHeartbeat, res.Balance)
	}
}

func TestHeartbeatTooSoonDoesNotCredit(t *testing.T) {
	svc, clock, userID := newTestServiceAndUser(t)

	svc.Heartbeat(userID, testSession)
	clock.advance(5 * time.Second) // well under AFKHeartbeatMinInterval

	res, err := svc.Heartbeat(userID, testSession)
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if res.Credited != 0 {
		t.Errorf("expected 0 credited for a too-soon heartbeat, got %d", res.Credited)
	}
}

func TestHeartbeatTooLongGapDoesNotCredit(t *testing.T) {
	svc, clock, userID := newTestServiceAndUser(t)

	svc.Heartbeat(userID, testSession)
	clock.advance(10 * time.Minute) // well over AFKHeartbeatMaxInterval

	res, err := svc.Heartbeat(userID, testSession)
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if res.Credited != 0 {
		t.Errorf("expected 0 credited after a lapsed session, got %d", res.Credited)
	}

	// But the session should have restarted cleanly: the next in-window
	// heartbeat credits normally.
	clock.advance(30 * time.Second)
	res, err = svc.Heartbeat(userID, testSession)
	if err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	if res.Credited != AFKCoinsPerHeartbeat {
		t.Errorf("expected session to restart and credit normally, got %d", res.Credited)
	}
}

func TestHeartbeatRejectsSecondConcurrentTab(t *testing.T) {
	svc, clock, userID := newTestServiceAndUser(t)

	// Tab A owns the session.
	svc.Heartbeat(userID, "tab-A")
	clock.advance(30 * time.Second)

	// Tab B tries to earn while A is still fresh — rejected, no credit.
	if _, err := svc.Heartbeat(userID, "tab-B"); !errors.Is(err, ErrAFKSessionElsewhere) {
		t.Fatalf("expected ErrAFKSessionElsewhere for a second tab, got %v", err)
	}

	// Tab A keeps earning.
	res, err := svc.Heartbeat(userID, "tab-A")
	if err != nil {
		t.Fatalf("Heartbeat A: %v", err)
	}
	if res.Credited != AFKCoinsPerHeartbeat {
		t.Errorf("expected tab A to keep earning, got %d", res.Credited)
	}

	// Once A goes stale, B can take over on its next beat.
	clock.advance(10 * time.Minute)
	if _, err := svc.Heartbeat(userID, "tab-B"); err != nil {
		t.Fatalf("expected tab B to take over a stale session, got %v", err)
	}
	clock.advance(30 * time.Second)
	res, err = svc.Heartbeat(userID, "tab-B")
	if err != nil {
		t.Fatalf("Heartbeat B: %v", err)
	}
	if res.Credited != AFKCoinsPerHeartbeat {
		t.Errorf("expected tab B to earn after takeover, got %d", res.Credited)
	}
}

func TestClaimDailyRewardSucceedsThenBlocksUntilIntervalPasses(t *testing.T) {
	svc, clock, userID := newTestServiceAndUser(t)

	credited, balance, err := svc.ClaimDailyReward(userID)
	if err != nil {
		t.Fatalf("ClaimDailyReward: %v", err)
	}
	if credited != DailyRewardAmount || balance != DailyRewardAmount {
		t.Errorf("expected credited=balance=%d, got credited=%d balance=%d", DailyRewardAmount, credited, balance)
	}

	if _, _, err := svc.ClaimDailyReward(userID); !errors.Is(err, ErrDailyRewardAlreadyClaimed) {
		t.Errorf("expected ErrDailyRewardAlreadyClaimed on immediate re-claim, got %v", err)
	}

	clock.advance(DailyRewardInterval - time.Minute)
	if _, _, err := svc.ClaimDailyReward(userID); !errors.Is(err, ErrDailyRewardAlreadyClaimed) {
		t.Errorf("expected still blocked just under the interval, got %v", err)
	}

	clock.advance(2 * time.Minute)
	credited, balance, err = svc.ClaimDailyReward(userID)
	if err != nil {
		t.Fatalf("expected claim to succeed after the interval has passed: %v", err)
	}
	if credited != DailyRewardAmount || balance != 2*DailyRewardAmount {
		t.Errorf("expected credited=%d balance=%d, got credited=%d balance=%d", DailyRewardAmount, 2*DailyRewardAmount, credited, balance)
	}
}

func TestAdminAdjustCreditsAndDebits(t *testing.T) {
	svc, _, userID := newTestServiceAndUser(t)

	balance, err := svc.AdminAdjust(userID, 500, "gift")
	if err != nil {
		t.Fatalf("AdminAdjust credit: %v", err)
	}
	if balance != 500 {
		t.Errorf("expected balance 500, got %d", balance)
	}

	balance, err = svc.AdminAdjust(userID, -200, "correction")
	if err != nil {
		t.Fatalf("AdminAdjust debit: %v", err)
	}
	if balance != 300 {
		t.Errorf("expected balance 300, got %d", balance)
	}
}

func TestAdminAdjustCannotGoNegative(t *testing.T) {
	svc, _, userID := newTestServiceAndUser(t)

	if _, err := svc.AdminAdjust(userID, -1, "oops"); !errors.Is(err, repo.ErrInsufficientBalance) {
		t.Errorf("expected ErrInsufficientBalance, got %v", err)
	}
}

func TestLedgerHistoryIsRecorded(t *testing.T) {
	svc, _, userID := newTestServiceAndUser(t)

	svc.AdminAdjust(userID, 50, "one")
	svc.AdminAdjust(userID, 25, "two")

	entries, err := svc.Ledger.ListByUser(userID, 10)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 ledger entries, got %d", len(entries))
	}
	// Most recent first.
	if entries[0].Metadata != "two" || entries[1].Metadata != "one" {
		t.Errorf("unexpected ledger order: %+v", entries)
	}
}
