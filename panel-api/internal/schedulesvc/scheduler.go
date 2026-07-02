// Package schedulesvc runs per-server automations: a background loop that asks
// the schedules repo which schedules are due (per their interval) and executes
// each one's action (power / backup / console command) against the server.
package schedulesvc

import (
	"context"
	"log"
	"time"

	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/repo"
	"github.com/Notbangbang-dev/sky-panel/panel-api/internal/serversvc"
)

type Scheduler struct {
	schedules *repo.Schedules
	svc       *serversvc.Service
	tick      time.Duration
}

func NewScheduler(schedules *repo.Schedules, svc *serversvc.Service, tick time.Duration) *Scheduler {
	if tick <= 0 {
		tick = time.Minute
	}
	return &Scheduler{schedules: schedules, svc: svc, tick: tick}
}

// Run blocks until ctx is cancelled, running due automations every tick.
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
	now := time.Now().UTC()
	due, err := s.schedules.Due(now)
	if err != nil {
		log.Printf("schedulesvc: failed to query due schedules: %v", err)
		return
	}
	for _, sch := range due {
		// Mark it run first so a slow/failing action can't cause a tight retry
		// loop firing the same automation every tick.
		if err := s.schedules.MarkRun(sch.ID, now); err != nil {
			log.Printf("schedulesvc: mark run %s failed: %v", sch.ID, err)
			continue
		}
		if err := s.svc.RunScheduleAction(sch.ServerID, sch.Action, sch.Payload); err != nil {
			log.Printf("schedulesvc: schedule %s (%s on %s) failed: %v", sch.ID, sch.Action, sch.ServerID, err)
			continue
		}
		log.Printf("schedulesvc: ran %s (%s) on server %s", sch.ID, sch.Action, sch.ServerID)
	}
}
