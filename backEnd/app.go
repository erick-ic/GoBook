package main

import (
	"GoBook/internal/events/article"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

type App struct {
	Server *gin.Engine
	//消费者类似web服务器，因此引入App结构体
	Consumers []article.Consumer
	// Cron 与 HTTP 服务共享应用生命周期，由 main 统一启动和优雅停止。
	Cron *cron.Cron
	// Closers 收集需要随应用退出释放的资源，例如排行榜任务持有的分布式锁。
	Closers []io.Closer
}
