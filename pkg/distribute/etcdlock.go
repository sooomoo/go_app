package distribute

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var errClientNotInit = errors.New("client not inited")

type EtcdLocker struct {
	localMut sync.RWMutex

	client *clientv3.Client // etcd 客户端
	// leaseID    clientv3.LeaseID     // 租约 ID
}

// NewEtcdLocker 创建分布式锁实例
func NewEtcdLocker(client *clientv3.Client) *EtcdLocker {
	return &EtcdLocker{
		localMut: sync.RWMutex{},
		client:   client,
	}
}

func (l *EtcdLocker) Close() error {
	l.localMut.Lock()
	defer l.localMut.Unlock()

	if l.client != nil {
		return l.client.Close()
	}
	l.client = nil
	return nil
}

type EtcdLockConfig struct {
	TTL         int           // configures the session's TTL in seconds. If TTL is <= 0, the default 60 seconds TTL will be used.
	LockTimeout time.Duration // configures the lock's timeout. If LockTimeout is <= 0, the default 8 seconds timeout will be used.
}

func (e *EtcdLockConfig) useDefaultIfNotSepecified() {
	if e.TTL <= 0 {
		e.TTL = 60
	}
	if e.LockTimeout <= 0 {
		e.LockTimeout = 8 * time.Second
	}
}

type EtcdLockOption func(*EtcdLockConfig)

func WithTTL(ttl int) EtcdLockOption {
	return func(config *EtcdLockConfig) {
		config.TTL = ttl
	}
}
func WithLockTimeout(timeout time.Duration) EtcdLockOption {
	return func(config *EtcdLockConfig) {
		config.LockTimeout = timeout
	}
}

// Lock 尝试获取锁
// ttl: 锁的存活时间（秒）
func (l *EtcdLocker) Lock(ctx context.Context, key string, options ...EtcdLockOption) (*EtcdLock, error) {
	l.localMut.RLock()
	defer l.localMut.RUnlock()

	if l.client == nil {
		return nil, errClientNotInit
	}

	config := EtcdLockConfig{}
	for _, v := range options {
		v(&config)
	}
	config.useDefaultIfNotSepecified()

	// 1. 创建会话并绑定租约
	session, err := concurrency.NewSession(l.client, concurrency.WithTTL(config.TTL))
	if err != nil {
		return nil, fmt.Errorf("创建会话失败: %v", err)
	}

	// 2. 创建互斥锁
	mutex := concurrency.NewMutex(session, key)

	// 3. 获取锁（带超时控制）
	ctx, cancel := context.WithTimeout(ctx, config.LockTimeout)
	defer cancel()
	if err := mutex.Lock(ctx); err != nil {
		session.Close()
		return nil, fmt.Errorf("获取锁失败: %v", err)
	}

	return &EtcdLock{
		key:     key,
		mutex:   mutex,
		session: session,
	}, nil
}

type EtcdLock struct {
	key     string
	mutex   *concurrency.Mutex
	session *concurrency.Session

	mut sync.Mutex
}

// Unlock 释放锁
func (l *EtcdLock) Unlock(ctx context.Context) error {
	l.mut.Lock()
	defer l.mut.Unlock()
	defer func() {
		l.mutex = nil
		l.session = nil
	}()

	if l.mutex != nil {
		if err := l.mutex.Unlock(ctx); err != nil {
			return fmt.Errorf("释放锁失败: %v", err)
		}
	}

	if l.session != nil {
		l.session.Close()
	}

	return nil
}
