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
// 调用链路：HTTP请求 → ArticleHandler → ArticleService → ArticleRepository → ArticleDAO
type ArticleHandler struct {
	svc      service.ArticleService     // 文章服务接口，处理文章核心业务
	interSvc service.InteractiveService // 互动服务接口，处理点赞/收藏/阅读数
	l        logger.LoggerV1            // 日志记录器
	biz      string                     // 业务标识，用于互动服务区分业务类型（"article"）
}

// NewArticleHandler 创建文章处理器实例
// biz 字段硬编码为 "article"，互动数据通过 biz+bizId 唯一标识
func NewArticleHandler(
	svc service.ArticleService,
	l logger.LoggerV1,
	interSvc service.InteractiveService,
) *ArticleHandler {
	return &ArticleHandler{
		svc:      svc,
		interSvc: interSvc,
		l:        l,
		biz:      "article",
	}
}

// RegisterRouters 注册文章相关的路由
// 路由分为两组：
//   - /articles/*：作者视角（需要登录，操作自己的文章）
//   - /pub/*：读者视角（访问已发表文章，点赞等）
func (ah *ArticleHandler) RegisterRouters(server *gin.Engine) {
	group := server.Group("/articles")
	group.POST("/edit", ah.Edit)         // 编辑/保存文章草稿
	group.POST("/publish", ah.Publish)   // 发表文章（同步到制作库和线上库）
	group.POST("/withdraw", ah.Withdraw) // 撤回文章（状态改为未发表）

	// 使用 ginx.WrapBodyAndToken 泛型封装，自动解析请求体和JWT claims
	group.POST("/list",
		ginx.WrapBodyAndToken[ListReq, *ijwt.UserClaims](ah.ArticleList))

	group.GET("/detail/:id", ah.Detail) // 查询文章详情（编辑用，从制作库取）

	pubGroup := server.Group("/pub")
	pubGroup.GET("/:id", ah.PubDetail)  // 读者访问已发表文章详情
	pubGroup.POST("/like/:id", ah.Like) // 点赞/取消点赞文章

}

// Like 处理文章点赞请求
// 调用 InteractiveService.Like 实现点赞/取消点赞的切换（幂等操作）
// 调用链路：POST /pub/like/:id → Like → InteractiveService.Like → Repository（Redis+DB）
func (ah *ArticleHandler) Like(ctx *gin.Context) {
	idStr := ctx.Param("id")
	articleId, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "id 参数错误！",
		})
		ah.l.Warn("查询文章失败，id格式不对！",
			logger.String("articleId", idStr),
			logger.Error(err),
		)
		return
	}

	// 从 gin.Context 中获取 JWT claims（由 JWT 中间件设置）
	c, ok := ctx.Get("claims")
	claims := c.(*ijwt.UserClaims)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		ah.l.Warn("查询文章失败，系统错误！",
			logger.Int64("articleId", articleId),
			logger.Int64("userId", claims.Uid),
			logger.Error(err),
		)
		return
	}

	// 调用互动服务，Like 方法内部判断是否已点赞，实现点赞/取消的切换
	err = ah.interSvc.Like(ctx, ah.biz, articleId, claims.Uid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		return
	}
	ctx.JSON(http.StatusOK, Result{
		Code: 0,
		Msg:  "OK~",
	})
}

