package domain

import "time"

// Article 文章领域模型
// 贯穿 Service 和 Repository 层，是业务核心数据结构
// 不包含数据库细节（如字段标签），保持业务纯粹性
type Article struct {
	Id      int64         // 文章ID
	Title   string        // 文章标题
	Content string        // 文章内容（完整内容，列表场景由上层截取摘要）
	Author  Author        // 文章作者
	Status  ArticleStatus // 文章状态（状态机：未发表/已发表/私密）
	Ctime   time.Time     // 创建时间
	Utime   time.Time     // 更新时间
}

// Abstract 返回文章摘要
// 考虑中文问题，按 rune 截取前100字（避免截断半个中文字符）
// 用于列表接口展示，避免传输完整内容
func (a Article) Abstract() string {
	cs := []rune(a.Content)
	if len(cs) < 100 {
		return a.Content
	}
	return string(cs[:100])
}

// Author 作者领域模型
type Author struct {
	Id   int64  // 作者ID（关联用户表）
	Name string // 作者名称（预留，当前未使用）
}

// ArticleStatus 文章状态常量
// 使用 iota 定义状态机，状态值从0开始递增
const (
	ArticleStatusUnknown     ArticleStatus = iota // 未知状态（默认值，不应出现）
	ArticleStatusUnPublished                      // 未发表状态（草稿）
	ArticleStatusPublished                        // 已发表状态（对外可见）
	ArticleStatusPrivate                          // 私密状态（仅作者可见）
)

// ArticleStatus 文章状态类型，基于uint8存储
// 使用 uint8 节省存储空间，仅占用1字节
type ArticleStatus uint8

// ToUint8 将文章状态转换为uint8类型，用于数据库存储和 JSON 响应
func (as ArticleStatus) ToUint8() uint8 {
	return uint8(as)
}

// Valid 判断文章状态是否为有效状态，当前仅校验未发表状态
// 用于 Withdraw 操作的校验（只有未发表状态才能撤回）
func (as ArticleStatus) Valid() bool {
	return as.ToUint8() == uint8(ArticleStatusUnPublished)
}

// IsNoPublished 判断文章是否为未发表状态
// 用于读者接口校验（未发表文章不应对外可见）
func (as ArticleStatus) IsNoPublished() bool {
	return as != ArticleStatusPublished
}
