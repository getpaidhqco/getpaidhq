package cron

import (
	"fmt"

	"github.com/robfig/cron/v3"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
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
	c.logger.Infof("Scheduling task with cron expression [%s]", cronExpression)
	_, err := c.cron.AddFunc(cronExpression, task)
	if err != nil {
		fmt.Println("Error adding job:", err)
		return err
	}

	c.cron.Start()
	return nil
}
