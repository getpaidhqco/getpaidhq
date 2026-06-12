package cron

import (
	"time"

	"getpaidhq/internal/core/port"

	"github.com/robfig/cron/v3"
)

type CronScheduler struct {
	cron   *cron.Cron
	logger port.Logger
}

func NewCronScheduler(logger port.Logger) port.Scheduler {
	c := cron.New()
	return &CronScheduler{
		logger: logger,
		cron:   c,
	}
}

func (c *CronScheduler) ScheduleTask(cronExpression string, task func()) error {
	c.logger.Infof("Scheduling task with cron expression [%s]", cronExpression)
	_, err := c.cron.AddFunc(cronExpression, task)
	if err != nil {
		c.logger.Error("cron AddFunc failed", "expr", cronExpression, "err", err.Error())
		return err
	}

	c.cron.Start()
	return nil
}

// Close stops the cron scheduler and waits (bounded) for any running jobs to
// finish, satisfying io.Closer for graceful shutdown. Safe to call even if no
// task was ever scheduled (the scheduler simply isn't running).
func (c *CronScheduler) Close() error {
	ctx := c.cron.Stop()
	select {
	case <-ctx.Done():
	case <-time.After(10 * time.Second):
		c.logger.Warn("cron scheduler did not stop within 10s")
	}
	return nil
}
