package interfaces

type Scheduler interface {
	ScheduleTask(cronExpression string, task func()) error
}
