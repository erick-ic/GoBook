// Package ginx 提供基于 Gin 框架的请求处理模板封装。
// 通过泛型将 HTTP 请求解析、错误处理、响应返回等通用逻辑抽离为 Wrapper 函数，
// 使业务 Handler 只需关注纯业务逻辑，无需重复编写 Bind、JSON 返回等样板代码。
package ginx

import (
	"GoBook/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Result 统一的 HTTP 响应结构体
type Result struct {
	Code int    `json:"code"` // 业务状态码，0 表示成功
	Msg  string `json:"msg"`  // 提示信息
	Data any    `json:"data"` // 响应数据
}

// WrapBody 绑定请求体并执行业务逻辑的通用包装器。
// 负责：解析请求参数 → 调用业务函数 → 记录错误日志 → 返回 JSON 响应。
// 适用场景：需要传入独立 logger 实例的接口。
func WrapBody[T any](l logger.LoggerV1, fn func(ctx *gin.Context, req T) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req T
		if err := ctx.Bind(&req); err != nil {
			return
		}
		// 调用业务函数
		res, err := fn(ctx, req)
		if err != nil {
			// 业务错误记录日志
			l.Error("处理业务逻辑错误", logger.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
	}
}

// L 全局日志实例，供 WrapBodyV1、WrapToken、WrapBodyAndToken 使用。
// 需在项目启动时完成初始化，否则可能导致空指针。
var L logger.LoggerV1

// WrapBodyV1 绑定请求体并执行业务逻辑的通用包装器（使用全局日志 L）。
// 负责：解析请求参数 → 调用业务函数 → 记录错误日志 → 返回 JSON 响应。
// 适用场景：不需要单独 logger 实例，直接使用全局日志的接口。
func WrapBodyV1[T any](fn func(ctx *gin.Context, req T) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req T
		if err := ctx.Bind(&req); err != nil {
			return
		}
		// 调用业务函数
		res, err := fn(ctx, req)
		if err != nil {
			// 业务错误记录日志
			L.Error("处理业务逻辑错误", logger.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
	}
}

// WrapToken 仅验证并提取 JWT Claims 的通用包装器。
// 负责：从上下文中获取用户信息 → 类型断言为指定 Claims 类型 → 调用业务函数 → 返回 JSON 响应。
// 适用场景：无需请求体，只需要登录态的接口（如获取当前用户信息）。
// 注意：依赖上下文中的 "users" 键，需确保认证中间件已正确注入。
func WrapToken[Cls jwt.Claims](fn func(ctx *gin.Context, cls Cls) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		val, ok := ctx.Get("claims")
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c, ok := val.(Cls)
		if !ok {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 调用业务函数
		res, err := fn(ctx, c)
		if err != nil {
			// 业务错误记录日志
			L.Error("处理业务逻辑错误", logger.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
	}
}

// WrapBodyAndToken 同时绑定请求体并验证 JWT Claims 的通用包装器。
// 负责：解析请求参数 → 提取登录态 Claims → 调用业务函数 → 返回 JSON 响应。
// 适用场景：既需要请求体又需要登录态的接口（如发表文章、修改个人资料）。
// 注意：依赖上下文中的 "users" 键，需确保认证中间件已正确注入。
func WrapBodyAndToken[Req any, Cls jwt.Claims](fn func(ctx *gin.Context, req Req, cls Cls) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		if err := ctx.Bind(&req); err != nil {
			return
		}

		val, ok := ctx.Get("claims")
		if !ok {
			// claims 不存在
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c, ok := val.(Cls)
		if !ok {
			//类型不匹配
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 调用业务函数
		res, err := fn(ctx, req, c)
		if err != nil {
			// 业务错误记录日志
			L.Error("处理业务逻辑错误", logger.Error(err))
		}
		ctx.JSON(http.StatusOK, res)
	}
}
