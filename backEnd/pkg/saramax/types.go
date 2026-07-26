package saramax

// Consumer 表示可由应用启动流程统一拉起的消息消费者。
type Consumer interface {
	Start() error
}
