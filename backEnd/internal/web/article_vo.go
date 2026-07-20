// VO 展示给前端的数据
package web

import "GoBook/internal/domain"

// ArticleReq 文章编辑/发表请求结构体
// 用于 /articles/edit、/articles/publish、/articles/withdraw 接口
type ArticleReq struct {
	Id      int64  `json:"id"`      // 文章ID，新增时为0，更新时为已有文章ID
	Title   string `json:"title"`   // 文章标题
	Content string `json:"content"` // 文章内容
}

// ListReq 文章列表查询请求结构体
// 用于 /articles/list 接口，支持分页查询
type ListReq struct {
	Offset int `json:"offset"` // 偏移量，从第几条开始
	Limit  int `json:"limit"`  // 每页数量
}

// ArticleVO 文章视图对象，返回给前端的展示数据
// 根据不同接口返回不同字段：
//   - 列表接口：返回 Id/Title/Abstract/Status/Ctime/Utime（不含 Content）
//   - 详情接口：返回完整字段（含 Content）
//   - 读者接口：返回互动数据（ReadCnt/LikeCnt/CollectCnt/Liked/Collected）
type ArticleVO struct {
	Id         int64  `json:"id,omitempty"`
	Title      string `json:"title,omitempty"`
	Abstract   string `json:"abstract,omitempty"` // 摘要，列表接口返回（前100字）
	Content    string `json:"content,omitempty"`  // 完整内容，详情接口返回
	AuthorId   int64  `json:"authorId,omitempty"`
	AuthorName string `json:"authorName,omitempty"`
	Status     uint8  `json:"status,omitempty"` // 文章状态：0未知/1未发表/2已发表/3私密
	Ctime      string `json:"ctime,omitempty"`  // 创建时间（格式化字符串）
	Utime      string `json:"utime,omitempty"`  // 更新时间（格式化字符串）

	// 互动数据，仅 PubDetail 接口返回
	ReadCnt    int64 `json:"readCnt"`    // 阅读数
	LikeCnt    int64 `json:"likeCnt"`    // 点赞数
	CollectCnt int64 `json:"collectCnt"` // 收藏数
	Liked      bool  `json:"liked"`      // 当前用户是否已点赞
	Collected  bool  `json:"collected"`  // 当前用户是否已收藏
}

// toDomain 将请求结构体转换为领域模型，注入作者ID
// 作者ID 从 JWT claims 中获取，确保数据归属正确
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
