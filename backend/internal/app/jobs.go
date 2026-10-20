package app

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/notification"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking"
)

// maintenanceInterval paces the worker's periodic jobs.
const maintenanceInterval = 5 * time.Minute

// rankingMaintainArgs is ranking's periodic job: archive ended boards,
// close an ended season (schema-ranking §6).
type rankingMaintainArgs struct{}

func (rankingMaintainArgs) Kind() string { return "ranking.maintain" }

// dailySetArgs publishes today's and tomorrow's Daily Challenge sets
// ahead of the first players (category-modes §5.1).
type dailySetArgs struct{}

func (dailySetArgs) Kind() string { return "content.daily_sets" }

// notificationPurgeArgs deletes expired inbox items and old push
// delivery records (schema-platform §1).
type notificationPurgeArgs struct{}

func (notificationPurgeArgs) Kind() string { return "notification.purge" }

// addJobs registers the worker's periodic jobs and returns their
// schedules; River runs them on the elected leader only.
func addJobs(workers *river.Workers, rk *ranking.Module, cont *content.Module, notif *notification.Module) []*river.PeriodicJob {
	river.AddWorker(workers, river.WorkFunc(func(ctx context.Context, _ *river.Job[notificationPurgeArgs]) error {
		return notif.Purge(ctx)
	}))
	river.AddWorker(workers, river.WorkFunc(func(ctx context.Context, _ *river.Job[rankingMaintainArgs]) error {
		return rk.Maintain(ctx)
	}))
	river.AddWorker(workers, river.WorkFunc(func(ctx context.Context, _ *river.Job[dailySetArgs]) error {
		today := game.DailyDay(time.Now())
		for _, day := range []time.Time{today, today.AddDate(0, 0, 1)} {
			if _, err := cont.DailySet(ctx, day); err != nil {
				return err
			}
		}
		return nil
	}))
	return []*river.PeriodicJob{
		river.NewPeriodicJob(river.PeriodicInterval(24*time.Hour), func() (river.JobArgs, *river.InsertOpts) {
			return notificationPurgeArgs{}, &river.InsertOpts{UniqueOpts: river.UniqueOpts{ByPeriod: 24 * time.Hour}}
		}, nil),
		river.NewPeriodicJob(river.PeriodicInterval(time.Hour), func() (river.JobArgs, *river.InsertOpts) {
			return dailySetArgs{}, &river.InsertOpts{UniqueOpts: river.UniqueOpts{ByPeriod: time.Hour}}
		}, &river.PeriodicJobOpts{RunOnStart: true}),
		river.NewPeriodicJob(river.PeriodicInterval(maintenanceInterval), func() (river.JobArgs, *river.InsertOpts) {
			return rankingMaintainArgs{}, &river.InsertOpts{UniqueOpts: river.UniqueOpts{ByPeriod: maintenanceInterval}}
		}, &river.PeriodicJobOpts{RunOnStart: true}),
	}
}
