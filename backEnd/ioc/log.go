package ioc

import (
	"GoBook/pkg/logger"

	"go.uber.org/zap"
)

// InitLogger 全局Logger
func InitLogger() logger.LoggerV1 {
	l, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return logger.NewZapLogger(l)
}
