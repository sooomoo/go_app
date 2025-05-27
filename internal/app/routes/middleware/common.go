package middleware

import (
	"goapp/pkg/core"
)

var bufferPool *core.ByteBufferPool = core.NewByteBufferPool(0, 1024)
