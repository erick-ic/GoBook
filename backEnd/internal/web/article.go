package web

import (
	"GoBook/internal/domain"
	"GoBook/internal/service"
	ijwt "GoBook/internal/web/jwt"
	"GoBook/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ArticleHandler struct {
	svc service.ArticleService
	l   logger.LoggerV1
}

func NewArticleHandler(svc service.ArticleService, l logger.LoggerV1) *ArticleHandler {
	return &ArticleHandler{
		svc: svc,
		l:   l,
	}
}

func (ah *ArticleHandler) RegisterRouters(server *gin.Engine) {
	group := server.Group("/articles")
	group.POST("/edit", ah.Edit)
	group.POST("/publish", ah.Publish)
}

func (ah *ArticleHandler) Edit(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	//检测输入
	//...

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

func (ah *ArticleHandler) Publish(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		return
	}
	//检测输入
	//...

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

type ArticleReq struct {
	Id      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

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
