package web

import (
	"GoBook/internal/service"
	"GoBook/internal/service/oauth2/wechat"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	uuid "github.com/lithammer/shortuuid/v4"
)

type OAuth2WechatHandler struct {
	svc     wechat.Service
	userSvc service.UserService
	jwtHandler
	stateKey []byte
	cfg      WechatHandlerConfig
}

type WechatHandlerConfig struct {
	Secure bool
}

func NewOAuth2WechatHandler(svc wechat.Service, userSvc service.UserService, cfg WechatHandlerConfig) *OAuth2WechatHandler {
	return &OAuth2WechatHandler{
		svc:        svc,
		userSvc:    userSvc,
		stateKey:   []byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf6"),
		cfg:        cfg,
		jwtHandler: newJwtHandler(),
	}
}

func (oh *OAuth2WechatHandler) RegisterWechatRouters(server *gin.Engine) {
	group := server.Group("/oauth2/wechat")
	group.GET("/authurl", oh.AuthURl)
	group.GET("/callback", oh.Callback)
	//用户确认授权后，微信的服务器并不会把结果发给后端的其他接口，
	//而是命令浏览器（通过 302 重定向）自动跳转到事先在 redirect_uri 中
	//指定的地址，也就是 GET /oauth2/wechat/callback。
}

// AuthURl 请求code
func (oh *OAuth2WechatHandler) AuthURl(ctx *gin.Context) {
	state := uuid.New()
	url, err := oh.svc.AuthURL(ctx, state)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "构造扫码登录URL失败！",
		})
		return
	}

	//token中保存state
	if err = oh.setStateCookie(ctx, state); err != nil {
		return
	}

	ctx.JSON(http.StatusOK, Result{
		Code: 0,
		Data: url,
	})
}

// Callback 获取并解析code，用户扫码确认后浏览器重定向，调用callback
func (oh *OAuth2WechatHandler) Callback(ctx *gin.Context) {
	//从路径中解析出code
	code := ctx.Query("code")
	err := oh.verifyCode(ctx)
	if err != nil {
		return
	}
	wechatInfo, err := oh.svc.VerifyCode(ctx, code)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		return
	}
	//登录成功

	//获取uid
	u, err := oh.userSvc.FindOrCreateByWechat(ctx, wechatInfo)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		return
	}

	err = oh.setJWTToken(ctx, u.Id)
	if err != nil {
		return
	}

	err = oh.setRefreshToken(ctx, u.Id)
	if err != nil {
		return
	}

	ctx.JSON(http.StatusOK, Result{
		Code: 0,
		Msg:  "OK~",
	})
}

type StateClaims struct {
	jwt.RegisteredClaims
	State string `json:"state"`
}

func (oh *OAuth2WechatHandler) verifyCode(ctx *gin.Context) error {
	state := ctx.Query("state")
	//校验state
	checkState, err := ctx.Cookie("jwt-state")
	if err != nil {
		//计入监控，有人攻击...
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  fmt.Sprintf("拿不到state的cookie, %s", err),
		})
		return err
	}
	var sc StateClaims
	token, err := jwt.ParseWithClaims(checkState, &sc,
		func(token *jwt.Token) (interface{}, error) {
			return oh.stateKey, nil
		})
	if err != nil || !token.Valid {
		//计入监控，有人攻击...
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  fmt.Sprintf("token过期, %s", err),
		})
		return err
	}
	if sc.State != state {
		//计入监控，有人攻击...
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  fmt.Sprintf("sc.State: %s, state: %s", sc.State, state),
		})
		return fmt.Errorf("token过期, %w", err)
	}
	return nil
}

func (oh *OAuth2WechatHandler) setStateCookie(ctx *gin.Context, state string) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, StateClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			//过期时间，扫码登录
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 10)),
		},
		State: state,
	})
	tokenStr, err := token.SignedString(oh.stateKey)
	if err != nil {
		return err
	}
	//无法设置header，页面会进行刷新。
	ctx.SetCookie("jwt-state", tokenStr,
		600,
		"/oauth2/wechat/callback",
		"", oh.cfg.Secure, true)
	return nil
}
