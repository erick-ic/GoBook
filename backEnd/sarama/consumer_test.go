package sarama

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// 使用 ConsumerGroup + Handler 模式消费消息
// ConsumerGroup 三大核心方法：
//   - Setup():    分区分配后调用，可用于重置偏移量、初始化资源
//   - Cleanup():  分区释放前调用，可用于清理资源
//   - ConsumeClaim(): 实际消费逻辑，从 claim.Messages() 读取消息

// TestConsumer 同步消费者组
func TestConsumer(t *testing.T) {
	// 1. 创建 Sarama 配置
	cfg := sarama.NewConfig()

	// 2. 创建消费者组
	// 参数2 "test_group" 是 groupID，同一 groupID 的消费者共享消费进度（负载均衡）
	// 不同 groupID 的消费者各自独立消费（发布-订阅模式）
	consumer, err := sarama.NewConsumerGroup(addr, "test_group", cfg)
	require.NoError(t, err)

	start := time.Now()

	// 3. 创建带取消的 context，控制消费时长
	// 方式一（已注释）：WithTimeout，超时自动取消
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	//defer cancel()
	// 方式二（当前）：WithCancel + AfterFunc，5秒后调用 cancel() 取消
	// 区别：AfterFunc 可以精确控制取消时机，WithTimeout 的超时从创建时开始算
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(time.Second*5, func() {
		cancel()
	})

	// 4. 开始消费
	// Consume 是阻塞方法，会在以下情况返回：
	//   - ctx 被取消（如超时）
	//   - 发生不可恢复的错误
	//   - rebalance 触发（分区重新分配时，内部会重新调用 Setup/Cleanup）
	// 参数2：要消费的 topic 列表
	// 参数3：实现了 ConsumerGroupHandler 接口的处理器
	err = consumer.Consume(
		ctx,
		[]string{TopicReadEvent},
		testConsumerGroupHandler{},
	)
	// 消费结束后打印耗时和错误
	t.Log("消费结束：", err, time.Since(start).String())
}

// testConsumerGroupHandler 实现 sarama.ConsumerGroupHandler 接口
// 该接口有三个方法：Setup → ConsumeClaim → Cleanup，构成消费生命周期
type testConsumerGroupHandler struct{}

// Setup 在分区分配给本消费者后调用（rebalance 时触发）
// 典型用途：重置偏移量、初始化数据库连接等资源
func (t testConsumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	// ResetOffset：将消费偏移量重置到最早（OffsetOldest），即从头开始消费
	// 等效于命令行：kafka-consumer-groups --reset-offsets --to-earliest
	// 生产环境中通常不会每次都从头消费，这里仅作演示
	partitions := session.Claims()[TopicReadEvent]
	for _, part := range partitions {
		session.ResetOffset(TopicReadEvent, part, sarama.OffsetOldest, "")
	}
	log.Println("testConsumerGroupHandler.Setup")
	return nil
}

// Cleanup 在分区从本消费者释放前调用（rebalance 时触发）
// 典型用途：关闭数据库连接、刷新缓冲区等清理操作
func (t testConsumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	log.Println("testConsumerGroupHandler.Cleanup")
	return nil
}

