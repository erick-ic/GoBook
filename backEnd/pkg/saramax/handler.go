// Package saramax 封装 sarama 消费者组的泛型 Handler
// 目的：将 JSON 反序列化 + 错误日志 + 消息标记的逻辑统一抽取，
// 业务层只需传入一个 func(msg, event) error 即可消费消息
package saramax

import (
	"GoBook/pkg/logger"
	"encoding/json"

	"github.com/IBM/sarama"
)

// Handler 泛型消费者组处理器
// T 是消息体反序列化的目标类型，编译期类型安全
type Handler[T any] struct {
	l  logger.LoggerV1                                  // 日志记录器
	fn func(msg *sarama.ConsumerMessage, event T) error // 业务处理函数
}

// NewHandler 创建泛型 Handler
// 用法：saramax.NewHandler[ReadEvent](logger, consumeFunc)
// T 在编译期确定，Handler 内部自动将 msg.Value 反序列化为 T
func NewHandler[T any](l logger.LoggerV1, fn func(msg *sarama.ConsumerMessage, event T) error) *Handler[T] {
	return &Handler[T]{l: l, fn: fn}
}

// Setup 会话建立时回调（sarama.ConsumerGroupHandler 接口要求）
// 可用于初始化资源，这里不需要
func (h *Handler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup 会话结束时回调（sarama.ConsumerGroupHandler 接口要求）
// 可用于释放资源，这里不需要
func (h *Handler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 核心消费循环，sarama 会在每个分区的 goroutine 中调用
// 流程：遍历消息通道 → JSON 反序列化 → 调用业务函数 → 标记消息已消费
// 注意：无论业务函数返回 error 与否，都会 MarkMessage，
// 这样单条消息处理失败不会阻塞后续消息（失败消息已被标记为已消费，不会重投）
func (h *Handler[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgs := claim.Messages()
	for msg := range msgs {
		// 1. 反序列化消息体
		var t T
		err := json.Unmarshal(msg.Value, &t)
		if err != nil {
			// 反序列化失败只记录日志，不中断消费循环
			h.l.Error("反序列消息体失败",
				logger.String("topic", msg.Topic),
				logger.Int64("partition", int64(msg.Partition)),
				logger.Int64("offset", int64(msg.Offset)),
				logger.Error(err))
			continue
		}
		// 2. 调用业务处理函数
		err = h.fn(msg, t)
		if err != nil {
			// 业务处理失败只记录日志，不中断消费循环
			h.l.Error("处理消息失败",
				logger.String("topic", msg.Topic),
				logger.Int64("partition", int64(msg.Partition)),
				logger.Int64("offset", int64(msg.Offset)),
				logger.Error(err))
		}
		// 3. 标记消息为已消费（提交 offset）
		// 即使处理失败也标记，避免消息积压；如需重试需引入死信队列
		session.MarkMessage(msg, "")
	}
	return nil
}
