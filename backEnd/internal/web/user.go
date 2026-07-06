package web

import (
	"GoBook/internal/domain"
	"GoBook/internal/service"
	"errors"
	"fmt"
	"net/http"
	"time"

	regexp "github.com/dlclark/regexp2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
	svc           *service.UserService
	codeSvc       *service.CodeService
}

func NewUserHandler(svc *service.UserService, codeSvc *service.CodeService) *UserHandler {
	emailExp := regexp.MustCompile(emailRegexPattern, regexp.None)
	passwordExp := regexp.MustCompile(passwordRegexPattern, regexp.None)

	return &UserHandler{
		emailRegex:    emailExp,
		passwordRegex: passwordExp,
		svc:           svc,
		codeSvc:       codeSvc,
	}
}

func (uh *UserHandler) RegisterUsersRouters(server *gin.Engine) {
	ug := server.Group("/users")
	ug.POST("/signup", uh.SignUp)
	//ug.POST("/login", uh.Login)
	ug.POST("/login", uh.LoginJWT)
	ug.POST("/create", uh.Create)
	ug.POST("/delete", uh.Delete)
	ug.POST("/edit", uh.Edit)
	ug.GET("/profile", uh.Profile)
	ug.POST("/sendSMSCode", uh.SendSMSCode)
	ug.POST("/loginSMS", uh.LoginSMSCode)
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

	err = uh.svc.SignUp(ctx, domain.User{
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

type UserClaims struct {
	//继承RegisteredClaims，实现claims
	jwt.RegisteredClaims
	//放入token的数据
	Uid       int64
	UserAgent string
}

// LoginJWT 登录
func (uh *UserHandler) LoginJWT(ctx *gin.Context) {
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
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "账号/邮箱或密码错误！",
		})
		return
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "用户不存在！",
		})
		return
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		return
	}

	//登录成功
	err = uh.setJWTToken(ctx, u.Id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		return
	}

	ctx.JSON(http.StatusOK, Result{
		Code: 0,
		Msg:  "登录成功～",
	})
	return
}

func (uh *UserHandler) setJWTToken(ctx *gin.Context, uid int64) error {
	//生成token
	//token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
	//	"userId": u.Id,
	//})
	claims := UserClaims{
		//设置token有效期
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
		},
		Uid:       uid,
		UserAgent: ctx.Request.UserAgent(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//随机生成32位key 95osj3fUD7fo0mlYdDbncXz4VD2igvf0
	tokenStr, err := token.SignedString([]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"))
	if err != nil {
		return err
	}
	//通过Http Response Header x-jwt-token返回
	ctx.Header("x-jwt-token", tokenStr)
	return nil
}

// LoginSMSCode 验证码登录
func (uh *UserHandler) LoginSMSCode(ctx *gin.Context) {
	// 1. 绑定请求参数（Phone + Code）
	type LoginReq struct {
		Phone string `json:"phone"`
		Code  string `json:"code"`
	}
	const biz = "login"
	var req LoginReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	// 2. 业务校验：手机号非空
	if req.Phone == "" {
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "手机号码错误！",
		})
		return
	}
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
			ctx.JSON(http.StatusInternalServerError, Result{
				Code: 5,
				Msg:  "系统错误！",
			})
		}
		return
	}

	if !ok {
		// 验证码不匹配或已过期
		ctx.JSON(http.StatusOK, Result{
			Code: 5,
			Msg:  "验证码错误！",
		})
		return
	}
	// 4. 【核心】查找或创建用户（传入手机号）
	//查找或创建该用户
	user, err := uh.svc.FindOrCreate(ctx, req.Phone)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		return
	}
	// 5. 生成 JWT 并写入响应头
	if err := uh.setJWTToken(ctx, user.Id); err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		return
	}

	// 6. 成功返回
	ctx.JSON(http.StatusOK, Result{
		Code: 0,
		Msg:  "登录成功～",
	})
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
	claims, ok := v.(*UserClaims)
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
