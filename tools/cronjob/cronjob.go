package cronjob

import (
	"github.com/pkg/errors"
)

var (
	ErrInvalidParam = errors.New("invalid param")
)

type Job struct {
	title    string
	interval string
	job      func()
}

func NewJob(title string, interval string, job func()) (*Job, error) {
	if title == "" {
		return nil, errors.WithMessage(ErrInvalidParam, "job title not set")
	}
	if interval == "" {
		return nil, errors.WithMessagef(ErrInvalidParam, "interval for %s job not set", title)
	}
	if job == nil {
		return nil, errors.WithMessagef(ErrInvalidParam, "job for %s job not set", title)
	}

	return &Job{
		title:    title,
		interval: interval,
		job:      job,
	}, nil
}

func (job *Job) Title() string {
	return job.title
}

func (job *Job) Interval() string {
	return job.interval
}

func (job *Job) Func() func() {
	return job.job
}
