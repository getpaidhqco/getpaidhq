package cron

import (
	"github.com/robfig/cron/v3"
	"payloop/internal/core/port"
	"payloop/internal/lib"
)

type CronScheduler struct {
	cron   *cron.Cron
	logger port.Logger
}

func NewCronScheduler(logger port.Logger, env lib.Env) port.Scheduler {
	c := cron.New()
	return &CronScheduler{
		logger: logger,
		cron:   c,
	}
}

func (c *CronScheduler) ScheduleTask(cronExpression string, task func()) error {
	c.logger.Info("scheduling task", "cronExpression", cronExpression)
	_, err := c.cron.AddFunc(cronExpression, task)
	if err != nil {
		c.logger.Error("error adding job", "error", err)
		return err
	}

	c.cron.Start()
	return nil
}
