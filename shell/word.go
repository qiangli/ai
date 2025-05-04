package shell

import (
	"container/heap"
	"hash/fnv"
	"math"
	"strings"
	"sync"
	"unicode"
)

type WordFreq struct {
	Word   string
	Count  int
	Recent int
	Score  float64
}

type WordStat struct {
	Count       int         // total occurrences
	Recent      int         // last-seen “tick”
	ClassCounts map[int]int // histogram by class
}

// bump total & per-class counters
func (ws *WordStat) touch(class int) {
	ws.Count++
	ws.ClassCounts[class]++
}

// return the class with the highest frequency
func (ws *WordStat) dominantClass() int {
	best, bestCnt := 0, 0
	for cls, cnt := range ws.ClassCounts {
		if cnt > bestCnt {
			best, bestCnt = cls, cnt
		}
	}
	return best
}

// LRU heap (oldest at top)
type lruItem struct {
	word   string
	recent int
	index  int // heap index
}

type lruHeap []*lruItem

func (h lruHeap) Len() int           { return len(h) }
func (h lruHeap) Less(i, j int) bool { return h[i].recent < h[j].recent } // oldest first
func (h lruHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i]; h[i].index, h[j].index = i, j }
func (h *lruHeap) Push(x any)        { *h = append(*h, x.(*lruItem)) }
func (h *lruHeap) Pop() any          { old := *h; n := len(old); it := old[n-1]; *h = old[:n-1]; return it }

// Min-heap on score (lowest at top) for top-N
type minHeap []WordFreq

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].Score < h[j].Score }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any)        { *h = append(*h, x.(WordFreq)) }
func (h *minHeap) Pop() any          { old := *h; n := len(old); x := old[n-1]; *h = old[:n-1]; return x }

type WordCounter struct {
	sync.RWMutex

	words      map[string]*WordStat
	recencySeq int

	/* inverted indices for quick pre-filtering */
	charIdx    map[byte]map[string]struct{}
	bigramIdx  map[string]map[string]struct{}
	trigramIdx map[string]map[string]struct{}

	/* LRU eviction */
	maxSize  int
	lruH     lruHeap
	lruItems map[string]*lruItem

	/* per-class weights */
	classW map[int]float64
}

func NewWordCounter(maxSize int, classWeights map[int]float64) *WordCounter {
	cw := make(map[int]float64, len(classWeights))
	for k, v := range classWeights {
		cw[k] = v
	}
	return &WordCounter{
		words:      make(map[string]*WordStat),
		charIdx:    make(map[byte]map[string]struct{}),
		bigramIdx:  make(map[string]map[string]struct{}),
		trigramIdx: make(map[string]map[string]struct{}),
		maxSize:    maxSize,
		lruItems:   make(map[string]*lruItem),
		classW:     cw,
	}
}

// DefaultWordCounter returns a WordCounter with reasonable defaults so callers
// don’t have to worry about the parameters in the common case.
//
// Defaults:
//   - maxSize      – 100 000 distinct words
//   - classWeights – nil  (all classes weighted equally)
func DefaultWordCounter() *WordCounter {
	const defaultMaxSize = 100_000
	return NewWordCounter(defaultMaxSize, nil)
}

func addToIdx[K comparable](idx map[K]map[string]struct{}, key K, word string) {
	m, ok := idx[key]
	if !ok {
		m = make(map[string]struct{}, 4)
		idx[key] = m
	}
	m[word] = struct{}{}
}

func indexWord(c *WordCounter, w string) {
	bytes := []byte(strings.ToLower(w))

	// single characters
	seen := make(map[byte]struct{}, 8)
	for _, b := range bytes {
		if _, ok := seen[b]; !ok {
			addToIdx(c.charIdx, b, w)
			seen[b] = struct{}{}
		}
	}

	// bigrams
	for i := 0; i+1 < len(bytes); i++ {
		bg := string(bytes[i : i+2])
		addToIdx(c.bigramIdx, bg, w)
	}

	// trigrams
	for i := 0; i+2 < len(bytes); i++ {
		tg := string(bytes[i : i+3])
		addToIdx(c.trigramIdx, tg, w)
	}
}

func (c *WordCounter) lookupCandidates(q string) []string {
	q = strings.ToLower(q)
	switch len(q) {
	case 0:
		return nil
	case 1:
		if set, ok := c.charIdx[q[0]]; ok {
			return setToSlice(set)
		}
	case 2:
		if set, ok := c.bigramIdx[q]; ok {
			return setToSlice(set)
		}
	default:
		if set, ok := c.trigramIdx[q[:3]]; ok { // use first trigram as anchor
			return setToSlice(set)
		}
	}
	return nil
}

func setToSlice(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for w := range m {
		out = append(out, w)
	}
	return out
}

func removeFromIdx[K comparable](idx map[K]map[string]struct{}, word string) {
	for _, m := range idx {
		delete(m, word)
	}
}

func (c *WordCounter) AddWords(words []string) {
	c.AddWordsWithClass(0, words)
}

