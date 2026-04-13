package port

// Scheduler runs tasks on cron-like schedules.
type Scheduler interface {
	ScheduleTask(cronExpression string, task func()) error
}
