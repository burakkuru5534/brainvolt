package app

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking"
)

// maintenanceInterval paces the worker's periodic jobs.
const maintenanceInterval = 5 * time.Minute

// rankingMaintainArgs is ranking's periodic job: archive ended boards,
// close an ended season (schema-ranking §6).
type rankingMaintainArgs struct{}

func (rankingMaintainArgs) Kind() string { return "ranking.maintain" }

// addJobs registers the worker's periodic jobs and returns their
// schedules; River runs them on the elected leader only.
func addJobs(workers *river.Workers, rk *ranking.Module) []*river.PeriodicJob {
	river.AddWorker(workers, river.WorkFunc(func(ctx context.Context, _ *river.Job[rankingMaintainArgs]) error {
		return rk.Maintain(ctx)
	}))
	return []*river.PeriodicJob{
		river.NewPeriodicJob(river.PeriodicInterval(maintenanceInterval), func() (river.JobArgs, *river.InsertOpts) {
			return rankingMaintainArgs{}, &river.InsertOpts{UniqueOpts: river.UniqueOpts{ByPeriod: maintenanceInterval}}
		}, &river.PeriodicJobOpts{RunOnStart: true}),
	}
}
