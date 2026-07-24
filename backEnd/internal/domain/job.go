package domain

import (
	"time"

	"github.com/robfig/cron/v3"
)

// Job 是调度层使用的运行时任务，不包含 status、version、next_time 等持久化细节。
// DAO 抢占成功后由 Repository 将数据库记录转换成该对象，再交给对应 Executor 执行。
type Job struct {
	Id   int64
	Name string

	// Expression 决定下一次触发时间；当前解析器支持包含秒字段的 Cron 表达式。
	Expression string
	// Executor 用于选择执行器，例如 local；Cfg 保存执行器需要的扩展配置。
	Executor string
	Cfg      string
	// CancelFunc 由 Service 在抢占成功后注入，用于停止心跳并把任务释放回等待状态。
	CancelFunc func()
}

// NextTime 根据当前时间计算下一次计划执行时间。
// 当前实现假设 Expression 在任务写入数据库前已经校验过，因此忽略解析错误。
func (j Job) NextTime() time.Time {
	c := cron.NewParser(cron.Second | cron.Minute | cron.Hour |
		cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	s, _ := c.Parse(j.Expression)
	return s.Next(time.Now())
}
