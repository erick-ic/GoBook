// VO 展示给前端的数据
package web

import "GoBook/internal/domain"

// ArticleReq 文章请求结构体
type ArticleReq struct {
	Id      int64  `json:"id"`      // 文章ID，新增时为0
	Title   string `json:"title"`   // 文章标题
	Content string `json:"content"` // 文章内容
}

type ListReq struct {
	offset int `form:"offset"`
	limit  int `form:"limit"`
}

type ArticleVO struct {
	Id         int64  `json:"id,omitempty"`
	Title      string `json:"title,omitempty"`
	Abstract   string `json:"abstract,omitempty"`
	Content    string `json:"content,omitempty"`
	AuthorId   int64  `json:"authorId,omitempty"`
	AuthorName string `json:"authorName,omitempty"`
	Status     uint8  `json:"status,omitempty"`
	Ctime      string `json:"ctime,omitempty"`
	Utime      string `json:"utime,omitempty"`

	ReadCnt    int64 `json:"readCnt"`
	LikeCnt    int64 `json:"likeCnt"`
	CollectCnt int64 `json:"collectCnt"`
	Liked      bool  `json:"liked"`
	Collected  bool  `json:"collected"`
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
