// Package retentionsvc runs periodic housekeeping that keeps unbounded tables
// from growing forever on the single-writer SQLite database: it prunes expired
// refresh tokens and audit-log entries older than a retention window.
package retentionsvc

import (
	"context"
	"log"
	"time"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
)

type Scheduler struct {
	tokens   *repo.RefreshTokens
	audit    *repo.Audit
	tick     time.Duration
	auditTTL time.Duration
}

// NewScheduler builds a retention loop. tick is how often it runs; auditTTL is
// how long audit entries are kept.
func NewScheduler(tokens *repo.RefreshTokens, audit *repo.Audit, tick, auditTTL time.Duration) *Scheduler {
	if tick <= 0 {
		tick = 6 * time.Hour
	}
	if auditTTL <= 0 {
		auditTTL = 90 * 24 * time.Hour
	}
	return &Scheduler{tokens: tokens, audit: audit, tick: tick, auditTTL: auditTTL}
}

// Run blocks until ctx is cancelled, sweeping once immediately and then every
// tick.
func (s *Scheduler) Run(ctx context.Context) {
	s.runOnce()
	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce()
		}
	}
}

func (s *Scheduler) runOnce() {
	now := time.Now().UTC()
	if n, err := s.tokens.DeleteExpired(now); err != nil {
		log.Printf("retentionsvc: prune expired refresh tokens: %v", err)
	} else if n > 0 {
		log.Printf("retentionsvc: pruned %d expired refresh token(s)", n)
	}
	if n, err := s.audit.PruneOlderThan(now.Add(-s.auditTTL)); err != nil {
		log.Printf("retentionsvc: prune old audit entries: %v", err)
	} else if n > 0 {
		log.Printf("retentionsvc: pruned %d audit entr(ies) older than %s", n, s.auditTTL)
	}
}
