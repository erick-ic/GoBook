package web

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ObserverAbilityHandler 可观测性测试接口
// 提供一个专门用于测试 Prometheus 指标采集的接口，模拟随机响应时间
type ObserverAbilityHandler struct{}

func NewObserverAbilityHandler() *ObserverAbilityHandler {
	return &ObserverAbilityHandler{}
}

// RegisterRouters 注册可观测性测试路由
// /test/metrics 接口随机 sleep 0~1秒，用于验证 Prometheus 中间件能否正确采集响应时间分布
func (o *ObserverAbilityHandler) RegisterRouters(server *gin.Engine) {
	group := server.Group("/test")
	group.GET("/metrics", func(ctx *gin.Context) {
		// 随机产生 0~1000 毫秒的延迟，模拟真实接口的耗时波动
		// 多次请求后，Prometheus 中能看到 P50/P90/P99 等分位数的变化
		sleep := rand.Int31n(1000)
		time.Sleep(time.Duration(sleep) * time.Millisecond)
		ctx.String(http.StatusOK, "OK~")
	})
}
