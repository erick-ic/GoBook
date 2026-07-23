package job

// Job 屏蔽具体调度框架，只描述一个可命名、可执行的后台任务。
type Job interface {
	Name() string
	Run() error
}
