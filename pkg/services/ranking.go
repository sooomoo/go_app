package services

import (
	"context"
	"goapp/pkg/cache"
	"strconv"

	"github.com/redis/go-redis/v9"
)

var (
	luaRelease = redis.NewScript(`
	local res = redis.call('SET', KEYS[1] .. '_' .. ARGV[1], '0', 'EX', '50', 'NX')
	if res ~= false then
		redis.call('ZINCRBY', KEYS[1], ARGV[2], KEYS[2])
	end
	return 0
	`)
)

// 排名服务
type RankingService struct {
	db   *cache.Cache // redis 客户端，用于存储排名信息
	name string       // 排名的名称
}

func NewRankingService(db *cache.Cache, name string) *RankingService {
	return &RankingService{db: db}
}

// 裁剪排名数据：仅仅只保留不超过指定数量的名次
//
// 当需要精确排名时，则不需要裁剪
func (r *RankingService) Truncate(size int) {

}

// 增加计数
//
// requestId: 用于幂等更新，防止多次更新
//
// id: 需要被排名的主体的 id：可以是用户 ID->积分排名、步数排名; 可以是作品 Id-> 热度排名(播放量，点赞，收藏等)
//
// score: 权重，增加此权重之后，会导致排名变化
func (r *RankingService) Increment(ctx context.Context, requestId string, id int64, score int64) error {
	_, err := luaRelease.Eval(ctx, r.db.Master(), []string{r.name, strconv.FormatInt(id, 10)}, requestId, strconv.FormatInt(score, 10)).Result()
	if err != nil {
		return err
	}
	// r.db.Master().ZAdd(ctx, r.name, redis.Z{Member: id, Score: float64(score)})
	return nil
}
