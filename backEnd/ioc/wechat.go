package ioc

import (
	"GoBook/internal/service/oauth2/wechat"
	"GoBook/internal/web"
)

func InitOAuth2WechatService() wechat.Service {
	appId := "123456"
	appSecret := "2345678"
	return wechat.NewService(appId, appSecret)
}

func NewOAuth2WechatConfig() web.WechatHandlerConfig {
	return web.WechatHandlerConfig{
		Secure: false,
	}
}
