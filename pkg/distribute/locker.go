package distribute

import (
	"context"
	"errors"
	"fmt"
	"goapp/pkg/core"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

type Locker struct {
	mutex                sync.RWMutex
	redisClient          *redis.Client
	defaultTtl           time.Duration
	defaultRetryStrategy RetryStrategy
	defaultLockTimeout   time.Duration
}

type LockerOption func(*Locker)

func WithDefaultTtl(ttl time.Duration) LockerOption {
	return func(opt *Locker) {
		opt.defaultTtl = ttl
	}
}
func WithDefaultRetryStrategy(retryStrategy RetryStrategy) LockerOption {
	return func(opt *Locker) {
		opt.defaultRetryStrategy = retryStrategy
	}
}
func WithDefaultLockTimeout(lockTimeout time.Duration) LockerOption {
	return func(opt *Locker) {
		opt.defaultLockTimeout = lockTimeout
	}
}

// NewLocker creates a new Locker.
//
// defaultTTL 8s; defaultLockTimeout 60s; defaultRetryStrategy LinearRetryStrategy(100 * time.Millisecond)
func NewLocker(ctx context.Context, opt *redis.Options, options ...LockerOption) (*Locker, error) {
	client := redis.NewClient(opt)
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}
	l := &Locker{
		mutex:       sync.RWMutex{},
		redisClient: client,
	}
	for _, v := range options {
		v(l)
	}
	if l.defaultTtl <= 0 {
		l.defaultTtl = 8 * time.Second
	}
	if l.defaultLockTimeout <= 0 {
		l.defaultLockTimeout = 60 * time.Second
	}
	if l.defaultRetryStrategy == nil {
		l.defaultRetryStrategy = LinearRetryStrategy(100 * time.Millisecond)
	}
	return l, nil
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

type LockOptions struct {
	Resource          string
	Owner             string
	Ttl               time.Duration
	RetryStrategy     RetryStrategy
	lockTimeout       time.Duration
	disableAutoExtend bool
}

type LockOption func(*LockOptions)

func LockWithTtl(ttl time.Duration) LockOption {
	return func(opt *LockOptions) {
		opt.Ttl = ttl
	}
}
func LockWithRetryStrategy(retryStrategy RetryStrategy) LockOption {
	return func(opt *LockOptions) {
		opt.RetryStrategy = retryStrategy
	}
}
func LockWithOwner(owner string) LockOption {
	return func(opt *LockOptions) {
		opt.Owner = owner
	}
}
func LockWithLockTimeout(lockTimeout time.Duration) LockOption {
	return func(opt *LockOptions) {
		opt.lockTimeout = lockTimeout
	}
}
func LockWithDisableAutoExtend() LockOption {
	return func(opt *LockOptions) {
		opt.disableAutoExtend = true
	}
}

func (l *Locker) Lock(ctx context.Context, resource string, options ...LockOption) (*Lock, error) {
	opt := &LockOptions{Resource: resource}
	for _, v := range options {
		v(opt)
	}
	if opt.Ttl <= 0 {
		opt.Ttl = l.defaultTtl
	}
	if opt.RetryStrategy == nil {
		opt.RetryStrategy = l.defaultRetryStrategy
	}
	if opt.lockTimeout <= 0 {
		opt.lockTimeout = l.defaultLockTimeout
	}
	opt.Owner = strings.TrimSpace(opt.Owner)
	if len(opt.Owner) == 0 {
		opt.Owner = core.NewSeqID().Hex()
	}
	return l.lockWithOptions(ctx, opt)
}

func (l *Locker) lockWithOptions(ctx context.Context, opt *LockOptions) (*Lock, error) {
	// make sure we don't retry forever
	ctx, cancel := context.WithTimeout(ctx, opt.lockTimeout)
	defer cancel()

	minDur := 10 * time.Millisecond
	for {
		l.mutex.RLock()
		client := l.redisClient
		l.mutex.RUnlock()

		if client == nil {
			return nil, ErrLockFailed
		}

		ok, err := client.SetNX(ctx, opt.Resource, opt.Owner, opt.Ttl).Result()
		if err != nil {
			return nil, err
		} else if ok {
			lock := &Lock{
				client:   client,
				resource: opt.Resource,
				owner:    opt.Owner,
				ttl:      opt.Ttl,
			}
			if !opt.disableAutoExtend {
				lock.autoExtend(context.Background(), opt.Ttl/3*2)
			}
			return lock, nil
		}

		// retry
		backoff := max(opt.RetryStrategy.Next(), minDur)

		select {
		case <-ctx.Done():
			return nil, ErrLockFailed
		case <-time.After(backoff):
			fmt.Println("retrying...")
		}
	}
}

type Lock struct {
	client   *redis.Client
	resource string
	owner    string
	ttl      time.Duration

	released             atomic.Bool
	mut                  sync.RWMutex
	autoExtendCancelFunc context.CancelFunc
	chCancelDone         chan core.Empty
}

func (l *Lock) autoExtend(ctx context.Context, extendInterval time.Duration) {
	l.mut.Lock()
	defer l.mut.Unlock()

	ctx, l.autoExtendCancelFunc = context.WithCancel(ctx)
	l.chCancelDone = make(chan core.Empty, 1)
	go func(ctx context.Context) {
		defer func() {
			l.chCancelDone <- core.Empty{}
		}()
		for {
			if l.released.Load() {
				fmt.Println("auto extend: lock released")
				return
			}

			select {
			case <-time.After(extendInterval):
				err := l.Extend(ctx)
				fmt.Println("auto extend: extended")
				if err != nil {
					fmt.Println("auto extend:extend lock failed:", err)
					return
				}
				if l.released.Load() {
					fmt.Println("auto extend:lock released")
					return
				}
			case <-ctx.Done():
				fmt.Println("auto extend: autoextend done")
				return
			}
		}
	}(ctx)
}

// 续期锁
func (l *Lock) Extend(ctx context.Context) error {
	if l.released.Load() {
		return ErrLockNotHeld
	}

	ttlVal := strconv.FormatInt(int64(l.ttl/time.Millisecond), 10)
	status, err := luaRefresh.Run(ctx, l.client, []string{l.resource}, l.owner, ttlVal).Result()
	if err != nil {
		return err
	} else if status == int64(1) {
		return nil
	}
	return ErrLockNotHeld
}

// 释放获取的锁
func (l *Lock) Unlock(ctx context.Context) error {
	l.mut.Lock()
	defer l.mut.Unlock()
	defer func() {
		l.chCancelDone = nil
		l.autoExtendCancelFunc = nil
	}()

	if l.released.Load() {
		fmt.Println("lock already released")
		return nil
	}
	l.released.Store(true)

	fmt.Println("unlocking...")

	if l.autoExtendCancelFunc != nil {
		l.autoExtendCancelFunc()
		// 等待自动续期协程退出
		fmt.Println("waiting auto extend goroutine exit...")
		select {
		case <-l.chCancelDone:
			fmt.Println("auto extend goroutine exited")
		case <-time.After(1 * time.Second):
			fmt.Println("auto extend goroutine exit Timeout")
		}
	}
	fmt.Println("unlock redis lock...")
	res, err := luaRelease.Run(ctx, l.client, []string{l.resource}, l.owner).Result()
	fmt.Printf("unlock redis lock result: %v, err: %v\n", res, err)
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
