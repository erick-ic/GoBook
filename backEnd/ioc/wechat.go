package ioc

import (
	"GoBook/internal/service/oauth2/wechat"
	"GoBook/internal/web"
	"GoBook/pkg/logger"
)

func InitOAuth2WechatService(l logger.LoggerV1) wechat.Service {
	appId := "123456"
	appSecret := "2345678"
	return wechat.NewService(appId, appSecret, l)
}

func NewOAuth2WechatConfig() web.WechatHandlerConfig {
	return web.WechatHandlerConfig{
		Secure: false,
	}
}
