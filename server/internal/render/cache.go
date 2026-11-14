package render

import "sync"

// ByteCache is a small FIFO cache for rendered meme images. It keeps generated
// images in memory only, so durable storage stays limited to meme specs.
type ByteCache struct {
	mu       sync.Mutex
	maxBytes int
	size     int
	order    []string
	items    map[string][]byte
}

// NewByteCache creates a cache capped at maxBytes.
func NewByteCache(maxBytes int) *ByteCache {
	return &ByteCache{maxBytes: maxBytes, items: make(map[string][]byte)}
}

// Get returns a cached image.
func (c *ByteCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, ok := c.items[key]
	return data, ok
}

// Put stores an image, evicting the oldest entries until it fits.
func (c *ByteCache) Put(key string, data []byte) {
	if len(data) == 0 || len(data) > c.maxBytes {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.items[key]; exists {
		return
	}
	for c.size+len(data) > c.maxBytes && len(c.order) > 0 {
		oldest := c.order[0]
		c.order = c.order[1:]
		c.size -= len(c.items[oldest])
		delete(c.items, oldest)
	}
	c.items[key] = data
	c.order = append(c.order, key)
	c.size += len(data)
}
