package cron

import (
	"fmt"
	"github.com/robfig/cron"
	"payloop/internal/application/interfaces"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type CronScheduler struct {
	cron   *cron.Cron
	logger logger.Logger
}

func NewCronScheduler(logger logger.Logger, env lib.Env) interfaces.Scheduler {
	c := cron.New()
	return &CronScheduler{
		logger: logger,
		cron:   c,
	}
}

func (c *CronScheduler) ScheduleTask(cronExpression string, task func()) error {
	c.logger.Infof("Scheduling task with cron expression [%s]", cronExpression)
	err := c.cron.AddFunc(cronExpression, task)
	if err != nil {
		fmt.Println("Error adding job:", err)
		return err
	}

	c.cron.Start()
	return nil
}
