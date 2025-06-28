package services_test

import (
	"context"
	"fmt"
	"goapp/pkg/cache"
	"goapp/pkg/core"
	"goapp/pkg/services"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestRankingSvcTruncate(t *testing.T) {
	ctx := context.Background()
	cache, err := cache.NewCacheWithAddr(ctx, "", "")
	if err != nil {
		t.Error(err)
		return
	}

	// lua := redis.NewScript(`
	// return redis.call('SET', KEYS[1], '0', 'EX', ARGV[4], 'NX')
	// `)
	// idempotentKey := fmt.Sprintf("%s:idempotent:%s", "ranking_test", "00de75693780008b2c2c77")
	// v, err := lua.Eval(ctx, cache.Master(), []string{idempotentKey}, 0, 0, 0, 600).Result()
	// if err != nil {
	// 	t.Error(err)
	// 	return
	// }
	// fmt.Println(v)
	svc := services.NewRankingService(cache, "ranking_test")

	arr, err := svc.Paginate(ctx, 4, 4)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(arr)
	r := svc.FuzzyRank(ctx, "item_19", 44, 10)
	fmt.Println(r)

	svc.UpdateScore(ctx, "00de75693780008b2c2c77", fmt.Sprintf("item_%v", 6), 2345, 600)
	wg := sync.WaitGroup{}
	count := 20
	wg.Add(count)
	for i := range count {
		go func(i int) {
			defer wg.Done()
			time.Sleep(time.Duration(rand.Float32()) * 5 * time.Second)
			svc.UpdateScore(ctx, core.NewSeqID().Hex(), fmt.Sprintf("item_%v", i), int64(rand.Intn(10000000)), 60)
		}(i)
	}
	wg.Wait()
	// deleted, _ := svc.Truncate(ctx, 4, 2, time.Second)
	deleted, _ := svc.TruncateAll(ctx, 4)
	fmt.Println("deleted:", deleted)
}
