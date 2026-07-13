package logger

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/gin-gonic/gin"
)

type MiddlewareBuilder struct {
	//是否允许打印请求体
	allowReqBody bool

	//是否允许打印响应体
	allowRespBody bool

	//日志记录回调函数
	loggerFunc func(ctx context.Context, al *AccessLog)
}

func NewMiddlewareBuilder(fn func(ctx context.Context, al *AccessLog)) *MiddlewareBuilder {
	return &MiddlewareBuilder{
		loggerFunc: fn,

		//默认false
		//allowRespBody: false,
		//allowReqBody: false,
	}
}

// SetAllowReqBody 打印请求体
func (mb *MiddlewareBuilder) SetAllowReqBody() *MiddlewareBuilder {
	mb.allowReqBody = true
	//返回自己，实现链式调用
	return mb
}

// SetAllowRespBody 打印响应体
func (mb *MiddlewareBuilder) SetAllowRespBody() *MiddlewareBuilder {
	mb.allowRespBody = true
	return mb
}

func (mb *MiddlewareBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		//记录开始时间
		start := time.Now()

		//截取 URL 防止内存溢出
		url := ctx.Request.URL.String()
		if len(url) > 1024 {
			url = url[:1024]
		}

		//初始化AccessLog 对象
		al := &AccessLog{
			Method: ctx.Request.Method,
			Url:    url,
		}

		//请求体处理：读取并重置Body
		if mb.allowReqBody && ctx.Request.Body != nil {
			//读取原始字节流
			body, _ := ctx.GetRawData()
			//重新封装为ReadCloser
			//ctx.Request.Body为ReadCloser类型
			//数据流只能被读取一次。
			//如果这里读了，后面的业务代码（比如 ShouldBindJSON）就读不到了
			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(body))

			//将字节转为字符串存入 AccessLog
			//该操作会引起复制，消耗CPU，进行截断
			if len(body) > 1024 {
				body = body[:1024]
			}
			al.ReqBody = string(body)
		}

		//响应体操作：劫持ResponseWriter
		//ctx.Writer 是 gin.ResponseWriter 接口。
		//将它替换成自定义的 responseWriter 结构体。
		if mb.allowRespBody {
			ctx.Writer = responseWriter{
				//保存原始Writer
				ResponseWriter: ctx.Writer,
				//传入AccessLog引用
				al: al,
			}
		}

		//defer 延迟执行（确保在业务逻辑结束后运行）
		//无论业务是否正常返回，还是调用ctx.Abort()，defer都会在函数退出前执行。
		defer func() {
			//计算耗时
			al.Duration = time.Since(start).String()
			//调用回调函数，打印日志
			mb.loggerFunc(ctx, al)
		}()

		//执行业务逻辑
		ctx.Next()
	}
}

// AccessLog 日志数据模型
type AccessLog struct {
	Method   string // HTTP 方法（GET/POST）
	Url      string // 请求路径（截断后）
	ReqBody  string // 请求体（如 JSON）
	RespBody string // 响应体（如返回的 JSON）
	Duration string // 耗时（如 "15.2ms"）
	Status   int    // HTTP 状态码（200/404/500）
}

// responseWriter 劫持器，捕获响应数据（装饰器模式：追加功能）
type responseWriter struct {
	// 匿名字段，继承了原始 Writer 的所有方法
	// 嵌入gin.ResponseWriter，自动实现接口的所有方法，因此只需覆盖自己需要的方法即可。
	gin.ResponseWriter
	// 持有 AccessLog 引用
	al *AccessLog
}

func (rw responseWriter) WriteHeader(statusCode int) {
	//捕获状态码
	rw.al.Status = statusCode
	// 调用原始方法，真正设置响应头
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw responseWriter) Write(data []byte) (int, error) {
	// 捕获响应体（转为字符串）
	rw.al.RespBody = string(data)
	// 将数据真正写回客户端
	return rw.ResponseWriter.Write(data)
}

func (rw responseWriter) WriteString(data string) (int, error) {
	// 捕获响应体
	rw.al.RespBody = data
	return rw.ResponseWriter.WriteString(data)
}
