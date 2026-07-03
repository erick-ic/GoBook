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
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	emailExp := regexp.MustCompile(emailRegexPattern, regexp.None)
	passwordExp := regexp.MustCompile(passwordRegexPattern, regexp.None)

	return &UserHandler{
		svc:           svc,
		emailRegex:    emailExp,
		passwordRegex: passwordExp,
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
	//生成token

	//token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
	//	"userId": u.Id,
	//})
	claims := UserClaims{
		//设置token有效期
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
		},
		Uid:       u.Id,
		UserAgent: ctx.Request.UserAgent(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//随机生成32位key 95osj3fUD7fo0mlYdDbncXz4VD2igvf0
	tokenStr, err := token.SignedString([]byte("95osj3fUD7fo0mlYdDbncXz4VD2igvf0"))
	if err != nil {
		ctx.String(http.StatusInternalServerError, "系统错误！")
	}
	//通过Http Response Header x-jwt-token返回
	ctx.Header("x-jwt-token", tokenStr)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    u,
		"code":    http.StatusOK,
	})
	return

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
//
//	func (uh *UserHandler) Profile(ctx *gin.Context) {
//		ctx.String(http.StatusOK, "this is profile~")
//	}
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
