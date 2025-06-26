package core

import (
	"bytes"
	"sync"
)

type CoroutinePool interface {
	Submit(task func()) error
	Release()
}

type BytePool struct{ p sync.Pool }

func NewBytePool(size, cap int) *BytePool {
	return &BytePool{
		p: sync.Pool{
			New: func() any {
				b := make([]byte, size, cap)
				return &b
			},
		},
	}
}

func (p *BytePool) Get() []byte {
	b := p.p.Get().(*[]byte)
	return *b
}

func (p *BytePool) Put(b []byte) {
	b = b[:0] // 重置已用长度
	p.p.Put(&b)
}

type ByteBufferPool struct {
	p sync.Pool
}

func NewByteBufferPool(size, cap int) *ByteBufferPool {
	return &ByteBufferPool{
		p: sync.Pool{
			New: func() any { return bytes.NewBuffer(make([]byte, size, cap)) },
		},
	}
}

func (p *ByteBufferPool) Get() *bytes.Buffer {
	b := p.p.Get().(*bytes.Buffer)
	b.Reset()
	return b
}

func (p *ByteBufferPool) Put(b *bytes.Buffer) {
	b.Reset() // 重置已用长度
	p.p.Put(b)
}
