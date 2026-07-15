package domain

// Article 文章领域模型
type Article struct {
	Id      int64         // 文章ID
	Title   string        // 文章标题
	Content string        // 文章内容
	Author  Author        // 文章作者
	Status  ArticleStatus // 文章状态
}

// Author 作者领域模型
type Author struct {
	Id   int64  // 作者ID
	Name string // 作者名称
}

// ArticleStatus 文章状态常量
const (
	ArticleStatusUnknown     ArticleStatus = iota // 未知状态
	ArticleStatusUnPublished                      // 未发表状态
	ArticleStatusPublished                        // 已发表状态
	ArticleStatusPrivate                          // 私密状态
)

// ArticleStatus 文章状态类型，基于uint8存储
type ArticleStatus uint8

// ToUint8 将文章状态转换为uint8类型，用于数据库存储
func (as ArticleStatus) ToUint8() uint8 {
	return uint8(as)
}

// Valid 判断文章状态是否为有效状态，当前仅校验未发表状态
func (as ArticleStatus) Valid() bool {
	return as.ToUint8() == uint8(ArticleStatusUnPublished)
}

// IsNoPublished 判断文章是否为未发表状态
func (as ArticleStatus) IsNoPublished() bool {
	return as != ArticleStatusPublished
}
