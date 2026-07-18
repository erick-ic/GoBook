package main

import (
	"GoBook/internal/events/article"

	"github.com/gin-gonic/gin"
)

type App struct {
	Server    *gin.Engine
	Consumers []article.Consumer
}
