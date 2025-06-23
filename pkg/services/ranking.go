package services

import (
	"context"
	"fmt"
	"goapp/pkg/cache"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	luaIncr = redis.NewScript(`
	local res = redis.call('SET', KEYS[1], '0', 'EX', ARGV[3], 'NX')
	if res ~= false then
		return redis.call('ZADD', KEYS[2], ARGV[2], ARGV[1])
	end
	return 0
	`)
	luaTruncate = redis.NewScript(`
	local elements = redis.call('ZRANGE', KEYS[1], '+inf', '-inf','BYSCORE', 'REV', 'LIMIT', ARGV[1], ARGV[2]); 
	if #elements > 0 then 
		return redis.call('ZREM', KEYS[1], unpack(elements)); 
	end
	return 0; 
	`)
)

var (
	rankingTimeFuture = time.Date(2050, 0, 0, 0, 0, 0, 0, time.UTC)
)

// 排名服务
type RankingService struct {
	db  *cache.Cache // redis 客户端，用于存储排名信息
	key string       // 排名的名称
}

func NewRankingService(db *cache.Cache, key string) *RankingService {
	return &RankingService{db: db, key: key}
}

// 裁剪排名数据：仅仅只保留不超过指定数量的名次（当需要精确排名时，则不需要裁剪）
//
// 【默认】分批次删除，当数据量很大时，分批删除，可以防止 redis 响应延迟
//
// 如果正确删除，则返回被删除的条目的数量
//
// size：删除多少名之后的数据
//
// batchSize: 每批次删除多少的数据
//
// delayEachBatch：删除一个批次后，休息多长时间再删除下一个批次；不能设置为 0，要给 redis 休息时间
func (r *RankingService) Truncate(ctx context.Context, size int64, batchSize int64, delayEachBatch time.Duration) (int64, error) {
	fn := func() (int64, error) {
		val, err := luaTruncate.Eval(ctx, r.db.Master(), []string{r.key}, size, batchSize).Result()
		if err != nil {
			return -1, err
		}
		if v, ok := val.(int64); ok {
			return v, nil
		}
		return 0, nil
	}
	if batchSize <= 0 {
		return fn()
	}
	count := int64(0)
	for {
		cnt, err := fn()
		if err != nil {
			return cnt, err
		}
		if cnt == 0 {
			break
		}
		count += cnt
		time.Sleep(delayEachBatch)
	}
	return count, nil
}

// 一次性删除所有（当需要精确排名时，不需要裁剪）
//
// 当删除数量很大时，可能导致 redis 长时间阻塞
func (r *RankingService) TruncateAll(ctx context.Context, size int64) (int64, error) {
	val, err := luaTruncate.Eval(ctx, r.db.Master(), []string{r.key}, size, -1).Result()
	if err != nil {
		return -1, err
	}
	if v, ok := val.(int64); ok {
		return v, nil
	}
	return 0, nil
}

type RankingItem struct {
	Member string
	Score  int64
	Frac   float64
}

// 分批获取排名
//
// offset：开始位置，结果会包含此位置的值
//
// limit：此批次获取多少条记录
func (r *RankingService) Paginate(ctx context.Context, offset, limit int64) ([]RankingItem, error) {
	vals, err := r.db.Master().ZRangeArgsWithScores(ctx, redis.ZRangeArgs{
		Key:     r.key,
		Start:   0,
		Stop:    math.MaxInt64,
		Offset:  offset,
		Count:   limit,
		ByScore: true,
		Rev:     true,
	}).Result()
	if err != nil {
		return nil, err
	}
	arr := make([]RankingItem, len(vals))
	for i, v := range vals {
		mem, ok := v.Member.(string)
		if !ok {
			continue
		}
		// 分离整数和小数部分
		fscore, frac := math.Modf(v.Score)
		arr[i] = RankingItem{Member: mem, Score: int64(fscore), Frac: frac}
	}
	return arr, nil
}

// 获取指定成员的精确排名
func (r *RankingService) Rank(ctx context.Context, member string) int64 {
	val, err := r.db.Slave().ZRevRank(ctx, r.key, member).Result()
	if err != nil {
		return -1
	}
	return val
}

