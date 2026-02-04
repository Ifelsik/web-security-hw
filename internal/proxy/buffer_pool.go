package proxy

import "sync"

const (
	bufferSize = 32 * 1024 // KB
)

var pool = &BytePool{
	pool: sync.Pool{
		New: func() any {
			b := make([]byte, bufferSize)
			return b
		},
	},
}

type BytePool struct {
	pool sync.Pool
}

func (bp *BytePool) Get() (b []byte) {
	ifce := bp.pool.Get()
	if ifce != nil {
		return ifce.([]byte)
	}
	return
}

func (bp *BytePool) Put(b []byte) {
	bp.pool.Put(b)
}
