package distribute

import (
	"context"
	"errors"
	"fmt"
	"goapp/pkg/core"
	"math/rand"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	goredislib "github.com/redis/go-redis/v9"
)

var ErrFailed = errors.New("failed")

// 需要支持的功能：
// 1. 自动续期, 此时需要设置锁的最大持有时间，防止一直续期
type RedLocker struct {
	client  *goredislib.Client
	pool    redis.Pool
	redsync *redsync.Redsync
}

func NewRedLocker(addr string) *RedLocker {
	client := goredislib.NewClient(&goredislib.Options{
		Addr: addr,
	})
	pool := goredis.NewPool(client)
	redsync := redsync.New(pool)
	return &RedLocker{
		client:  client,
		pool:    pool,
		redsync: redsync,
	}
}

type RedLockerConfig struct {
	TTL                time.Duration // 锁的存活时间（秒）：默认 8s
	AcquireTries       int           // 尝试获取锁的次数：默认 32
	RetryDelay         time.Duration // 重试间隔：默认 rand(50ms, 250ms)
	AutoExtendLock     bool          // 是否自动续期：默认 false
	AutoExtendInterval time.Duration // 自动续期间隔：默认 TTL * 0.67
}

func (c *RedLockerConfig) useDefaultIfNotSepecified() {
	if c.AcquireTries <= 0 {
		c.AcquireTries = 32
	}
	if c.TTL <= 0 {
		c.TTL = 8 * time.Second
	}
	if c.RetryDelay <= 0 {
		c.RetryDelay = time.Duration(time.Duration(50+rand.Intn(200)) * time.Millisecond)
	}
	if c.AutoExtendLock {
		if c.AutoExtendInterval <= 0 {
			c.AutoExtendInterval = time.Duration(float64(c.TTL.Milliseconds())*0.67) * time.Millisecond
		}
	}
}

type RedLockerOption func(*RedLockerConfig)

func RedLockWithTTL(ttl time.Duration) RedLockerOption {
	return func(config *RedLockerConfig) {
		config.TTL = ttl
	}
}
func RedLockWithAcquireTries(tries int) RedLockerOption {
	return func(config *RedLockerConfig) {
		config.AcquireTries = tries
	}
}
func RedLockWithRetryDelay(retryDelay time.Duration) RedLockerOption {
	return func(config *RedLockerConfig) {
		config.RetryDelay = retryDelay
	}
}
func RedLockWithAutoExtend(extendInterval time.Duration) RedLockerOption {
	return func(config *RedLockerConfig) {
		config.AutoExtendLock = true
		config.AutoExtendInterval = extendInterval
	}
}

func (r *RedLocker) Lock(ctx context.Context, mutexname string, options ...RedLockerOption) (*RedLock, error) {
	config := &RedLockerConfig{}
	for _, v := range options {
		v(config)
	}
	config.useDefaultIfNotSepecified()

	mutex := r.redsync.NewMutex(mutexname, redsync.WithExpiry(config.TTL), redsync.WithTries(config.AcquireTries), redsync.WithRetryDelay(config.RetryDelay))
	if err := mutex.LockContext(ctx); err != nil {
		return nil, err
	}
	lock := &RedLock{mutex: mutex, mut: sync.RWMutex{}}
	if config.AutoExtendLock {
		lock.autoExtend(ctx, config.AutoExtendInterval)
	}

	return lock, nil
}

type RedLock struct {
	mutex *redsync.Mutex

	mut              sync.RWMutex
	cancelExtend     context.CancelFunc
	chExtendError    chan error
	chAutoExtendDone chan core.Empty
}

// 调用了 AutoExtend 之后，必须在 Unlock 之前调用 CancelAutoExtend 取消自动续期
func (r *RedLock) autoExtend(ctx context.Context, extendInterval time.Duration) {
	r.mut.Lock()
	defer r.mut.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	r.cancelExtend = cancel
	r.chExtendError = make(chan error, 1)
	r.chAutoExtendDone = make(chan core.Empty, 1)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				r.chAutoExtendDone <- core.Empty{}
				return
			case <-time.After(extendInterval):
				val, err := r.mutex.ExtendContext(ctx)
				if !val {
					// 说明锁已经失效了
					r.chExtendError <- ErrFailed
					return
				}
				if err != nil {
					r.chExtendError <- err
					return
				}
			}
		}
	}(ctx)
}

func (r *RedLock) AutoExtendError() <-chan error {
	return r.chExtendError
}

// 释放锁定的资源
func (r *RedLock) Unlock(ctx context.Context) error {
	r.mut.Lock()
	defer r.mut.Unlock()
	defer func() {
		r.mutex = nil
		r.cancelExtend = nil
		r.chAutoExtendDone = nil
		r.chExtendError = nil
	}()
	if r.mutex == nil {
		return nil
	}
	if r.cancelExtend != nil {
		r.cancelExtend()
		<-r.chAutoExtendDone // 等待自动续期协程退出
	}
	if r.chAutoExtendDone != nil {
		close(r.chAutoExtendDone)
	}
	if r.chExtendError != nil {
		close(r.chExtendError)
	}

	val, err := r.mutex.UnlockContext(ctx)
	fmt.Printf("RedLock unlock result -> val: %v, err: %v\n", val, err)

	if !val {
		return ErrFailed
	}

	return err
}
