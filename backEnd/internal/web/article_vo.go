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
	Id       int64  `json:"id"`
	Title    string `json:"title"`
	Abstract string `json:"abstract"`
	Content  string `json:"content"`
	Status   uint8  `json:"status"`
	Author   string `json:"author"`
	Ctime    string `json:"ctime"`
	Utime    string `json:"utime"`
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
