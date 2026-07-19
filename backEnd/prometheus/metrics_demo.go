// Package prometheus_demo Prometheus Go SDK 使用学习演示
//
// 本文件展示 Prometheus 四种核心指标类型的使用方法，以及 Vector（多维标签）用法。
//
// 四种指标类型速记：
//   - Counter: 计数器，只增不减（请求数、错误数、阅读量）
//   - Gauge:   仪表盘，可增可减（在线人数、内存占用、队列长度）
//   - Histogram: 直方图，分桶统计（响应时间分布、请求体大小分布）
//   - Summary:   摘要，分位数统计（P50/P95/P99 响应时间）
//
// 安装依赖：
//
//	go get github.com/prometheus/client_golang/prometheus@latest
//	go get github.com/prometheus/client_golang/prometheus/promhttp@latest
//
// 命名约定：namespace_subsystem_name，三部分用下划线连接，
// 目的是通过指标名就能快速定位到具体业务，避免重名。
package prometheus_demo

import (
	"github.com/prometheus/client_golang/prometheus"
)

// ============================================================
// 一、Counter（计数器）
// ============================================================
//
// 特点：只增不减，重启后归零
// 适用场景：请求总数、错误次数、阅读量、点击量
// 核心方法：Inc() +1，Add(float64) 增加指定值（必须为正数）
//
// 命名三段式说明：
//   Namespace: 命名空间，通常代表部门/小组，如 "gobook"
//   Subsystem: 子系统，通常代表模块/服务，如 "article"
//   Name:      指标名，具体采集的数据，如 "read_total"
//   最终指标名：gobook_article_read_total（三段自动拼接）
// ============================================================

func CounterDemo() {
	// NewCounter 创建一个 Counter 指标
	// CounterOpts 配置指标的元信息
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "gobook",     // 命名空间：项目/部门级标识
		Subsystem: "article",    // 子系统：模块/服务级标识
		Name:      "read_total", // 指标名：具体采集的数据（_total 是 Counter 的命名惯例）
		Help:      "文章阅读总次数",    // 帮助文本，描述指标含义，Prometheus UI 中会显示
	})

	// MustRegister 注册指标到 Prometheus 默认 Registry
	// MustXXX 系列函数：注册失败会 panic（适用于启动时初始化，失败就不能启动）
	// Register 系列函数：注册失败返回 error（适用于动态注册的场景）
	prometheus.MustRegister(counter)

	// Inc() +1，最常用的操作
	counter.Inc()

	// Add(float64) 增加指定值，参数必须是正数
	// 如果传负数会 panic（Counter 语义上只增不减）
	counter.Add(10)
	counter.Add(2.5)
}

// ============================================================
// 二、Gauge（仪表盘）
// ============================================================
//
// 特点：可增可减，反映当前瞬时状态
// 适用场景：在线用户数、内存占用、CPU使用率、队列长度、goroutine 数
// 核心方法：Set(float64) 设置值，Add(float64) 增加（可负），Sub(float64) 减少
// ============================================================

func GaugeDemo() {
	// NewGauge 创建一个 Gauge 指标
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "gobook",
		Subsystem: "article",
		Name:      "online_users", // 指标名：在线用户数（Gauge 没有 _total 后缀）
		Help:      "当前在线用户数",
	})

	prometheus.MustRegister(gauge)

	// Set(float64) 直接设置值
	// 用于已知确切数值的场景，如内存用量、连接数
	gauge.Set(100)
	gauge.Set(150)

	// Add(float64) 增加值，可以传负数表示减少
	gauge.Add(10) // 150 + 10 = 160
	gauge.Add(-5) // 160 - 5 = 155

	// Sub(float64) 减少值，等价于 Add(-x)
	gauge.Sub(3) // 155 - 3 = 152
}

