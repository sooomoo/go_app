package core

import (
	"bytes"
	"sync"
)

type CoroutinePool interface {
	Submit(task func()) error
	Release()
}

type Resetter interface {
	Reset()
}

type ByteBufferPool struct {
	pool sync.Pool
}

func NewByteBufferPool(size, cap int) *ByteBufferPool {
	return &ByteBufferPool{
		pool: sync.Pool{
			New: func() any { return bytes.NewBuffer(make([]byte, size, cap)) },
		},
	}
}

func (p *ByteBufferPool) Get() *bytes.Buffer {
	b := p.pool.Get().(*bytes.Buffer)
	return b
}

// 回收对象，同时会调用 Reset 方法重置对象。
// 以便于GC回收相关引用的对象
func (p *ByteBufferPool) Put(b *bytes.Buffer) {
	b.Reset() // 重置已用长度
	p.pool.Put(b)
}

type ObjectPool[T Resetter] struct {
	pool sync.Pool
}

func NewObjectPool[T Resetter](newFunc func() T) *ObjectPool[T] {
	return &ObjectPool[T]{
		pool: sync.Pool{
			New: func() any { return newFunc() },
		},
	}
}

// 获取对象
func (p *ObjectPool[T]) Get() T {
	obj := p.pool.Get().(T)
	return obj
}

// 回收对象，同时会调用 Resetter 的 Reset 方法重置对象。
// 以便于GC回收相关引用的对象
func (p *ObjectPool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}
