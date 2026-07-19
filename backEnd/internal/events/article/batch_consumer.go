package article

import (
	"GoBook/internal/repository"
	"GoBook/pkg/logger"
	"GoBook/pkg/saramax"
	"context"
	"time"

	"github.com/IBM/sarama"
)

type InteractiveReadEventBatchConsumer struct {
	repo   repository.InteractiveRepository // 互动仓储，用于更新阅读量
	client sarama.Client                    // Kafka 客户端（用于创建 ConsumerGroup）
	l      logger.LoggerV1                  // 日志记录器
}

func NewInteractiveReadEventBatchConsumer(
	repo repository.InteractiveRepository,
	client sarama.Client,
	l logger.LoggerV1) *InteractiveReadEventBatchConsumer {
	return &InteractiveReadEventBatchConsumer{
		repo:   repo,
		client: client,
		l:      l,
	}
}
func (i *InteractiveReadEventBatchConsumer) Start() error {
	cg, err := sarama.NewConsumerGroupFromClient("interactive", i.client)
	if err != nil {
		return err
	}
	go func() {
		er := cg.Consume(context.Background(),
			[]string{TopicReadEvent},
			saramax.NewBatchHandler[ReadEvent](i.l, i.Consume, 10, time.Second))
		if er != nil {
			i.l.Error("退出消费", logger.Error(er))
		}
	}()
	return err
}

func (i *InteractiveReadEventBatchConsumer) Consume(
	msgs []*sarama.ConsumerMessage,
	ts []ReadEvent,
) error {
	bizs := make([]string, 0, len(msgs))
	ids := make([]int64, 0, len(ts))

	for _, evt := range ts {
		bizs = append(bizs, "article")
		ids = append(ids, evt.ArticleId)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := i.repo.BatchIncrReadCnt(ctx, bizs, ids)
	if err != nil {
		i.l.Error(
			"批量增加阅读计数失败！",
			logger.Error(err),
			logger.Field{Key: "ids", Value: ids})
		return err
	}
	return nil
}