// ConsumeClaim 实际消费逻辑 —— 批量 + 并发消费模式（高性能版本）
// 每次从 channel 中取一批消息（bitchSize=10），用 errgroup 并行处理，
// 全部成功后一次性 MarkMessage 提交偏移量，减少提交次数提升吞吐量。
//
// session：代表与 Kafka 的会话，贯穿分区分配到释放的全生命周期
//   - session.MarkMessage() 标记消息已消费，rebalance 时会据此提交偏移量
//   - session.Claims() 获取分配给本消费者的分区
//
// claim：代表一个分区的消息流
//   - claim.Messages() 返回消息 channel，持续产生消息直到分区被释放
func (t testConsumerGroupHandler) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	msgs := claim.Messages()

	// 批量大小：每次取 10 条消息并发处理
	const bitchSize = 10

	for {
		var eg errgroup.Group
		var lastMsg *sarama.ConsumerMessage

		// 每批设置1秒超时，超时则提前提交已处理的消息
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)

		for i := 0; i < bitchSize; i++ {
			done := false
			select {
			case <-ctx.Done():
				// 超时：1秒内没凑够一批，直接提交已处理的消息
				done = true
			case m1, ok := <-msgs:
				if !ok {
					cancel()
					// channel 关闭，说明分区被释放（rebalance 或 ctx 取消），退出消费
					return nil
				}
				lastMsg = m1
				// 用 errgroup 启动并发处理
				// 如果业务逻辑很慢（如写数据库），多条消息可以并行执行
				eg.Go(func() error {
					// 模拟耗时业务处理（如写数据库）
					time.Sleep(time.Millisecond * 100)

					// 重试逻辑（已预留，未实现）
					// 生产环境中，消费失败通常需要重试或写入死信队列
					for i := 0; i < 3; i++ {
					}
					log.Println("m1.Value:", string(m1.Value))
					return nil
				})
			}
			if done {
				break
			}
		}

		// 等待本批所有消息处理完毕
		err := eg.Wait()
		if err != nil {
			// 有消费失败的，记日志但不提交偏移量（continue 后下一轮会重新消费这些消息）
			// 注意：这里 continue 后 msgs channel 里的消息已经读走了，
			// 实际生产中需要更精细的错误处理（如写入死信队列）
			continue
		}

		if ctx.Err() != nil {
			return nil
		}
		// 批量提交：只标记最后一条消息，Kafka 会自动提交该分区到此偏移量之前的所有消息
		// 这是 MarkMessage 的核心特性：偏移量是单调递增的，标记 N 就等于标记 [0, N]
		session.MarkMessage(lastMsg, "")
	}
}

// ConsumeClaimV1 实际消费逻辑 —— 逐条消费模式（简单版本）
// for range 遍历消息 channel，每条消息同步处理、立即提交。
// 优点：简单易懂，消息不会丢失
// 缺点：吞吐量低，单条处理单条提交，无法利用并发
//
// 与 ConsumeClaim 的区别：
//   - ConsumeClaimV1：逐条处理 → 逐条 MarkMessage（简单，低吞吐）
//   - ConsumeClaim：  批量并发处理 → 批量 MarkMessage（复杂，高吞吐）
//func (t testConsumerGroupHandler) ConsumeClaimV1(
//	// session：与 Kafka 的会话，用于 MarkMessage 提交偏移量
//	// 生命周期：从分区分配（Setup）到分区释放（Cleanup）
//	session sarama.ConsumerGroupSession,
//	claim sarama.ConsumerGroupClaim,
//) error {
//	msgs := claim.Messages()
//	// for range 遍历消息 channel
//	// channel 关闭时（分区被释放），range 自动退出
//	for msg := range msgs {
//		// 实际业务中通常需要反序列化消息体：
//		// var bizMsg MyBizMsg
//		// err := json.Unmarshal(msg.Value, &bizMsg)
//		// if err != nil {
//		//     // 反序列化失败：消息格式异常，记日志后 skip（不重试，避免阻塞后续消息）
//		//     continue
//		// }
//		println("msg.value:", string(msg.Value))
//
//		// MarkMessage：标记消息已成功消费
//		// 注意：MarkMessage 不是立即提交到 Kafka！
//		// 它只是在本地图录偏移量，真正提交发生在：
//		//   - session 结束时（rebalance 触发）
//		//   - Sarama 内部定时自动提交（cfg.Consumer.Offsets.AutoCommit.Interval）
//		session.MarkMessage(msg, "")
//	}
//	// msgs channel 关闭，分区被释放，退出消费
//	return nil
//
//}

type MyBizMsg struct {
	Name string
}

// 返回只读channel（优先使用）
func ChannelV1() <-chan struct{} {
	panic("implement me")
}

// 返回可读可写channel
func ChannelV2() chan struct{} {
	panic("implement me")
}

// 返回只写channel（极少使用）
func ChannelV3() chan<- struct{} {
	panic("implement me")
}
