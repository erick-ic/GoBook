package article

import (
	"encoding/json"

	"github.com/IBM/sarama"
)

// TopicReadEvent Kafka 主题名称，用于文章阅读事件
const TopicReadEvent = "article_read"

// Producer 消息生产者接口，定义业务层使用的发消息方法
// 业务层只依赖接口，不直接依赖 sarama，方便测试 mock
type Producer interface {
	ProduceReadEvent(evt ReadEvent) error
}

// KafkaProducer 基于 sarama.SyncProducer 的 Kafka 生产者实现
type KafkaProducer struct {
	producer sarama.SyncProducer
}

// NewKafkaProducer 创建 Kafka 生产者
// 注意：返回值是 Producer 接口（不是 *KafkaProducer）
func NewKafkaProducer(producer sarama.SyncProducer) Producer {
	return &KafkaProducer{
		producer: producer,
	}
}

// ProduceReadEvent 发送一条文章阅读事件到 Kafka
// 流程：结构体 → JSON 序列化 → 发送到 topic "article_read"
func (k *KafkaProducer) ProduceReadEvent(evt ReadEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, _, err = k.producer.SendMessage(&sarama.ProducerMessage{
		Topic: TopicReadEvent,           // 目标主题
		Value: sarama.ByteEncoder(data), // 消息体（JSON 编码后的字节数组）
	})
	return err
}

// ReadEvent 文章阅读事件结构体，作为 Kafka 消息的 payload
type ReadEvent struct {
	Uid       int64 // 触发阅读的用户ID
	ArticleId int64 // 被阅读的文章ID
}