func (c *WordCounter) AddWordsWithClass(class int, words []string) {
	c.Lock()
	defer c.Unlock()

	for _, w := range words {
		c.recencySeq++

		stat, exists := c.words[w]
		if !exists {
			stat = &WordStat{ClassCounts: make(map[int]int)}
			c.words[w] = stat
			indexWord(c, w)

			item := &lruItem{word: w, recent: c.recencySeq}
			heap.Push(&c.lruH, item)
			item.index = c.lruH.Len() - 1
			c.lruItems[w] = item
		} else {
			// Update LRU position
			it := c.lruItems[w]
			it.recent = c.recencySeq
			heap.Fix(&c.lruH, it.index)
		}

		stat.touch(class)
		stat.Recent = c.recencySeq

		// Optional eviction
		if c.maxSize > 0 {
			for len(c.words) > c.maxSize {
				c.evictOldest()
			}
		}
	}
}

// Scoring knobs
var (
	WFreq    = 0.7 // frequency weight
	WRecency = 0.4 // recency weight
	WLen     = 0.3 // length bonus
	WSepPen  = 0.5 // separator-ratio penalty
	WClass   = 0.2 // class weight

	WComplex = 1.4 // influence of low-complexity detector

	WInfo = 0.8 // information content (entropy + RLE)
)

// Shannon entropy, normalised to 0‥1
func shannonEntropy(b *[256]int, n int) float64 {
	if n == 0 {
		return 0
	}
	uniq, H := 0, 0.0
	for _, c := range b {
		if c == 0 {
			continue
		}
		uniq++
		p := float64(c) / float64(n)
		H -= p * math.Log2(p)
	}
	if uniq <= 1 {
		return 0
	}
	return H / math.Log2(float64(uniq))
}

// Run-length ratio  (1.0 ⇒ no compression, 0.0 ⇒ one giant run)
func rleRatio(s string) float64 {
	if len(s) < 2 {
		return 0
	}
	runes, runs := []rune(s), 1
	for i := 1; i < len(runes); i++ {
		if runes[i] != runes[i-1] {
			runs++
		}
	}
	return float64(runs) / float64(len(runes))
}

// Blend entropy + RLE → information score   (0‥1)
func informationScore(s string, b *[256]int) float64 {
	n := len(s)
	if n == 0 {
		return 0
	}
	return 0.6*shannonEntropy(b, n) + 0.4*rleRatio(s)
}

// Logistic-ish length bonus (words of 6–15 chars get ≈1, 1-2 chars get 0)
func lengthBonus(n int) float64 {
	if n <= 1 {
		return 0
	}
	return 1 / (1 + math.Exp(-(float64(n)-6)/2))
}

const (
	lengthBias          = 8.0  // how fast the length factor saturates
	ngramK              = 5    // use 5-grams; good compromise for words / hex
	wEntropyLen float64 = 0.30 // weight of entropy·length component
	wNGram      float64 = 0.70 // weight of distinct-n-gram component

	kGram         = 5    // keep 5-grams
	wDup  float64 = 0.20 // ← new: stutter penalty weight
)

/* k-gram duplicate ratio 0‥1 ──────────────────────────────────────────*/
func dupRatio(s string, k int) float64 {
	n := len(s)
	if n < k+1 {
		return 0
	}
	windows := n - k + 1
	// Cheap rolling hash: FNV-1a, 64 bit.
	h := fnv.New64a()

	seen := make(map[uint64]struct{}, windows)
	dup := 0

	for i := 0; i < windows; i++ {
		frag := s[i : i+k]
		h.Reset()
		_, _ = h.Write([]byte(frag))
		key := h.Sum64()

		if _, ok := seen[key]; ok {
			dup++
		} else {
			seen[key] = struct{}{}
		}
	}
	return float64(dup) / float64(windows)
}

const (
	// kGram                = 5
	wLen     float64 = 0.40 // ← NEW: length weight
	wEntropy float64 = 0.30
	pivotLen         = 24 // where the sigmoid is ~0.5
)

func lenScore(s string) float64 {
	n := len(s)
	if n == 0 {
		return 0
	}
	// logistic: 1 / (1 + e^(–k·(n – pivot)))
	const k = 0.25
	exp := math.Exp(-k * (float64(n) - float64(pivotLen)))
	return 1.0 / (1.0 + exp)
}

func complexityScore(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	ls := lenScore(s)                  // 0‥1  ← new
	es := entropyLengthComponent(s)    // existing 0‥1
	ug := distinctNGramRatio(s, kGram) // existing 0‥1
	dr := dupRatio(s, kGram)           // existing 0‥1

	score := wLen*ls + wEntropy*es + wNGram*ug - wDup*dr
	if score < 0 {
		return 0
	}
	return score
}

