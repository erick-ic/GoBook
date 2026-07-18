package sarama

import (
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Kafka 集群地址（单个 broker，实际生产环境通常是多个）
var addr = []string{"127.0.0.1:9094"}

// TestSyncProducer 同步发送
func TestSyncProducer(test *testing.T) {
	// 1. 创建 Sarama 配置对象
	cfg := sarama.NewConfig()
	//client, err := sarama.NewClient(addr, cfg)
	//producer, err := sarama.NewSyncProducerFromClient(client)

	// 2. 设置生产者参数
	// Return.Successes 必须为 true，同步生产者需要等待成功响应
	cfg.Producer.Return.Successes = true

	// 指定分区策略：使用 HashPartitioner，根据 Key 的哈希值选择分区
	// 这样可以确保相同 Key 的消息进入同一分区，从而保证消息顺序
	cfg.Producer.Partitioner = sarama.NewHashPartitioner

	// 3. 创建同步生产者（SyncProducer）
	// 参数：broker 地址列表、配置
	// 返回：生产者实例、错误
	producer, err := sarama.NewSyncProducer(addr, cfg)
	// 断言没有错误，如果有错则测试失败
	assert.NoError(test, err)

	// 4. 发送同步消息
	// SendMessage 是同步方法，会阻塞直到 broker 确认（或超时）
	// 返回：分区号、偏移量、错误
	_, _, err = producer.SendMessage(&sarama.ProducerMessage{
		// 主题名称，必须已在 Kafka 中创建
		Topic: "topic001",

		// Key：用于分区选择（HashPartitioner 会计算其哈希值）
		// 这里用字符串 "oid-123"，保证相同订单 ID 的消息进入同一分区，实现顺序消费
		Key: sarama.StringEncoder("oid-123"),

		// Value：消息体（实际业务数据）
		Value: sarama.StringEncoder("hello this is sync message~"),

		// Headers：消息头，用于传递额外的键值对元数据（类似 HTTP 头部）
		// 常见用途：传递 trace_id、user_id 等上下文信息，用于链路追踪
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("trace_id"),
				Value: []byte("123456"),
			},
		},

		// Metadata：开发者自定义的元数据，不发送到 Kafka，仅客户端本地使用
		// 可用于记录一些调试信息或业务标识，不影响消息内容
		Metadata: "metadata",
	})

	// 如果发送出错，直接返回（测试用例失败会由上方断言捕获，这里简单处理）
	if err != nil {
		return
	}
	// 注：通常应使用 assert.NoError(test, err) 来断言发送成功
	// 这里返回后测试结束，若 err 不为 nil 则测试会通过（但未断言），不推荐。
	// 建议改为：assert.NoError(test, err)
	assert.NoError(test, err)
}

// 核心区别：
//   - SyncProducer: 调用 SendMessage() 同步等待返回，每条消息都阻塞
//   - AsyncProducer: 消息写入 Input() channel 后立即返回，发送结果异步从 Successes()/Errors() 获取

// TestAsyncProducer 异步发送
// 异步生产者通过 channel 发送消息，不阻塞等待 broker 确认，
// 吞吐量高于同步生产者，适合高并发场景。
func TestAsyncProducer(t *testing.T) {
	// 1. 创建 Sarama 配置
	cfg := sarama.NewConfig()
	// 异步生产者必须显式开启成功和错误回调通道
	// 如果不开 Successes，Successes() channel 不会产生数据（也不会阻塞）
	// 如果不开 Errors，发送失败时错误会被丢弃（静默失败）
	cfg.Producer.Return.Successes = true
	cfg.Producer.Return.Errors = true

	// 2. 创建异步生产者
	// NewAsyncProducer 内部会启动后台 goroutine 负责将消息批量发送到 Kafka
	producer, err := sarama.NewAsyncProducer(addr, cfg)
	// require.NoError 会在出错时直接 Fatal 终止测试（panic 机制）
	// 对比 assert.NoError 只是标记失败但继续执行，这里生产者创建失败后续无意义，所以用 require
	require.NoError(t, err)

	// 3. 获取输入 channel，所有消息通过这个 channel 投递给生产者
	// 生产者内部 goroutine 会从该 channel 消费消息并发送到 Kafka
	msgChan := producer.Input()

	// 4. 构造消息
	msg := &sarama.ProducerMessage{
		// 主题：消息发往哪个 topic
		Topic: "topic001",
		// Key：用于分区路由，HashPartitioner 会根据 Key 哈希值选择分区
		// 相同 Key 永远进同一分区，保证同一业务实体（如同一订单）的消息顺序消费
		Key: sarama.StringEncoder("oid-456"),
		// Value：消息体，实际业务数据
		Value: sarama.StringEncoder("hello this is async message~"),
		// Headers：消息头元数据，类似 HTTP Header
		// 常用于传递 trace_id 实现链路追踪，不影响消息分区和存储
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("trace_id"),
				Value: []byte("654321"),
			},
		},
		// Metadata：客户端本地元数据，不会发送到 Kafka
		// 当消息从 Successes() channel 返回时会带上，可用于关联请求
		Metadata: "metadata",
	}

	// 5. 将消息投递到 Input channel
	// 这一步是非阻塞的（channel 有缓冲区），消息投递后立即继续执行
	// 实际的网络发送由生产者内部 goroutine 异步完成
	msgChan <- msg

	// 6. 获取错误和成功回调 channel
	// 两个 channel 都必须在配置中开启 Return.Successes/Errors 才有数据
	errChan := producer.Errors()
	successChan := producer.Successes()

	// 7. 通过 select 监听发送结果
	// 注意：这里的 for 循环会一直阻塞，适合长运行的服务
	// 在测试中可以用 break 或设置超时来退出
	for {
		select {
		case err = <-errChan:
			// 发送失败：broker 返回错误（如 leader 不可用、超时等）
			// err 是 *ProducerError 类型，包含原始消息和错误信息
			t.Log("发送失败：", err)
		case msg = <-successChan:
			// 发送成功：broker 已确认写入
			// msg 包含分区号(partition)和偏移量(offset)，可用于追踪
			t.Log("发送成功：", msg)
		}
	}
}