// 更新计数
//
// requestId: 用于幂等更新，防止多次更新；idempotentExpSecs：用于设置幂等 key 的过期时间
//
// member: 需要被排名的主体的 id：可以是用户 ID->积分排名、步数排名; 可以是作品 Id-> 热度排名(播放量，点赞，收藏等)
//
// score: 权重，更新此权重之后，会导致排名变化
func (r *RankingService) UpdateScore(ctx context.Context, requestId string, member string, score int64, idempotentExpSecs int64) error {
	idempotentKey := fmt.Sprintf("%s:idempotent:%s", r.key, requestId)
	timestamp := int64(time.Until(rankingTimeFuture).Seconds())
	frac, err := strconv.ParseFloat(fmt.Sprintf("0.%d", timestamp), 64)
	if err != nil {
		return err
	}
	finalScore := float64(score) + frac
	// 使用 Lua 脚本来保证原子性
	_, err = luaIncr.Eval(ctx, r.db.Master(), []string{idempotentKey, r.key}, member, finalScore, idempotentExpSecs).Result()
	if err != nil {
		return err
	}
	return nil
}

// 模糊排名统计数据: rangeStart 和 rangeEnd 表示积分范围，Count 表示此范围内的成员数量
//
// rangeStart <= memberScore < rangeEnd
type RankingFuzzyCount struct {
	RangeStart int64
	RangeEnd   int64
	Count      int64
}

// 当更新了积分之后，如果需要模糊排名，则需要通过此方法更新每一个积分范围内的（用户、作品等）数量
//
// 各个范围不能重叠
//
// 这可以在 Increment 之后，发布更新命令到消息队列，由专门的服务来统计数量。
// 为了高并发，该服务可以合并多个请求（如收到请求后开启一个 timer，2s 后才更新数量）
func (r *RankingService) FuzzySet(ctx context.Context, rangesCount []RankingFuzzyCount) error {
	key := fmt.Sprintf("%s:fuzzy", r.key)
	vals := make(map[string]any)
	for _, v := range rangesCount {
		field := fmt.Sprintf("%d-%d", v.RangeStart, v.RangeEnd)
		vals[field] = v.Count
	}
	_, err := r.db.HMSet(ctx, key, vals)
	return err
}

// 获取模糊排名的统计数据
//
// 返回的结果是一个数组，每个元素表示一个积分范围内的数量
func (r *RankingService) FuzzyGet(ctx context.Context) ([]RankingFuzzyCount, error) {
	key := fmt.Sprintf("%s:fuzzy", r.key)
	vals, err := r.db.HGetAll(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(vals) == 0 {
		return nil, nil
	}
	arr := make([]RankingFuzzyCount, 0, len(vals))
	for field, count := range vals {
		var start, end int64
		_, err := fmt.Sscanf(field, "%d-%d", &start, &end)
		if err != nil {
			return nil, err
		}
		c, err := strconv.ParseInt(count, 10, 64)
		if err != nil {
			return nil, err
		}
		arr = append(arr, RankingFuzzyCount{
			RangeStart: start,
			RangeEnd:   end,
			Count:      c,
		})
	}
	return arr, nil
}

// 获取成员的大致排名
//
// member：需要获取模糊排名的成员 ID
//
// memberScore：成员的积分
//
// accurateRankSize：需要获取的精确排名的数量
func (r *RankingService) FuzzyRank(ctx context.Context, member string, memberScore int64, accurateRankSize int64) int64 {
	// 首先获取精确排名
	concretRank := r.Rank(ctx, member)
	if concretRank >= 0 && concretRank < accurateRankSize {
		return concretRank
	}
	// 需要计算模糊排名
	ranges, err := r.FuzzyGet(ctx)
	if err != nil {
		return -1
	}
	if len(ranges) == 0 {
		return -1
	}
	// 按照积分范围进行排序：由高到低
	slices.SortFunc(ranges, func(a, b RankingFuzzyCount) int {
		return int(b.RangeStart - a.RangeStart)
	})
	// 遍历积分范围，找到成员所在的范围
	rank := int64(0)
	for _, v := range ranges {
		if memberScore >= v.RangeStart && memberScore < v.RangeEnd {
			// 找到范围后，计算大致排名
			rangeRank := math.Floor(float64(v.RangeEnd-memberScore) / float64(v.RangeEnd-v.RangeStart))
			return rank + int64(rangeRank)
		} else {
			rank += v.Count
		}
	}
	return rank
}
