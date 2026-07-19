package saramax

import (
	"GoBook/pkg/logger"
	"context"
	"encoding/json"
	"time"

	"github.com/IBM/sarama"
)

type BatchHandler[T any] struct {
	l             logger.LoggerV1
	fn            func(msgs []*sarama.ConsumerMessage, ts []T) error
	batchSize     int
	batchDuration time.Duration
}

func NewBatchHandler[T any](
	l logger.LoggerV1,
	fn func(msgs []*sarama.ConsumerMessage, ts []T) error,
	batchSize int,
	batchDuration time.Duration,
) *BatchHandler[T] {
	return &BatchHandler[T]{
		l:             l,
		fn:            fn,
		batchSize:     10,
		batchDuration: time.Second,
	}
}

func (b *BatchHandler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (b *BatchHandler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 批量消费
func (b *BatchHandler[T]) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	msgsCh := claim.Messages()

	// 批量大小：每次取 10 条消息并发处理
	msgs := make([]*sarama.ConsumerMessage, 0, b.batchSize)

	for {
		var lastMsg *sarama.ConsumerMessage

		// 每批设置1秒超时，超时则提前提交已处理的消息
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		ts := make([]T, 0, b.batchSize)
		for i := 0; i < b.batchSize; i++ {
			done := false
			select {
			case <-ctx.Done():
				// 超时：1秒内没凑够一批，直接提交已处理的消息
				done = true
			case m1, ok := <-msgsCh:
				if !ok {
					cancel()
					// channel 关闭，说明分区被释放（rebalance 或 ctx 取消），退出消费
					return nil
				}
				lastMsg = m1

				var t T
				err := json.Unmarshal(m1.Value, &t)
				if err != nil {
					b.l.Error(
						"反序列化失败！",
						logger.Error(err),
						logger.String("topic", m1.Topic),
						logger.Int64("offset", m1.Offset),
						logger.Int64("partition", int64(m1.Partition)))
					continue
				}

				msgs = append(msgs, m1)
				ts = append(ts, t)
			}
			if done {
				break
			}
		}
		cancel()

		if len(msgs) == 0 {
			continue
		}

		err := b.fn(msgs, ts)
		if err != nil {
			b.l.Error("业务批量消费失败！", logger.Error(err))
			//继续处理下一批
			continue
		}
		// 批量提交：只标记最后一条消息，Kafka 会自动提交该分区到此偏移量之前的所有消息
		// 这是 MarkMessage 的核心特性：偏移量是单调递增的，标记 N 就等于标记 [0, N]
		session.MarkMessage(lastMsg, "")

		//万无一失
		//for _, m := range msgs {
		//	session.MarkMessage(m, "")
		//}
	}
}
