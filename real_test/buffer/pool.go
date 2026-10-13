package buffer

import (
	"bytes"
	"sync"
)

type Pool struct {
	p sync.Pool
}

func NewPool() *Pool {
	return &Pool{
		p: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

func (p *Pool) Get() *bytes.Buffer {
	buf := p.p.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func (p *Pool) Put(buf *bytes.Buffer) {
	buf.Reset()
	p.p.Put(buf)
}
