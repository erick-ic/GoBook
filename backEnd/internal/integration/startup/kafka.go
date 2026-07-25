package startup

import (
	articleevent "GoBook/internal/events/article"

	"github.com/IBM/sarama"
)

// noopArticleProducer 供不验证 Kafka 的集成测试使用。
// 它保留 ArticleService 对 Producer 接口的依赖，但不会建立网络连接或发送消息。
type noopArticleProducer struct{}

var _ articleevent.Producer = noopArticleProducer{}

func (noopArticleProducer) ProduceReadEvent(articleevent.ReadEvent) error {
	return nil
}

// InitArticleProducer 为集成测试提供无副作用的文章阅读事件生产者。
func InitArticleProducer() articleevent.Producer {
	return noopArticleProducer{}
}

// InitSaramaClient 保留给需要验证真实 Kafka 链路的集成测试使用。
func InitSaramaClient() sarama.Client {
	scfg := sarama.NewConfig()
	scfg.Producer.Return.Successes = true
	client, err := sarama.NewClient([]string{"localhost:9094"}, scfg)
	if err != nil {
		panic(err)
	}
	return client
}

// InitSyncProducer 保留给需要验证真实 Kafka 链路的集成测试使用。
func InitSyncProducer(c sarama.Client) sarama.SyncProducer {
	p, err := sarama.NewSyncProducerFromClient(c)
	if err != nil {
		panic(err)
	}
	return p
}
