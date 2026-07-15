package web

import (
	"GoBook/internal/domain"
	"GoBook/internal/service"
	ijwt "GoBook/internal/web/jwt"
	"GoBook/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ArticleHandler 文章处理器，处理文章相关的HTTP请求
type ArticleHandler struct {
	svc service.ArticleService // 文章服务接口
	l   logger.LoggerV1        // 日志记录器
}

// NewArticleHandler 创建文章处理器实例
func NewArticleHandler(svc service.ArticleService, l logger.LoggerV1) *ArticleHandler {
	return &ArticleHandler{
		svc: svc,
		l:   l,
	}
}

// RegisterRouters 注册文章相关的路由
func (ah *ArticleHandler) RegisterRouters(server *gin.Engine) {
	group := server.Group("/articles")
	group.POST("/edit", ah.Edit)         // 编辑/保存文章接口
	group.POST("/publish", ah.Publish)   // 发表文章接口
	group.POST("/withdraw", ah.Withdraw) // 撤回文章接口
}

// Edit 处理文章编辑请求，保存草稿到制作库
func (ah *ArticleHandler) Edit(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		return
	}

	c, ok := ctx.Get("claims")
	claims := c.(*ijwt.UserClaims)

	if !ok {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		ah.l.Error("未发现用户session信息")
		return
	}

	id, err := ah.svc.Save(ctx, req.toDomain(claims.Uid))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		ah.l.Error("修改帖子失败", logger.Error(err))
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Code: 0,
		Msg:  "编辑成功～",
		Data: id,
	})
}

// Publish 处理文章发表请求，将文章同步到制作库和线上库
func (ah *ArticleHandler) Publish(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		return
	}

	c, ok := ctx.Get("claims")
	claims := c.(*ijwt.UserClaims)

	if !ok {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		ah.l.Error("未发现用户session信息")
		return
	}

	id, err := ah.svc.Publish(ctx, req.toDomain(claims.Uid))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		ah.l.Error("发表帖子失败", logger.Error(err))
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Code: 0,
		Msg:  "OK～",
		Data: id,
	})
}

// Withdraw 处理文章撤回请求，将已发表的文章状态改为未发表
func (ah *ArticleHandler) Withdraw(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		return
	}

	c, ok := ctx.Get("claims")
	claims := c.(*ijwt.UserClaims)

	if !ok {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		ah.l.Error("未发现用户session信息")
		return
	}

	id, err := ah.svc.Withdraw(ctx, req.toDomain(claims.Uid))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		ah.l.Error("撤回帖子失败", logger.Error(err))
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Code: 0,
		Msg:  "OK～",
		Data: id,
	})
}

// ArticleReq 文章请求结构体
type ArticleReq struct {
	Id      int64  `json:"id"`      // 文章ID，新增时为0
	Title   string `json:"title"`   // 文章标题
	Content string `json:"content"` // 文章内容
}

// toDomain 将请求结构体转换为领域模型，注入作者ID
func (ar *ArticleReq) toDomain(uid int64) domain.Article {
	return domain.Article{
		Id:      ar.Id,
		Title:   ar.Title,
		Content: ar.Content,
		Author: domain.Author{
			Id: uid,
		},
	}
}
