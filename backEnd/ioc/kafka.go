package ioc

import (
	"GoBook/internal/events/article"

	"github.com/IBM/sarama"
	"github.com/spf13/viper"
)

// InitSaramaClient 初始化 Kafka 客户端（sarama.Client）
// 一个 Client 是底层连接池，可以基于它创建 Producer 和 ConsumerGroup
// 配置从 viper 的 "kafka.addr" 读取，对应 dev.yaml 中的 kafka.addr
func InitSaramaClient() sarama.Client {
	type Config struct {
		Addr []string `yaml:"addr"`
	}
	var cfg Config
	err := viper.UnmarshalKey("kafka", &cfg)
	if err != nil {
		panic(err)
	}
	scfg := sarama.NewConfig()
	// 使用 SyncProducer（同步生产者）时必须开启，否则 SendMessage 会阻塞无返回
	scfg.Producer.Return.Successes = true
	client, err := sarama.NewClient(cfg.Addr, scfg)
	if err != nil {
		panic(err)
	}
	return client
}

// InitSyncProducer 基于 sarama.Client 创建同步生产者
// SyncProducer 会阻塞等待 Kafka 的 ACK 确认后才返回，保证消息发送成功
// 适合对可靠性要求高的场景；如果追求吞吐量可以用 AsyncProducer
func InitSyncProducer(c sarama.Client) sarama.SyncProducer {
	p, err := sarama.NewSyncProducerFromClient(c)
	if err != nil {
		panic(err)
	}
	return p
}

// InitConsumers 将所有消费者汇总成一个切片，供 App 启动时统一调用 Start()
// 新增消费者时，只需多加一个参数，Wire 自动注入
func InitConsumers(c1 *article.InteractiveReadEventConsumer) []article.Consumer {
	return []article.Consumer{c1}
}
