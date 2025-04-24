package distribute

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrLockFailed  = errors.New("lock failed")
	ErrLockNotHeld = errors.New("lock not held")
)

var (
	luaRefresh = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("pexpire", KEYS[1], ARGV[2]) else return 0 end`)
	luaRelease = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end`)
	// luaPTTL    = redis.NewScript(`if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("pttl", KEYS[1]) else return -3 end`)
)

type LockOptions struct {
	Resource      string
	Owner         string
	Ttl           time.Duration
	RetryStrategy RetryStrategy
}

type Locker struct {
	mutex                sync.RWMutex
	redisClient          *redis.Client
	defaultTtl           time.Duration
	defaultRetryStrategy RetryStrategy
}

func NewLocker(ctx context.Context, opt *redis.Options, ttl time.Duration, retryStrategy RetryStrategy) (*Locker, error) {
	client := redis.NewClient(opt)
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}
	return &Locker{mutex: sync.RWMutex{}, redisClient: client, defaultTtl: ttl, defaultRetryStrategy: retryStrategy}, nil
}

func (l *Locker) Close() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.redisClient == nil {
		return
	}
	l.redisClient.Close()
	l.redisClient = nil
}

func (l *Locker) Lock(ctx context.Context, resource string, owner string) (*Lock, error) {
	return l.LockWithOptions(ctx, &LockOptions{
		Resource:      resource,
		Owner:         owner,
		Ttl:           l.defaultTtl,
		RetryStrategy: l.defaultRetryStrategy,
	})
}

// 在指定资源上加锁，默认5s
func (l *Locker) LockWithOptions(ctx context.Context, opt *LockOptions) (*Lock, error) {
	ttl := l.defaultTtl

	if opt.Ttl > 0 {
		ttl = opt.Ttl
	}
	retryStrategy := l.defaultRetryStrategy
	if opt.RetryStrategy != nil {
		retryStrategy = opt.RetryStrategy
	}
	// make sure we don't retry forever
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(ttl))
		defer cancel()
	}

	for {
		l.mutex.RLock()
		client := l.redisClient
		l.mutex.RUnlock()

		if client == nil {
			return nil, ErrLockFailed
		}

		ok, err := client.SetNX(ctx, opt.Resource, opt.Owner, ttl).Result()
		if err != nil {
			return nil, err
		} else if ok {
			return &Lock{client, opt.Resource, opt.Owner}, nil
		}

		// retry
		backoff := retryStrategy.Next()
		if backoff <= time.Duration(0) {
			return nil, ErrLockFailed
		}
		delay := time.After(backoff)

		select {
		case <-ctx.Done():
			return nil, ErrLockFailed
		case <-delay:
		}
	}
}

type Lock struct {
	client   *redis.Client
	resource string
	owner    string
}

func (i *Lock) Refresh(ctx context.Context, ttl time.Duration) error {
	if i == nil {
		return nil
	}
	ttlVal := strconv.FormatInt(int64(ttl/time.Millisecond), 10)
	status, err := luaRefresh.Run(ctx, i.client, []string{i.resource}, i.owner, ttlVal).Result()
	if err != nil {
		return err
	} else if status == int64(1) {
		return nil
	}
	return ErrLockNotHeld
}

// 释放获取的锁
func (i *Lock) Release(ctx context.Context) error {
	if i == nil {
		return nil
	}
	res, err := luaRelease.Run(ctx, i.client, []string{i.resource}, i.owner).Result()
	if err == redis.Nil {
		return ErrLockNotHeld
	} else if err != nil {
		return err
	}

	if i, ok := res.(int64); !ok || i != 1 {
		return ErrLockNotHeld
	}
	return nil
}
