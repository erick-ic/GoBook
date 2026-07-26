package main

import (
	"GoBook/pkg/grpcx"
	"GoBook/pkg/saramax"
)

// App 聚合互动微服务运行时依赖，由 Wire 完成装配。
type App struct {
	server    *grpcx.Server
	consumers []saramax.Consumer
}
