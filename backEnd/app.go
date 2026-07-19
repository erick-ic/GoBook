package main

import (
	"GoBook/internal/events/article"

	"github.com/gin-gonic/gin"
)

type App struct {
	Server *gin.Engine
	//消费者类似web服务器，因此引入App结构体
	Consumers []article.Consumer
}
