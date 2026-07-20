package web

import (
	"GoBook/internal/domain"
	"GoBook/internal/service"
	ijwt "GoBook/internal/web/jwt"
	"GoBook/pkg/ginx"
	"GoBook/pkg/logger"
	"errors"
	"fmt"
	"net/http"

	regexp "github.com/dlclark/regexp2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	emailRegexPattern    = "^\\w+([-+.]\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$"
	passwordRegexPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,}$`
)

// UserHandler 定义user相关路由
type UserHandler struct {
	emailRegex    *regexp.Regexp
	passwordRegex *regexp.Regexp
	svc           service.UserService
	codeSvc       service.CodeService
	ijwt.JWTHandler
	cmd redis.Cmdable
	l   logger.LoggerV1
}

func NewUserHandler(
	svc service.UserService,
	codeSvc service.CodeService,
	cmd redis.Cmdable,
	jwtHandler ijwt.JWTHandler,
	l logger.LoggerV1,
) *UserHandler {
	emailExp := regexp.MustCompile(emailRegexPattern, regexp.None)
	passwordExp := regexp.MustCompile(passwordRegexPattern, regexp.None)

	return &UserHandler{
		emailRegex:    emailExp,
		passwordRegex: passwordExp,
		svc:           svc,
		codeSvc:       codeSvc,
		JWTHandler:    jwtHandler,
		cmd:           cmd,
		l:             l,
	}
}

func (uh *UserHandler) RegisterUsersRouters(server *gin.Engine) {
	ug := server.Group("/users")
	ug.POST("/signup", uh.SignUp)
	//ug.POST("/login", uh.Login)
	//ug.POST("/login", uh.LoginJWT)
	ug.POST("/login", ginx.WrapBodyV1[LoginReq](uh.LoginJWT))
	ug.POST("/create", uh.Create)
	ug.POST("/delete", uh.Delete)
	ug.POST("/edit", uh.Edit)
	ug.GET("/profile", uh.Profile)
	ug.POST("/sendSMSCode", uh.SendSMSCode)
	//ug.POST("/loginSMS", uh.LoginSMSCode)
	ug.POST("/loginSMS",
		ginx.WrapBody[LoginSMSReq](
			uh.l.With(logger.String("method", "loginSMS")),
			uh.LoginSMSCode,
		))
	ug.POST("/refreshToken", uh.RefreshToken)
	ug.POST("/logout", uh.LogoutJWT)
}

// SignUp 注册
func (uh *UserHandler) SignUp(ctx *gin.Context) {
	//请求参数结构体
	type SignUpReq struct {
		Email           string `json:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirm_password"`
	}

	//创建请求参数实例
	var req SignUpReq

	//Bind方法会根据Content-Type解析数据到req
	//若解析错误，则返回4XX错误
	if err := ctx.Bind(&req); err != nil {
		return
	}

	isEmail, err := uh.emailRegex.MatchString(req.Email)
	//系统内部错误
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	//邮箱格式不匹配
	if !isEmail {
		ctx.String(http.StatusOK, "邮箱格式错误！")
		return
	}

	if req.ConfirmPassword != req.Password {
		ctx.String(http.StatusOK, "两次输入的密码不一致")
		return
	}

	isPassword, err := uh.passwordRegex.MatchString(req.Password)
	if err != nil {
		//写入日志
		ctx.JSON(http.StatusOK, "系统错误")
		return
	}
	//密码格式不匹配
	if !isPassword {
		ctx.String(http.StatusOK, "密码必须大于8位，包含数字、特殊字符")
		return
	}

	err = uh.svc.SignUp(ctx.Request.Context(), domain.User{
		Email:    req.Email,
		Password: req.Password,
	})

	if errors.Is(err, service.ErrUserDuplicated) {
		ctx.String(http.StatusOK, "邮箱重复，请换一个！")
		return
	}

	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	ctx.String(http.StatusOK, "SignUp success~")
	fmt.Printf("req:%v \n", req)

}

type LoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginJWT 登录
func (uh *UserHandler) LoginJWT(ctx *gin.Context, req LoginReq) (Result, error) {
	//var req LoginReq
	//if err := ctx.Bind(&req); err != nil {
	//	return
	//}

	u, err := uh.svc.Login(ctx, req.Email, req.Password)

	if errors.Is(err, service.ErrInvalidUserPassword) {
		//ctx.JSON(http.StatusOK, Result{
		//	Code: 5,
		//	Msg:  "账号/邮箱或密码错误！",
		//})
		//return
		return Result{Code: 5, Msg: "账号/邮箱或密码错误！"}, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		//ctx.JSON(http.StatusOK, Result{
		//	Code: 5,
		//	Msg:  "用户不存在！",
		//})
		//return
		return Result{Code: 5, Msg: "用户不存在！"}, nil
	}
	if err != nil {
		//ctx.JSON(http.StatusInternalServerError, Result{
		//	Code: 5,
		//	Msg:  "系统错误！",
		//})
		//return
		return Result{Code: 5, Msg: "系统错误！"}, nil
	}

	////登录成功
	//err = uh.setJWTToken(ctx, u.Id)
	//if err != nil {
	//	ctx.JSON(http.StatusInternalServerError, Result{
	//		Code: 5,
	//		Msg:  "系统错误！",
	//	})
	//	return
	//}
	//
	//err = uh.setRefreshToken(ctx, u.Id)
	//if err != nil {
	//	ctx.JSON(http.StatusInternalServerError, Result{
	//		Code: 5,
	//		Msg:  "系统错误！",
	//	})
	//	return
	//}
	err = uh.SetLoginToken(ctx, u.Id)
	if err != nil {
		//ctx.JSON(http.StatusInternalServerError, Result{
		//	Code: 5,
		//	Msg:  "系统错误！",
		//})
		//return
		return Result{Code: 5, Msg: "系统错误！"}, nil
	}

	//ctx.JSON(http.StatusOK, Result{
	//	Code: 0,
	//	Msg:  "登录成功～",
	//})
	//return
	return Result{Code: 0, Msg: "登录成功～"}, nil
}

type LoginSMSReq struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

// LoginSMSCode 验证码登录
func (uh *UserHandler) LoginSMSCode(ctx *gin.Context, req LoginSMSReq) (ginx.Result, error) {
	// 1. 绑定请求参数（Phone + Code）
	//type LoginReq struct {
	//	Phone string `json:"phone"`
	//	Code  string `json:"code"`
	//}
	const biz = "login"
	//var req LoginReq
	//if err := ctx.Bind(&req); err != nil {
	//	return
	//}
	//// 2. 业务校验：手机号非空
	//if req.Phone == "" {
	//	ctx.JSON(http.StatusOK, Result{
	//		Code: 5,
	//		Msg:  "手机号码错误！",
	//	})
	//	return
	//}
	// 3. 调用验证码服务校验（biz="login" 区分业务场景）
	ok, err := uh.codeSvc.Verify(ctx, biz, req.Phone, req.Code)
	if err != nil {
		// 系统或业务错误（包括次数耗尽）
		switch err {
		case service.ErrCodeVerifyTooManyTimes:
			ctx.JSON(http.StatusOK, Result{
				Code: 5,
				Msg:  "验证码校验错误，请重新获取验证码",
			})
		default:
			//ctx.JSON(http.StatusInternalServerError, Result{
			//	Code: 5,
			//	Msg:  "系统错误！",
			//})
			//zap.L().Error("手机号码登录失败", zap.Error(err))
			//return ginx.Result{Code: 5, Msg: "手机号码登录失败"}, err
			return Result{Code: 5, Msg: "手机号码登录失败"},
				fmt.Errorf("手机号码登录失败 %w", err)
		}
	}

	if !ok {
		//// 验证码不匹配或已过期
		//ctx.JSON(http.StatusOK, ginx.Result{
		//	Code: 4,
		//	Msg:  "验证码错误！",
		//})
		return Result{Code: 4, Msg: "验证码错误!"}, nil
	}
	// 4. 【核心】查找或创建用户（传入手机号）
	//查找或创建该用户
	user, err := uh.svc.FindOrCreate(ctx, req.Phone)
	if err != nil {
		//ctx.JSON(http.StatusInternalServerError, Result{
		//	Code: 5,
		//	Msg:  "系统错误！",
		//})
		return Result{Code: 5, Msg: "系统错误！"},
			fmt.Errorf("登录或注册用户失败！ %w", err)
	}
	// 5. 生成 JWT 并写入响应头
	//if err := uh.setJWTToken(ctx, user.Id); err != nil {
	//	ctx.JSON(http.StatusInternalServerError, Result{
	//		Code: 5,
	//		Msg:  "系统错误！",
	//	})
	//	return
	//}
	//
	//if err := uh.setRefreshToken(ctx, user.Id); err != nil {
	//	ctx.JSON(http.StatusInternalServerError, Result{
	//		Code: 5,
	//		Msg:  "系统错误！",
	//	})
	//	return
	//}
	if err := uh.SetLoginToken(ctx, user.Id); err != nil {
		//ctx.JSON(http.StatusInternalServerError, Result{
		//	Code: 5,
		//	Msg:  "系统错误！",
		//})
		return Result{Code: 5, Msg: "系统错误！"},
			fmt.Errorf("系统错误！ %w", err)
	}

	// 6. 成功返回
	//ctx.JSON(http.StatusOK, Result{
	//	Code: 0,
	//	Msg:  "登录成功～",
	//})
	return Result{Code: 0, Msg: "登录成功～"}, nil
}

