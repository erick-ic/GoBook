package logger

import "sync"

// 私有的，真正被使用的实例
var gl LoggerV1

// 保护 gl 的读写锁
var lMutex sync.RWMutex

// SetGlobalLogger 线程安全地替换全局日志实例（通常在应用启动时调用）
func SetGlobalLogger(l LoggerV1) {
	// 写锁（独占锁）
	lMutex.Lock()
	defer lMutex.Unlock()
	gl = l
	/*
		动作：替换全局变量 gl 的内存指针。

		规则：在替换的这一瞬间，绝对不允许有任何代码在读取 gl（否则可能读到半个指针或被替换掉的旧内存），也不允许有其他代码在同时执行替换。

		并发表现：如果此时有 1000 个请求正在执行 logger.L()（读锁），这个 SetGlobalLogger 会阻塞等待，直到所有读锁释放，
			然后独占地把 gl 换掉。这就是“写锁”的独占性。
	*/
}

// L 线程安全地获取当前全局日志实例
func L() LoggerV1 {
	//读锁（共享锁）
	lMutex.RLock()
	g := gl
	lMutex.RUnlock()
	return g
	/*
		动作：仅仅是把 gl 当前的指针值复制一份给局部变量 g。

		规则：这种纯读取操作不会破坏数据，所以允许多个协程并发执行。

		并发表现：如果有 1000 个请求同时调用 L()，它们都会瞬间拿到读锁并返回，彼此不阻塞。
			只有当 SetGlobalLogger 想拿写锁时，才会被这 1000 个读锁挡在外面。
	*/
}

// GL 包级全局变量，默认 NopLogger（空实现），避免空指针。
// 公有的，硬编码空实现
// 用于测试或默认行为，确保日志调用不会产生副作用
var GL LoggerV1 = &NopLogger{}
