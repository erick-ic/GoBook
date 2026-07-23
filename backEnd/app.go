package main

import (
	"GoBook/internal/events/article"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

type App struct {
	Server *gin.Engine
	//消费者类似web服务器，因此引入App结构体
	Consumers []article.Consumer
	// Cron 与 HTTP 服务共享应用生命周期，由 main 统一启动和优雅停止。
	Cron *cron.Cron
}
