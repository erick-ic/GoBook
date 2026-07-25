package domain

// Interactive 互动数据领域模型
// 记录某个业务实体（如文章）的阅读数、点赞数、收藏数等聚合数据
// 通过 biz+bizId 唯一标识一个业务实体的互动数据（支持多业务复用）
type Interactive struct {
	BizId      int64 // 业务实体ID（如文章ID）
	ReadCnt    int64 // 阅读数
	LikeCnt    int64 // 点赞数
	CollectCnt int64 // 收藏数
}
