package main

import (
	"context"
	"time"
)

type ExpireJob struct {
	repo   *Repo
	events *Events
}

func NewExpireJob(repo *Repo, events *Events) *ExpireJob {
	return &ExpireJob{repo: repo, events: events}
}

// Run expires unpaid orders every minute so their reserved stock is sold again.
func (j *ExpireJob) Run(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			ids, _ := j.repo.ExpirePending(ctx)
			for _, id := range ids {
				_ = j.events.Publish(ctx, id, "order_expired", "expired")
			}
		}
	}
}