// entropyLengthComponent = (Shannon-entropy / maxEntropy) * lengthFactor.
func entropyLengthComponent(s string) float64 {
	n := len(s)

	// 256 ASCII + 1 overflow bucket for non-ASCII
	var freq [257]int
	uniq := 0
	for _, r := range s {
		b := 256
		if r < 256 {
			b = int(r)
		}
		if freq[b] == 0 {
			uniq++
		}
		freq[b]++
	}

	// Shannon entropy H = −Σ p log₂ p
	N := float64(n)
	h := 0.0
	for _, c := range freq {
		if c == 0 {
			continue
		}
		p := float64(c) / N
		h -= p * math.Log2(p)
	}
	maxH := 0.0
	if uniq > 1 {
		maxH = math.Log2(float64(uniq))
	}
	if maxH == 0 { // all chars identical
		return 0
	}
	entropyNorm := h / maxH         // 0‥1
	lengthF := N / (N + lengthBias) // 0‥1
	return entropyNorm * lengthF    // 0‥1
}

// distinctNGramRatio returns (#unique k-grams) / (#total k-grams).
func distinctNGramRatio(s string, k int) float64 {
	n := len(s)
	if n < k {
		return 0
	}
	total := n - k + 1
	set := make(map[uint32]struct{}, total)

	// simple rolling FNV-1a hash over bytes; fast & allocation-free
	var h uint32
	const prime32 = 16777619
	window := []byte(s[:k])
	for _, b := range window {
		h ^= uint32(b)
		h *= prime32
	}
	set[h] = struct{}{}

	pow := uint32(1)
	for i := 0; i < k-1; i++ {
		pow *= prime32
	}

	for i := k; i < n; i++ {
		// slide window: remove s[i-k], add s[i]
		out, in := s[i-k], s[i]
		h ^= uint32(out)
		h *= pow // undo one multiply
		h ^= uint32(in)
		h *= prime32
		set[h] = struct{}{}
	}

	return float64(len(set)) / float64(total) // 0‥1
}

func isAlphaNumeric(r rune) bool {
	return unicode.IsDigit(r) || unicode.IsLetter(r)
}

// ScoreWord returns a positive value (≈0‥4). −1 means “discard”.
func (c *WordCounter) ScoreWord(w string, st *WordStat) float64 {
	n := len(w)
	if n == 0 {
		return -1
	}

	/* 1. Pass over runes: separator count + histogram for entropy */
	sep := 0
	var bucket [256]int
	for _, r := range w {
		if !isAlphaNumeric(r) {
			sep++
		}
		if r < 256 {
			bucket[r]++
		}
	}
	sepRatio := float64(sep) / float64(n)

	/* Optional hard filters */
	// too many separators → noise
	if sepRatio > 0.90 {
		return -1
	}

	info := informationScore(w, &bucket)
	// extremely repetitive (rwrwrw, 11111…)
	if info < 0.05 {
		return -1
	}

	complexity := complexityScore(w)
	if complexity < 0.90 {
		return -1
	}

	/* 2. Individual weighted terms */
	freq := math.Log(float64(st.Count) + 1) // ≥0
	age := float64(c.recencySeq-st.Recent) + 1
	recency := 1 / age       // (0,1]
	length := lengthBonus(n) // 0‥1

	class := 0.0
	if idx := st.dominantClass(); idx < len(c.classW) {
		class = c.classW[idx]
	}

	/* 3. Combine */
	score := WFreq*freq +
		WRecency*recency +
		WLen*length +
		WClass*class +
		WInfo*info +
		WComplex*complexity - // reward diversity
		WSepPen*sepRatio // penalty subtracts points

	return score
}

// Suggest / Top-N
func (c *WordCounter) Suggest(query string, n int) []WordFreq {
	if n <= 0 {
		return nil
	}

	c.RLock()

	var candidates []string
	if query == "" {
		// top-N mode: consider every word
		candidates = make([]string, 0, len(c.words))
		for w := range c.words {
			candidates = append(candidates, w)
		}
	} else {
		candidates = c.lookupCandidates(query)
		if len(candidates) == 0 { // fallback: all words
			candidates = make([]string, 0, len(c.words))
			for w := range c.words {
				candidates = append(candidates, w)
			}
		}
	}

	h := &minHeap{}
	heap.Init(h)

	for _, w := range candidates {
		if query != "" && !strings.Contains(strings.ToLower(w), strings.ToLower(query)) {
			continue
		}

		stat := c.words[w]
		score := c.ScoreWord(w, stat)
		wf := WordFreq{Word: w, Count: stat.Count, Score: score, Recent: stat.Recent}

		switch {
		case h.Len() < n:
			heap.Push(h, wf)
		case wf.Score > (*h)[0].Score ||
			(wf.Score == (*h)[0].Score && wf.Recent > (*h)[0].Recent):
			heap.Pop(h)
			heap.Push(h, wf)
		}
	}

	c.RUnlock()

	// turn min-heap into sorted slice (highest score first)
	res := make([]WordFreq, h.Len())
	for i := len(res) - 1; i >= 0; i-- {
		res[i] = heap.Pop(h).(WordFreq)
	}
	return res
}

// LRU eviction
func (c *WordCounter) evictOldest() {
	if len(c.lruH) == 0 {
		return
	}
	it := heap.Pop(&c.lruH).(*lruItem)
	delete(c.lruItems, it.word)

	// remove from indices (char, bi, tri)
	removeFromIdx(c.charIdx, it.word)
	removeFromIdx(c.bigramIdx, it.word)
	removeFromIdx(c.trigramIdx, it.word)

	delete(c.words, it.word)
}