// ============================================================
// 三、Histogram（直方图）
// ============================================================
//
// 特点：将观测值分到不同桶（bucket）中，统计每个桶的累计数量
// 适用场景：响应时间分布、请求体大小分布
// 核心方法：Observe(float64) 观测一个值，自动落入对应桶
//
// Bucket（桶）说明：
//   - 每个桶是一个上限值（le = less than or equal）
//   - 桶是累计的：le=50 的桶包含 le=10 的桶的所有数据
//   - 例如 Buckets: [10, 50, 100] 表示：
//       <=10ms  的请求数 → 第一个桶
//       <=50ms  的请求数 → 第二个桶（含 <=10 的）
//       <=100ms 的请求数 → 第三个桶（含 <=50 的）
//
// 分桶原则：
//   - 不一定等间距，应根据业务分布灵活设置
//   - 目标：让数据尽量均匀分布在各个桶，不要都挤在一个桶里
//   - 响应时间通常是长尾分布，桶的间距应该越来越大（如指数分布）
// ============================================================

func HistogramDemo() {
	// NewHistogram 创建一个 Histogram 指标
	hist := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "gobook",
		Subsystem: "http",
		Name:      "request_duration_seconds", // 惯例：_duration_seconds 表示以秒为单位的耗时
		Help:      "HTTP 请求响应时间分布（秒）",

		// Buckets 定义分桶的上限值
		// 这是一个非等间距的例子，适合响应时间（前端快、后端慢、长尾分布）
		// 0.01s = 10ms, 0.05s = 50ms, 0.1s = 100ms, 0.5s = 500ms, 1s, 5s, 10s
		Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5, 10},
	})

	prometheus.MustRegister(hist)

	// Observe(float64) 观测一个值
	// Prometheus 会自动把这个值累加到所有 >= 该值的桶中
	// 例如 Observe(0.12) 会累加到 0.5、1、5、10 这四个桶
	hist.Observe(0.02) // 落入 <=0.05
	hist.Observe(0.15) // 落入 <=0.5
	hist.Observe(2.5)  // 落入 <=5
	hist.Observe(12.3) // 超过最大桶，会进入 +Inf 桶（Prometheus 自动添加）
}

// ============================================================
// 四、Summary（摘要）
// ============================================================
//
// 特点：直接计算分位数（P50/P90/P99等），在客户端计算
// 适用场景：需要精确分位数、不想在服务端计算的场景
// 核心方法：Observe(float64) 观测一个值
//
// Objectives（分位数配置）说明：
//   - key:   分位数（0~1 之间），0.5 表示 P50，0.99 表示 P99
//   - value: 允许的误差范围，0.01 表示 ±1%
//   - 例如 0.9: 0.005 表示：90分位数的误差在 ±0.5% 以内
//
// Histogram vs Summary 对比：
//   | 维度        | Histogram                | Summary                    |
//   |-------------|--------------------------|----------------------------|
//   | 计算位置    | 服务端（Prometheus）     | 客户端（应用）             |
//   | 资源消耗    | 低（只计数）             | 高（实时计算分位数）       |
//   | 聚合能力    | 可跨实例聚合             | 不可跨实例聚合             |
//   | 精度        | 取决于桶的划分           | 取决于配置的误差           |
//   | 适用场景    | 大多数场景（推荐）       | 需要精确分位数的特殊场景   |
// ============================================================

func SummaryDemo() {
	// NewSummary 创建一个 Summary 指标
	s := prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: "gobook",
		Subsystem: "http",
		Name:      "request_latency_seconds",
		Help:      "HTTP 请求延迟分位数（秒）",

		// Objectives 配置分位数和对应的误差
		// key = 分位数，value = 允许的相对误差
		//
		// 0.5:  0.01    → P50，误差 ±1%（50% 请求的响应时间）
		// 0.75: 0.01    → P75，误差 ±1%（75% 请求的响应时间）
		// 0.9:  0.005   → P90，误差 ±0.5%（90% 请求的响应时间）
		// 0.98: 0.002   → P98，误差 ±0.2%
		// 0.99: 0.001   → P99，误差 ±0.1%
		// 0.999: 0.0001 → P999，误差 ±0.01%
		//
		// 分位数越高，配置的误差越小（因为高端尾延迟更受关注）
		// 注意：误差越小，内存和CPU消耗越大
		Objectives: map[float64]float64{
			0.5:   0.01,
			0.75:  0.01,
			0.9:   0.005,
			0.98:  0.002,
			0.99:  0.001,
			0.999: 0.0001,
		},
	})

	prometheus.MustRegister(s)

	// Observe(float64) 观测一个值
	// Summary 内部会维护滑动窗口，实时计算各分位数
	s.Observe(0.012) // 12ms
	s.Observe(0.025) // 25ms
	s.Observe(0.15)  // 150ms
	s.Observe(1.2)   // 1.2s
	s.Observe(12.3)  // 12.3s（长尾请求）
}

