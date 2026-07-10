package wechat

import (
	"GoBook/internal/domain"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// redirectURI参数需要转码
var redirectURI = url.PathEscape("https://ichenghub.cn/oauth2/wechat/callback")

type Service interface {
	AuthURL(ctx context.Context, state string) (string, error)
	VerifyCode(ctx context.Context, code string) (domain.WechatInfo, error)
}

type service struct {
	appId     string
	appSecret string
	client    *http.Client
}

//标准写法：
//func NewService(appId, appSecret string, client *http.Client) Service {
//	return &service{
//		appId:     appId,
//		appSecret: appSecret,
//		client:    client, //依赖注入
//	}
//}

func NewService(appId, appSecret string) Service {
	return &service{
		appId:     appId,
		appSecret: appSecret,
		client:    http.DefaultClient, //直接初始化
	}
}

func (s *service) AuthURL(ctx context.Context, state string) (string, error) {
	const urlPattern = "https://open.weixin.qq.com/connect/qrconnect?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_login&state=%s#wechat_redirect"
	return fmt.Sprintf(urlPattern, s.appId, redirectURI, state), nil
}

func (s *service) VerifyCode(ctx context.Context, code string) (domain.WechatInfo, error) {
	const targetPattern = "https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code"
	target := fmt.Sprintf(targetPattern, s.appId, s.appSecret, code)
	//发起http请求方式1
	//resp, err := http.Get(target)

	//发起http请求方式2
	//resp, err := http.NewRequest(http.MethodGet, target, nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return domain.WechatInfo{}, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return domain.WechatInfo{}, err
	}

	decoder := json.NewDecoder(resp.Body)
	var wechatRes WechatResult
	err = decoder.Decode(&wechatRes)
	if err != nil {
		return domain.WechatInfo{}, err
	}

	if wechatRes.ErrCode != 0 {
		return domain.WechatInfo{}, errors.New(wechatRes.ErrMsg)
	}

	return domain.WechatInfo{
		OpenId:  wechatRes.Openid,
		UnionId: wechatRes.Unionid,
	}, nil
}

//{
//	"access_token": "ACCESS_TOKEN",
//	"expires_in": 7200,
//	"refresh_token": "REFRESH_TOKEN",
//	"openid": "OPENID",
//	"scope": "SCOPE",
//	"unionid": "o6_bmasdasdsad6_2sgVt7hMZOPfL"
//}

type WechatResult struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Openid       string `json:"openid"`
	Scope        string `json:"scope"`
	Unionid      string `json:"unionid"`

	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}
