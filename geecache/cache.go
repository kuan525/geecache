package geecache

import (
	"geecache/lru"
	"sync"
)

// 封装一个加锁的结构体
type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64
}

// 添加，如果lru没有初始化，则新建一个。属于延迟更新
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

// 获取，c.lru.Get(key)返回的时value，这里将其转化为ByteView，实现了Len接口的
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}

	return
}
