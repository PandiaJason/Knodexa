package buffer

import (
	"testing"
)

func TestBufferPoolCleanliness(t *testing.T) {
	pool := NewPool()
	buf := pool.Get()
	buf.WriteString("hello secret data")
	pool.Put(buf)

	buf2 := pool.Get()
	if buf2.Len() != 0 {
		t.Fatalf("dirty buffer leaked from pool! len=%d", buf2.Len())
	}
}
