package ioc

import (
	"GoBook/interactive/events"
	"GoBook/pkg/saramax"

	"github.com/IBM/sarama"
	"github.com/spf13/viper"
)

// InitSaramaClient 根据 kafka.addr 创建可复用的 Kafka 底层客户端。
func InitSaramaClient() sarama.Client {
	type Config struct {
		Addr []string `mapstructure:"addr"`
	}
	var cfg Config
	if err := viper.UnmarshalKey("kafka", &cfg); err != nil {
		panic(err)
	}
	if len(cfg.Addr) == 0 {
		panic("interactive: kafka.addr 不能为空")
	}

	scfg := sarama.NewConfig()
	// 当前客户端同时可被同步生产者复用，因此开启成功回执。
	scfg.Producer.Return.Successes = true
	client, err := sarama.NewClient(cfg.Addr, scfg)
	if err != nil {
		panic(err)
	}
	return client
}

// InitConsumers 将所有消费者汇总成一个切片，供 App 启动时统一调用 Start()
// 新增消费者时只需扩展参数和返回切片，Wire 会继续完成依赖注入。
func InitConsumers(c1 *events.InteractiveReadEventConsumer) []saramax.Consumer {
	return []saramax.Consumer{c1}
}
