package shell

import (
	"container/heap"
	"os"
	"strings"
	"sync"
)

type WordStat struct {
	Count  int
	Recent int64 // Recency counter (higher = more recent)
}

type WordFreq struct {
	Word   string
	Count  int
	Score  int64
	Recent int64 // For info
}

type WordCounter struct {
	sync.Mutex
	Words      map[string]*WordStat
	recencySeq int64 // Monotonically increasing per word seen
}

func NewWordCounter() *WordCounter {
	return &WordCounter{Words: make(map[string]*WordStat)}
}

func (c *WordCounter) AddWords(words []string) {
	c.Lock()
	defer c.Unlock()
	for _, w := range words {
		c.recencySeq++
		stat, ok := c.Words[w]
		if !ok {
			stat = &WordStat{}
			c.Words[w] = stat
		}
		stat.Count++
		stat.Recent = c.recencySeq // Update most recent seen
	}
}

func wordScore(word string, stat *WordStat) int64 {
	pathBonus := 0
	if strings.Contains(word, string(os.PathSeparator)) {
		pathBonus = 1000000
	}
	lengthBonus := len(word) * 1000
	frequencyBonus := stat.Count                  // Least significant
	recencyBonus := int(stat.Recent * 1000000000) // Most significant

	// ORDER: recency > path > length > frequency
	return int64(recencyBonus + pathBonus + lengthBonus + frequencyBonus)
}

type minHeap []WordFreq

func (h minHeap) Len() int { return len(h) }
func (h minHeap) Less(i, j int) bool {
	if h[i].Score == h[j].Score {
		// More recent is better if scores are equal
		return h[i].Recent < h[j].Recent
	}
	return h[i].Score < h[j].Score
}
func (h minHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x interface{}) { *h = append(*h, x.(WordFreq)) }
func (h *minHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

func (c *WordCounter) TopN(n int) []WordFreq {
	c.Lock()
	defer c.Unlock()
	if n <= 0 {
		return nil
	}
	h := &minHeap{}
	heap.Init(h)
	for w, stat := range c.Words {
		score := wordScore(w, stat)
		wf := WordFreq{w, stat.Count, score, stat.Recent}
		if h.Len() < n {
			heap.Push(h, wf)
		} else if score > (*h)[0].Score ||
			(score == (*h)[0].Score && stat.Recent > (*h)[0].Recent) {
			heap.Pop(h)
			heap.Push(h, wf)
		}
	}
	res := make([]WordFreq, h.Len())
	for i := len(res) - 1; i >= 0; i-- {
		res[i] = heap.Pop(h).(WordFreq)
	}
	return res
}
