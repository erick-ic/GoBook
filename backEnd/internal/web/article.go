package web

import (
	"GoBook/internal/domain"
	"GoBook/internal/service"
	ijwt "GoBook/internal/web/jwt"
	"GoBook/pkg/ginx"
	"GoBook/pkg/logger"
	"net/http"
	"strconv"
	"time"

	"github.com/ecodeclub/ekit/slice"
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

	group.POST("/list",
		ginx.WrapBodyAndToken[ListReq, ijwt.UserClaims](ah.ArticleList))
	group.GET("/detail/:id", ah.Detail)
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

func (ah *ArticleHandler) ArticleList(ctx *gin.Context, req ListReq, uc ijwt.UserClaims) (Result, error) {
	res, err := ah.svc.List(ctx, uc.Uid, req.offset, req.limit)
	if err != nil {
		return Result{Code: 5, Msg: "系统错误！"}, nil
	}

	data := slice.Map[domain.Article, ArticleVO](
		res,
		func(idx int, src domain.Article) ArticleVO {
			return ArticleVO{
				Id:       src.Id,
				Title:    src.Title,
				Abstract: src.Abstract(),
				Status:   src.Status.ToUint8(),
				//列表不需要返回content
				//Content: src.Content,
				//列表不需要返回author
				//Author: src.Author.Name,
				Ctime: src.Ctime.Format(time.DateTime),
				Utime: src.Utime.Format(time.DateTime),
			}
		})
	return Result{
		Code: 0,
		Data: data,
	}, nil
}

func (ah *ArticleHandler) Detail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 4,
			Msg:  "id 参数错误！",
		})
		ah.l.Warn("查询文章失败，id 错误！",
			logger.String("id", idStr),
			logger.Error(err),
		)
		return
	}
	res, err := ah.svc.GetById(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		ah.l.Error("查询文章失败",
			logger.Int64("id", id),
			logger.Error(err),
		)
		return
	}
	vo := ArticleVO{
		Id:      res.Id,
		Title:   res.Title,
		Content: res.Content,
		Status:  res.Status.ToUint8(),
		Ctime:   res.Ctime.Format(time.DateTime),
		Utime:   res.Utime.Format(time.DateTime),
	}
	ctx.JSON(http.StatusOK, Result{
		Code: 0,
		Data: vo,
	})
}
