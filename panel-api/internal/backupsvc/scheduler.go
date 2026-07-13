// Package backupsvc runs scheduled backups: a background loop that
// periodically asks the servers repo which servers are due for a backup
// (per their backup_interval_hours) and dispatches one for each.
package backupsvc

import (
	"context"
	"log"
	"time"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/serversvc"
)

type Scheduler struct {
	servers *repo.Servers
	svc     *serversvc.Service
	tick    time.Duration
}

func NewScheduler(servers *repo.Servers, svc *serversvc.Service, tick time.Duration) *Scheduler {
	if tick <= 0 {
		tick = 15 * time.Minute
	}
	return &Scheduler{servers: servers, svc: svc, tick: tick}
}

// Run blocks until ctx is cancelled, checking for due backups every tick.
func (s *Scheduler) Run(ctx context.Context) {
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
	due, err := s.servers.DueForBackup(time.Now().UTC())
	if err != nil {
		log.Printf("backupsvc: failed to query due backups: %v", err)
		return
	}
	for _, server := range due {
		// A suspended server must not keep consuming node disk via scheduled
		// backups — suspension freezes all automation.
		if server.Suspended {
			continue
		}
		if _, err := s.svc.Backup(server.ID); err != nil {
			// A node being offline is expected/transient — log and move on;
			// the server stays "due" and will be retried next tick.
			log.Printf("backupsvc: scheduled backup for server %s failed: %v", server.ID, err)
			continue
		}
		log.Printf("backupsvc: scheduled backup completed for server %s", server.ID)
	}
}
