package article

import (
	"GoBook/internal/repository"
	"GoBook/pkg/logger"
	"GoBook/pkg/saramax"
	"context"
	"time"

	"github.com/IBM/sarama"
)

// Consumer 消费者接口，App 启动时统一调用 Start() 拉起所有消费者
type Consumer interface {
	Start() error
}

// InteractiveReadEventConsumer 文章阅读事件消费者
// 负责从 Kafka 消费 "article_read" 主题的消息，更新文章阅读计数
type InteractiveReadEventConsumer struct {
	repo   repository.InteractiveRepository // 互动仓储，用于更新阅读量
	client sarama.Client                    // Kafka 客户端（用于创建 ConsumerGroup）
	l      logger.LoggerV1                  // 日志记录器
}

// NewInteractiveReadEventConsumer 创建阅读事件消费者
// 注意：依赖 sarama.Client 而不是 sarama.ConsumerGroup，
// 因为 ConsumerGroup 需要在 Start() 时按需创建，且重连时需要复用 Client
func NewInteractiveReadEventConsumer(
	repo repository.InteractiveRepository,
	client sarama.Client,
	l logger.LoggerV1) *InteractiveReadEventConsumer {
	return &InteractiveReadEventConsumer{
		repo:   repo,
		client: client,
		l:      l,
	}
}

// Start 启动消费者，开始监听 Kafka 主题
// 关键点：
//  1. 创建 ConsumerGroup，指定 groupID="interactive"
//     - 同一组内的消费者分摊消息（负载均衡）
//     - 不同组各自收到全量消息（广播）
//  2. 使用 goroutine 异步消费，Start() 本身立即返回，不阻塞主流程
//  3. Consume() 是阻塞调用，当 ctx.Done() 或出错时才退出
//  4. saramax.NewHandler[T] 是泛型 Handler，自动完成 JSON 反序列化
func (i *InteractiveReadEventConsumer) Start() error {
	cg, err := sarama.NewConsumerGroupFromClient("interactive", i.client)
	if err != nil {
		return err
	}
	go func() {
		er := cg.Consume(context.Background(),
			[]string{TopicReadEvent},                      // 监听的主题列表
			saramax.NewHandler[ReadEvent](i.l, i.Consume)) // 泛型 Handler，自动反序列化为 ReadEvent
		if er != nil {
			i.l.Error("退出消费", logger.Error(er))
		}
	}()
	return err
}

// Consume 处理单条消息的业务逻辑
// 由 saramax.Handler 在 ConsumeClaim 循环中调用，无需关心消息拉取和反序列化
// 返回 error 只会被 Handler 记录日志，不会阻止后续消息消费（已 MarkMessage）
func (i *InteractiveReadEventConsumer) Consume(
	msg *sarama.ConsumerMessage,
	event ReadEvent,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// 调用仓储层增加阅读计数（数据库 + Redis 缓存）
	return i.repo.IncrReadCnt(ctx, "article", event.ArticleId)
}