// SendSMSCode 发送验证码
func (uh *UserHandler) SendSMSCode(ctx *gin.Context) {
	type SMSCodeReq struct {
		Phone string `json:"phone"`
	}
	const biz = "login"
	var req SMSCodeReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	err := uh.codeSvc.Send(ctx, biz, req.Phone)
	switch err {
	case nil:
		ctx.JSON(http.StatusOK, Result{
			Code: 0,
			Msg:  "发送成功～",
		})
	case service.ErrCodeSendTooMany:
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "短信发送频繁，请稍后再试！",
		})
	default:
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "系统异常!",
		})

	}
}

// Login 登录
func (uh *UserHandler) Login(ctx *gin.Context) {
	type LoginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req LoginReq
	if err := ctx.Bind(&req); err != nil {
		return
	}

	u, err := uh.svc.Login(ctx, req.Email, req.Password)
	if errors.Is(err, service.ErrInvalidUserPassword) {
		ctx.String(http.StatusOK, "账号/邮箱或密码错误！")
		return
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		ctx.String(http.StatusOK, "用户不存在")
		return
	}
	if err != nil {
		ctx.String(http.StatusOK, "系统错误！")
		return
	}

	//登录成功
	//步骤2: 设置session，session初始化
	sess := sessions.Default(ctx)
	sess.Set("userId", u.Id)
	sess.Options(sessions.Options{
		//生产环境设置
		//Secure: true, 仅限https请求
		//HttpOnly: true,

		//cookie有效期是固定的
		MaxAge: 30, //单位秒
	})
	sess.Save()
	ctx.String(http.StatusOK, "LoginSuccess~")
	return

}

// Logout 退出登录
func (uh *UserHandler) Logout(ctx *gin.Context) {
	sess := sessions.Default(ctx)
	sess.Options(sessions.Options{
		//设置cookie有效期，即删除当前用户的cookie
		MaxAge: -1,
	})
	sess.Save()
	ctx.String(http.StatusOK, "LogoutSuccess~")
}

// Create 创建
func (uh *UserHandler) Create(ctx *gin.Context) {
	ctx.String(http.StatusOK, "CreateSuccess~")
}

// Delete 删除
func (uh *UserHandler) Delete(ctx *gin.Context) {
	ctx.String(http.StatusOK, "DeleteSuccess~")
}

// Edit 修改
func (uh *UserHandler) Edit(ctx *gin.Context) {
	ctx.String(http.StatusOK, "EditSuccess~")
}

// Profile 查看
func (uh *UserHandler) Profile(ctx *gin.Context) {
	// 1. 从 JWT claims 中获取用户信息，如ID
	v, ok := ctx.Get("claims")
	if !ok {
		ctx.String(http.StatusOK, "系统错误")
	}
	claims, ok := v.(*ijwt.UserClaims)
	if !ok {
		ctx.String(http.StatusOK, "系统错误")
	}
	fmt.Println("claims:", claims)

	// 2. 调用 Service 获取用户信息（Service 内部会走缓存）
	data, err := uh.svc.Profile(ctx, claims.Uid)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "获取用户信息失败！")
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"code":    http.StatusOK,
	})
}

// RefreshToken 刷新access_token，即刷新短token
// 只有调用该方法，才会获得refresh_token，其余都是access_token
func (uh *UserHandler) RefreshToken(ctx *gin.Context) {
	//1.提取token
	refreshToken := uh.ExtractToken(ctx)

	//2.使用 refreshTokenKey 验签
	var rc ijwt.RefreshClaims
	token, err := jwt.ParseWithClaims(refreshToken, &rc, func(token *jwt.Token) (interface{}, error) {
		return ijwt.RefreshTokenKey, nil
	})
	if err != nil || !token.Valid {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	//查看ssid是否有效
	//cnt, err := uh.cmd.Exists(ctx, fmt.Sprintf("users:ssid:%s", rc.Ssid)).Result()
	err = uh.CheckSession(ctx, rc.Ssid)
	if err != nil {
		//Redis问题或已退出登录
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	//3.更新token
	err = uh.SetJWTToken(ctx, rc.Uid, rc.Ssid)
	if err != nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Code: 0,
		Msg:  "token刷新成功～",
	})
}

// LogoutJWT 退出登录
func (uh *UserHandler) LogoutJWT(ctx *gin.Context) {
	err := uh.ClearToken(ctx)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "退出登录失败！",
		})
		return
	}

	ctx.JSON(http.StatusOK, Result{
		Code: 0,
		Msg:  "退出登录成功～",
	})
}
