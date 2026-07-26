package domain

// Interactive 互动数据领域模型
// 记录某个业务实体（如文章）的阅读数、点赞数、收藏数等聚合数据
// 通过 biz+bizId 唯一标识一个业务实体的互动数据（支持多业务复用）
type Interactive struct {
	Biz   string `json:"biz"`    // 业务场景
	BizId int64  `json:"biz_id"` // 业务实体ID（如文章ID）

	ReadCnt    int64 `json:"read_cnt"`    // 阅读数
	LikeCnt    int64 `json:"like_cnt"`    // 点赞数
	CollectCnt int64 `json:"collect_cnt"` // 收藏数

	Liked     bool `json:"liked"`     // 当前用户是否已点赞
	Collected bool `json:"collected"` // 当前用户是否已收藏
}