// ============================================================
// 五、Vector（多维标签指标）
// ============================================================
//
// 特点：在基础指标上增加动态标签，按标签维度拆分统计
// 适用场景：按 HTTP 状态码、接口路径、HTTP 方法等维度统计
//
// 两类 Label：
//   - 固定标签（ConstLabels）：所有观测值都一样的标签，如 server、env、appname
//   - 动态标签：每个观测值可能不同的标签，如 pattern、method、status
//
// 四种 Vector 对应四种基础指标：
//   CounterVec、GaugeVec、HistogramVec、SummaryVec
//
// 使用方式：
//   1. 创建时指定动态标签名（如 []string{"pattern", "method", "status"}）
//   2. 观测时用 WithLabelValues(...) 传入具体标签值，按顺序对应
//   3. 每一组不同的标签值组合，都是一条独立的时间序列
//
// 注意：动态标签的基数（不同取值的数量）不能太大！
//   - 好的标签：method（GET/POST/PUT/DELETE，4种）
//   - 好的标签：status（200/400/401/500，少量几种）
//   - 差的标签：user_id（可能几百万种，会打爆 Prometheus）
// ============================================================

func VectorDemo() {
	// NewSummaryVec 创建一个带标签的 Summary 指标
	// 第二个参数 []string{"pattern", "method", "status"} 是动态标签名列表
	summaryVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "geekbang",
		Subsystem: "http_request",
		Name:      "duration_seconds",
		Help:      "HTTP 请求响应时间，按路径、方法、状态码拆分",

		// ConstLabels 固定标签，所有指标都带这些标签
		// 通常用于标识环境、机器、应用名等静态信息
		ConstLabels: map[string]string{
			"server":  "localhost:9091",
			"env":     "test",
			"appname": "test_app",
		},
	}, []string{"pattern", "method", "status"}) // 动态标签名列表

	prometheus.MustRegister(summaryVec)

	// WithLabelValues(...) 传入动态标签的具体值，按顺序对应标签名列表
	// 顺序必须一致：pattern → method → status
	//
	// 这行表示：请求 /user/:id（POST，200），响应时间 128ms
	// Prometheus 会为 {pattern="/user/:id", method="POST", status="200"}
	// 这一组标签组合维护一条独立的时间序列
	summaryVec.WithLabelValues("/user/:id", "POST", "200").Observe(0.128)

	// 另一组标签：GET /articles，200，响应时间 45ms
	summaryVec.WithLabelValues("/articles", "GET", "200").Observe(0.045)

	// 另一组标签：GET /pub/:id，200，响应时间 67ms
	summaryVec.WithLabelValues("/pub/:id", "GET", "200").Observe(0.067)

	// 另一组标签：POST /articles/publish，500，响应时间 230ms
	summaryVec.WithLabelValues("/articles/publish", "POST", "500").Observe(0.23)
}

// ============================================================
// 总结：四种指标类型选择指南
// ============================================================
//
// Q: 我要统计"总请求数"，用什么？
// A: Counter。只增不减，Inc() 每次请求 +1。
//
// Q: 我要统计"当前在线人数"，用什么？
// A: Gauge。用户登录 +1，登出 -1，反映瞬时状态。
//
// Q: 我要统计"接口响应时间的 P99"，用什么？
// A: Histogram（推荐）或 Summary。
//    - Histogram：在 Prometheus 端计算分位数，可跨实例聚合
//    - Summary：在客户端计算分位数，更精确但不可聚合
//
// Q: 我要按"状态码"分开统计成功和失败的请求数，用什么？
// A: CounterVec。带 status 标签，成功标 "200"，失败标 "500"。
//
// Q: 标签可以随便加吗？
// A: 不行。标签基数（不同值的数量）太大会导致时间序列爆炸，
//    Prometheus 存储和查询压力剧增。一般标签基数控制在几十以内。
//    反例：不要把 user_id、request_id 这种高基数字段当标签。
// ============================================================
