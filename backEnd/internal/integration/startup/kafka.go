package startup

import articleevent "GoBook/internal/events/article"

// noopArticleProducer 隔离集成测试与 Kafka，文章查询产生的阅读事件会被安全丢弃。
type noopArticleProducer struct{}

// ProduceReadEvent 实现 article.Producer；测试不验证事件投递，因此直接返回成功。
func (noopArticleProducer) ProduceReadEvent(articleevent.ReadEvent) error {
	return nil
}

// InitArticleProducer 为集成测试提供无外部网络副作用的文章事件生产者。
func InitArticleProducer() articleevent.Producer {
	return noopArticleProducer{}
}
