package queue

import (
	"bytes"
	"testing"
)

func TestPriorityQueueCorrectness(t *testing.T) {
	pq := New()
	pq.Push(Item{Priority: 10, Value: []byte("low")})
	pq.Push(Item{Priority: 1, Value: []byte("highest")})
	pq.Push(Item{Priority: 5, Value: []byte("medium")})

	it1, ok := pq.Pop()
	if !ok || it1.Priority != 1 || string(it1.Value) != "highest" {
		t.Fatalf("expected highest priority item first, got %+v", it1)
	}

	it2, ok := pq.Pop()
	if !ok || it2.Priority != 5 {
		t.Fatalf("expected priority 5, got %+v", it2)
	}

	it3, ok := pq.Pop()
	if !ok || it3.Priority != 10 {
		t.Fatalf("expected priority 10, got %+v", it3)
	}
}

func TestBulkEnqueueWithBufferReuse(t *testing.T) {
	pq := New()

	// Simulate buffer reuse as identified in Lesson L-017
	sharedBuf := bytes.NewBuffer(make([]byte, 0, 1024))
	sharedBuf.WriteString("shared message")

	// Enqueue slice from buffer
	pq.BulkEnqueue([]Item{
		{Priority: 2, Value: sharedBuf.Bytes()},
	})

	// Caller immediately mutates or resets the shared buffer
	sharedBuf.Reset()
	sharedBuf.WriteString("OVERWRITTEN_DATA")

	// Pop item: value MUST remain original "shared message", protected by safe cloning
	item, ok := pq.Pop()
	if !ok {
		t.Fatalf("expected item in queue")
	}
	if string(item.Value) != "shared message" {
		t.Fatalf("memory corruption detected! expected 'shared message', got %q", string(item.Value))
	}
}

func BenchmarkArrayHeapPushPop(b *testing.B) {
	pq := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pq.Push(Item{Priority: i % 100, Value: []byte("data")})
		if i%2 == 0 {
			pq.Pop()
		}
	}
}
