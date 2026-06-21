package cronjob

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

var (
	ErrGraceFullShutdown = errors.New("gs ends with err")
	ErrJobNotFound       = errors.New("job with required title not found")
)

type Cron struct {
	c    *cron.Cron
	data map[string]cron.EntryID
}

func NewCron(jobs ...*Job) (*Cron, error) {
	c := cron.New()
	jobIDs := make(map[string]cron.EntryID, len(jobs))

	for _, job := range jobs {
		id, err := c.AddFunc(job.Interval(), job.Func())
		if err != nil {
			return nil, errors.WithMessagef(ErrInvalidParam, "add task %s to cron fail", job.Title())
		}

		jobIDs[job.title] = id
	}

	return &Cron{
		c:    c,
		data: jobIDs,
	}, nil
}

func (cron *Cron) Start() error {
	cron.c.Start()

	slog.Info("cron started")

	return nil
}

func (cron *Cron) Stop(ctx context.Context) error {
	ctxCron := cron.c.Stop()
	select {
	case <-ctxCron.Done():
		slog.Info("Crone finished gracefully")
		return nil
	case <-ctx.Done():
		slog.Warn("crone finished with err", "err", ErrGraceFullShutdown)
		return errors.WithMessage(ErrGraceFullShutdown, "timeout was ended")
	}
}

func (cron *Cron) StopJob(title string) error {
	id, ok := cron.data[title]
	if !ok {
		return errors.WithMessagef(ErrJobNotFound, "job with title %s was not found", title)
	}

	cron.c.Remove(id)
	slog.Info("job was removed from cron", "job_title", title)
	return nil
}
