package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/crucial707/hci-asset/internal/repo"
	"github.com/robfig/cron/v3"
)

// Run starts a background scheduler that loads enabled scan schedules from the DB
// and runs runScan(target) at each schedule's cron time. It reloads schedules every 60 seconds.
func Run(scheduleRepo *repo.ScheduleRepo, runScan func(target string)) {
	c := cron.New()
	var mu sync.Mutex
	entryByID := make(map[int]cron.EntryID) // schedule ID -> cron entry

	syncSchedules := func() {
		mu.Lock()
		defer mu.Unlock()

		// Remove all current entries so we reflect DB (and pick up edits)
		for _, entryID := range entryByID {
			c.Remove(entryID)
		}
		entryByID = make(map[int]cron.EntryID)

		list, err := scheduleRepo.ListEnabled(context.Background())
		if err != nil {
			log.Printf("scheduler: list enabled schedules: %v", err)
			return
		}

		for _, s := range list {
			target := s.Target
			expr := s.CronExpr
			entryID, err := c.AddFunc(expr, func() { runScan(target) })
			if err != nil {
				log.Printf("scheduler: invalid cron_expr %q for schedule id=%d: %v", expr, s.ID, err)
				continue
			}
			entryByID[s.ID] = entryID
			log.Printf("scheduler: added schedule id=%d target=%q cron=%q", s.ID, target, expr)
		}
	}

	// Initial load
	syncSchedules()
	c.Start()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		syncSchedules()
	}
}
