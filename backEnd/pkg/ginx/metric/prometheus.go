package metric

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// MiddlewareMetricBuilder Prometheus 指标中间件构造器
// 用于生成一个 Gin 中间件，自动采集每个请求的响应时间，按 pattern/method/status 拆分
type MiddlewareMetricBuilder struct {
	NameSpace  string // 命名空间（项目/部门级）
	Subsystem  string // 子系统（模块级）
	Name       string // 指标名（会自动拼接 _resp_time 后缀）
	Help       string // 帮助文本
	InstanceId string // 实例ID，作为固定标签，用于区分不同实例
}

// NewMiddlewareMetricBuilder 创建构造器实例
func NewMiddlewareMetricBuilder(NameSpace, Subsystem, Name, Help, InstanceId string) *MiddlewareMetricBuilder {
	return &MiddlewareMetricBuilder{
		NameSpace:  NameSpace,
		Subsystem:  Subsystem,
		Name:       Name,
		Help:       Help,
		InstanceId: InstanceId,
	}
}

// BuildResponseTime 构建并返回 Gin 中间件函数
// 内部创建一个 SummaryVec 指标，注册到 Prometheus 默认 Registry
func (mb *MiddlewareMetricBuilder) BuildResponseTime() gin.HandlerFunc {
	// 动态标签：pattern（命中的路由模板）、method（HTTP方法）、status（响应状态码）
	// 这三个维度组合起来可以精准定位到"某个接口的某种方法的某种状态"的耗时
	labels := []string{"pattern", "method", "status"}

	// 创建带标签的 Summary 指标（响应时间分位数统计）
	summary := prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: mb.NameSpace,
			Subsystem: mb.Subsystem,
			Name:      mb.Name + "_resp_time", // 自动拼接后缀
			Help:      mb.Help,
			// ConstLabels 固定标签，所有指标都带，用于标识实例
			ConstLabels: map[string]string{
				"instance_id": mb.InstanceId,
			},
			// Objectives 分位数配置：key=分位数，value=允许误差
			// 分位数越高误差越小（尾延迟更受关注），但资源消耗越大
			Objectives: map[float64]float64{
				0.5:   0.01,
				0.75:  0.01,
				0.9:   0.005,
				0.98:  0.002,
				0.99:  0.001,
				0.999: 0.0001,
			},
		},
		labels,
	)
	prometheus.MustRegister(summary)

	// 返回实际的 Gin 中间件处理函数
	return func(c *gin.Context) {
		start := time.Now()

		// defer 在 c.Next() 之后执行，确保能拿到最终的 status
		defer func() {
			duration := time.Since(start)

			// c.FullPath() 返回命中的路由模板（如 /pub/:id），而非实际路径（如 /pub/123）
			// 用模板而非实际路径作为标签，避免高基数标签打爆 Prometheus
			pattern := c.FullPath()
			if pattern == "" {
				pattern = "unknown" // 未命中路由的请求（如 404）统一标记
			}

			// WithLabelValues 必须按 labels 定义顺序传值：pattern, method, status
			// Observe 记录响应时间（毫秒）
			summary.WithLabelValues(
				pattern,
				c.Request.Method,
				strconv.Itoa(c.Writer.Status()),
			).
				Observe(float64(duration.Milliseconds()))
		}()

		// c.Next() 执行后续 handler，defer 中的统计逻辑会在其返回后执行
		c.Next()
	}
}

// BuildActiveRequest 统计当前活跃请求数量（当前正在处理中的请求数）
// 使用 Gauge 指标，因为活跃数是可增可减的瞬时状态（请求开始 +1，结束 -1）
// 区别于 Build() 中的 Summary（统计响应时间分布），这里关注的是并发量
func (mb *MiddlewareMetricBuilder) BuildActiveRequest() gin.HandlerFunc {
	// 创建 Gauge 指标（可增可减，反映当前瞬时并发数）
	// 不需要动态标签：活跃请求数是全局指标，无需按 pattern/method/status 拆分
	// 如果加了标签，每条请求都会让对应标签组合的 Gauge +1/-1，
	// 反而失去了"全局并发量"的意义
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: mb.NameSpace,
		Subsystem: mb.Subsystem,
		Name:      mb.Name + "_active_request",
		Help:      mb.Help,
		ConstLabels: map[string]string{
			"instance_id": mb.InstanceId,
		},
	})
	prometheus.MustRegister(gauge)

	return func(c *gin.Context) {
		// 请求进入时 +1，表示多了一个正在处理的请求
		gauge.Inc()
		// defer 确保请求结束（无论成功失败、是否 panic）都会 -1
		// 必须用 defer 而非在函数末尾 Dec()，否则 panic 时会漏减导致计数只增不减
		defer gauge.Dec()
		// 执行后续中间件和业务 handler
		// c.Next() 返回后，上面的 defer 才会执行 Dec()
		c.Next()
	}
}
