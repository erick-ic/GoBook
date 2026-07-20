package context

import (
	"context"
	"testing"
	"time"
)

// Key1 推荐的自定义 key 类型
// 用 struct{} 作为 key 类型可以避免和其他包的字符串 key 冲突
// 因为不同包定义的同名字符串 key 是同一个 key，而不同类型的 struct{} 永远不会冲突
type Key1 struct{}

// TestContext 演示 context.WithValue 的基本用法和值覆盖行为
func TestContext(t *testing.T) {
	//推荐写法：用自定义类型作 key，避免冲突
	//ctx := context.WithValue(context.Background(), Key1{}, "value1")
	//val := ctx.Value(Key1{})
	//t.Log(val)

	// 用字符串作 key（不推荐：容易和其他包的字符串 key 冲突）
	ctx := context.WithValue(context.Background(), "key1", "value1")
	val := ctx.Value("key1")
	t.Log(val)
	//输出：value1

	// ⚠️ 这里是重新创建了 context，而不是修改原 context
	// WithValue 返回的是新 context，原 ctx 没被修改，只是这里用 ctx 变量接收了新值
	// 所以"原值被覆盖"的说法不准确，本质是 ctx 指向了新的 context 链
	ctx = context.WithValue(context.Background(), "key1", "value1-1")
	val = ctx.Value("key1")
	t.Log(val)
	//输出：value1-1

	// 在已有 ctx 的基础上派生子 context，追加新 key
	// 此时 ctx 链：Background → key1=value1-1 → key2=value2
	ctx = context.WithValue(ctx, "key2", "value2")
	val = ctx.Value("key2")
	t.Log(val)
	//输出：value2
}

// TestContextCancle 演示 WithCancel 的基本用法
func TestContextCancle(t *testing.T) {
	ctx, cancle := context.WithCancel(context.Background())
	//防止 goroutine 泄漏：函数退出时调用 cancel，确保下面那个监听 Done 的 goroutine 能退出
	defer cancle()
	//防止有人使用了 Done，在等待 ctx 结束信号
	//启动一个 goroutine 监听 ctx.Done()，如果不 cancel 它会永远阻塞，导致 goroutine 泄漏
	go func() {
		ch := ctx.Done()
		<-ch
	}()

	ctx = context.WithValue(ctx, "key1", "value1")
	val := ctx.Value("key1")
	t.Log(val)
	//输出：value1
}

// TestContextTimeout 演示 WithTimeout 返回的 cancel 函数必须调用
// ⚠️ 这段代码有 bug：没有接收 cancel 函数，导致无法取消，会造成 context 泄漏
func TestContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ctx = context.WithValue(ctx, "key1", "value1")
	val := ctx.Value("key1")
	t.Log(val)
}

// TestContextErr 演示如何通过 ctx.Err() 判断 context 的结束原因
func TestContextErr(t *testing.T) {
	ctx, cancle := context.WithTimeout(context.Background(), time.Second)
	defer cancle()
	ctx.Err()
	//如何区分被取消和超时？
	// - context.Canceled:         主动调用 cancel() 导致取消
	// - context.DeadlineExceeded: 超时自动取消
	// - nil:                      context 还活着，没结束
	if ctx.Err() == context.Canceled {
		t.Log("context is canceled")
	} else if ctx.Err() == context.DeadlineExceeded {
		t.Log("context is deadline exceeded")
	}
	//注意：这里立即调用 ctx.Err() 会返回 nil（因为还没到 1 秒，也没 cancel）
	//所以上面两个分支都不会命中。要看到效果需要在 sleep 后或 ctx.Done() 后再调用
}

// TestSubContext 演示 context 树的传播规则
/*
1. 控制是从上至下的：父级取消或者超时，所有派生的子级都被取消或者超时。
2. 查找是从下至上的：当找 key 的时候，子 context 先看自己有没有，没有则去祖先里面找。
*/
func TestSubContext(t *testing.T) {
	// 记录起始时间，用于后续计算耗时
	start := time.Now()

	// 父 context：1秒后自动超时（也会触发子 context 结束）
	ctx, cancel0 := context.WithTimeout(context.Background(), time.Second)

	// 子 context：从父 context 派生，自身可独立 cancel，但父结束它也会被结束
	// 这里故意不调用 subCancel，依赖父级超时链式触发
	subCtx, _ := context.WithCancel(ctx)

	// goroutine A：1秒后主动 cancel 父 context
	// 与 WithTimeout 的超时时间相同，两者几乎同时触发（谁先到看调度）
	go func() {
		time.Sleep(time.Second)
		cancel0()
	}()

	// 用 channel 替代 goroutine 内的 t.Log
	// 原因：t.Log 在 goroutine 中调用可能输出顺序混乱或被吞，
	// 通过 channel 把信号回传到主 goroutine 输出更可靠
	done := make(chan struct{})

	// goroutine B：监听子 context 的结束信号
	// 验证"父级取消 → 子级立即收到信号"的链式传播规则
	go func() {
		<-subCtx.Done() // 阻塞等待子 context 结束（父 cancel 后这里会立刻返回）
		close(done)     // 通过 close 通知主 goroutine
	}()

	t.Log("等待结束信号...", time.Since(start)) // ≈ 0s，主流程几乎立即执行到这里
	<-done                                // 阻塞等待子 context 结束，约 1s 后返回
	t.Log("收到结束信号...", time.Since(start)) // ≈ 1s，父 cancel 后子立刻收到信号

	// sleep 10秒验证：cancel 后 goroutine 已退出，不会出现泄漏
	// （sleep 10秒主要是为了观察 goroutine B 的输出，这里保留以模拟长流程）
	time.Sleep(time.Second * 10)
	t.Log("程序结束...", time.Since(start)) // ≈ 11s
}
