package queue

import (
	"sync"
)

type Item struct {
	Priority int
	Value    []byte
}

type PriorityQueue struct {
	mu    sync.Mutex
	items []Item
}

func New() *PriorityQueue {
	return &PriorityQueue{
		items: make([]Item, 0, 64),
	}
}

func (pq *PriorityQueue) Push(it Item) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.items = append(pq.items, it)
	pq.up(len(pq.items) - 1)
}

// BulkEnqueue implements high-throughput batch insertion preserving the O(log N) heap invariant
// and adhering to Lesson L-017 (safe copying of pooled buffer slices).
func (pq *PriorityQueue) BulkEnqueue(items []Item) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	for _, it := range items {
		// Clone value slice to prevent data race corruption if caller returns buffer to pool (Lesson L-017)
		valCopy := make([]byte, len(it.Value))
		copy(valCopy, it.Value)
		it.Value = valCopy

		pq.items = append(pq.items, it)
		pq.up(len(pq.items) - 1)
	}
}

func (pq *PriorityQueue) Pop() (Item, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	n := len(pq.items)
	if n == 0 {
		return Item{}, false
	}
	res := pq.items[0]
	pq.items[0] = pq.items[n-1]
	pq.items = pq.items[:n-1]
	if len(pq.items) > 0 {
		pq.down(0)
	}
	return res, true
}

func (pq *PriorityQueue) Len() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.items)
}

func (pq *PriorityQueue) up(i int) {
	for i > 0 {
		p := (i - 1) / 2
		if pq.items[i].Priority >= pq.items[p].Priority {
			break
		}
		pq.items[i], pq.items[p] = pq.items[p], pq.items[i]
		i = p
	}
}

func (pq *PriorityQueue) down(i int) {
	n := len(pq.items)
	for {
		left := 2*i + 1
		if left >= n {
			break
		}
		smallest := left
		right := left + 1
		if right < n && pq.items[right].Priority < pq.items[left].Priority {
			smallest = right
		}
		if pq.items[i].Priority <= pq.items[smallest].Priority {
			break
		}
		pq.items[i], pq.items[smallest] = pq.items[smallest], pq.items[i]
		i = smallest
	}
}