// PubDetail 处理读者访问已发表文章详情的请求
// 调用链路：GET /pub/:id → PubDetail → ArticleService.GetByPubId + InteractiveService.Get
//
// 执行流程：
//  1. 解析文章ID和用户身份
//  2. 调用 GetByPubId 获取文章内容（同时异步发送阅读事件到Kafka）
//  3. 调用 InteractiveService.Get 获取互动数据（阅读数/点赞数/收藏数）
//  4. 组装 ArticleVO 返回给前端
//
// 阅读计数说明：
//   - 不在此处同步递增阅读数（已废弃直接调用 InccrReadCnt 的方式）
//   - 改由 Service 层 GetByPubId 异步发送 Kafka 事件，消费者更新阅读数
//   - 优点：响应快；缺点：本次返回的阅读数是旧值（最终一致）
func (ah *ArticleHandler) PubDetail(ctx *gin.Context) {
	idStr := ctx.Param("id")
	articleId, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusOK, Result{
			Code: 4,
			Msg:  "id 参数错误！",
		})
		ah.l.Warn("查询文章失败，id格式不对！",
			logger.String("articleId", idStr),
			logger.Error(err),
		)
		return
	}

	c, ok := ctx.Get("claims")
	claims := c.(*ijwt.UserClaims)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误！",
		})
		ah.l.Warn("查询文章失败，系统错误！",
			logger.Int64("articleId", articleId),
			logger.Int64("userId", claims.Uid),
			logger.Error(err),
		)
		return
	}

	// 获取文章详情（从线上库），Service 层会异步发送阅读事件到 Kafka
	res, err := ah.svc.GetByPubId(ctx, articleId, claims.Uid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		ah.l.Error("查询文章失败",
			logger.Int64("articleId", articleId),
			logger.Error(err),
		)
		return
	}

	// 获取互动数据（阅读数/点赞数/收藏数/是否点赞/是否收藏）
	iter, err := ah.interSvc.Get(ctx, ah.biz, articleId, claims.Uid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, Result{
			Code: 5,
			Msg:  "系统错误",
		})
		ah.l.Error("查询文章失败",
			logger.Int64("articleId", articleId),
			logger.Error(err),
		)
		return
	}

	// 组装 VO 返回前端
	vo := ArticleVO{
		Id:         res.Id,
		Title:      res.Title,
		Content:    res.Content,
		ReadCnt:    iter.ReadCnt,
		LikeCnt:    iter.LikeCnt,
		CollectCnt: iter.CollectCnt,

		Status: res.Status.ToUint8(),
		Ctime:  res.Ctime.Format(time.DateTime),
		Utime:  res.Utime.Format(time.DateTime),
	}
	ctx.JSON(http.StatusOK, Result{
		Code: 0,
		Msg:  "OK~",
		Data: vo,
	})

	// 增加阅读计数：已改为通过 Kafka 异步解耦，由 Service 层 GetByPubId 发送阅读事件，
	// 消费者收到事件后调用 IncrReadCnt 更新数据库和缓存。
	// 以下直接调用方式已废弃，保留注释供参考：
	// go func() {
	// 	er := ah.interSvc.IncrReadCnt(ctx, ah.biz, articleId)
	// 	if er != nil {
	// 		ah.l.Error("增加阅读计数失败！",
	// 			logger.Int64("articleId", articleId),
	// 			logger.Error(er))
	// 		return
	// 	}
	// }()
}

// Edit 处理文章编辑请求，保存草稿到制作库
func (ah *ArticleHandler) Edit(ctx *gin.Context) {
	var req ArticleReq
	if err := ctx.Bind(&req); err != nil {
		return
	}

	c, ok := ctx.Get("claims")
	if !ok {
		// claims 不存在
		return
	}

	claims, ok := c.(*ijwt.UserClaims)
	if !ok {
		// 类型不匹配
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

// ArticleList 处理文章列表查询请求
// 调用链路：POST /articles/list → ginx.WrapBodyAndToken → ArticleList → ArticleService.List
//
// 注意：此方法签名符合 ginx.WrapBodyAndToken 的要求：
//   - 参数2：请求体（自动从 JSON 解析）
//   - 参数3：JWT claims（自动从 gin.Context 提取）
//   - 返回值1：Result（自动包装为 HTTP 响应）
//   - 返回值2：error（非nil时由 ginx 处理错误）
func (ah *ArticleHandler) ArticleList(ctx *gin.Context, req ListReq, uc *ijwt.UserClaims) (Result, error) {
	res, err := ah.svc.List(ctx, uc.Uid, req.Offset, req.Limit)
	if err != nil {
		return Result{Code: 5, Msg: "系统错误！"}, nil
	}

	// 将领域模型列表转换为 VO 列表
	// 列表接口不返回 content（数据量大），只返回摘要 Abstract
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

// Detail 处理文章详情查询请求（作者视角）
// 调用链路：GET /articles/detail/:id → Detail → ArticleService.GetById → Repository（制作库）
// 用于作者编辑文章时获取完整内容（含未发表草稿）
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
