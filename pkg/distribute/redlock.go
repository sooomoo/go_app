package distribute

import (
	"context"
	"errors"
	"fmt"
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
}

type RedLockerOption func(*RedLockerConfig)

func (r *RedLocker) Lock(ctx context.Context, mutexname string) (*RedLock, error) {
	mutex := r.redsync.NewMutex(mutexname)
	if err := mutex.LockContext(ctx); err != nil {
		return nil, err
	}

	// , options ...RedLockerOption
	// config := &RedLockerConfig{}
	// for _, v := range options {
	// 	v(config)
	// }

	return &RedLock{mutex: mutex}, nil
}

type RedLock struct {
	mutex *redsync.Mutex

	mut           sync.RWMutex
	cancelExtend  context.CancelFunc
	chExtendError chan error
}

// 调用了 AutoExtend 之后，必须在 Unlock 之前调用 CancelAutoExtend 取消自动续期
func (r *RedLock) AutoExtend(ctx context.Context, extendInterval time.Duration, maxTimes int) {
	r.mut.Lock()
	defer r.mut.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	r.cancelExtend = cancel
	r.chExtendError = make(chan error, 1)

	go func(ctx context.Context) {
		remainExtendTimes := maxTimes
		for {
			select {
			case <-ctx.Done():
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
				remainExtendTimes--
				if remainExtendTimes <= 0 {
					r.cancelExtend()
					return
				}
			}
		}
	}(ctx)
}

func (r *RedLock) AutoExtendError() <-chan error {
	return r.chExtendError
}

func (r *RedLock) CancelAutoExtend(waitUntilDone time.Duration) {
	r.mut.Lock()
	defer r.mut.Unlock()

	if r.cancelExtend != nil {
		r.cancelExtend()
		r.cancelExtend = nil
		time.Sleep(waitUntilDone)
	}
	if r.chExtendError != nil {
		close(r.chExtendError)
		r.chExtendError = nil
	}
}

func (r *RedLock) Extend(ctx context.Context) error {
	r.mut.RLock()
	defer r.mut.RUnlock()

	val, err := r.mutex.ExtendContext(ctx)
	if !val {
		return ErrFailed
	}
	return err
}

func (r *RedLock) Unlock(ctx context.Context) error {
	r.mut.Lock()
	defer r.mut.Unlock()
	defer func() {
		r.mutex = nil
	}()
	if r.mutex == nil {
		return nil
	}

	val, err := r.mutex.UnlockContext(ctx)
	fmt.Printf("RedLock unlock result -> val: %v, err: %v\n", val, err)

	if !val {
		return ErrFailed
	}

	return err
}
