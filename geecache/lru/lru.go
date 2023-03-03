package lru

import "container/list"

// 允许值是实现了Value借口的任意类型，只包含Len，用于返回值所占的内存大小
type Value interface {
	Len() int
}

// 允许最大内存，当前内存， 双向链表，哈希表，记录被移除时的回调函数
type Cache struct {
	maxBytes  int64
	nbyte     int64
	ll        *list.List
	cache     map[string]*list.Element
	OnEvicted func(key string, value Value)
}

// 双向链表的结构，保存key是因为方便从map中删除k-v
type entry struct {
	key   string
	value Value
}

// 实例化Cache，返回的是一个指针
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// 查找
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		// 如果存在，移到队尾 front作为队尾
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// 删除
func (c *Cache) RemoveOldest() {
	// 取到队首节点，从链表中删除
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		// 从字典c.cache中删除节点的映射关系
		delete(c.cache, kv.key)
		// 更新当前的使用内存大小
		c.nbyte -= int64(len(kv.key)) + int64(kv.value.Len())
		// 如果有回调函数，则执行
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 新增/修改
func (c *Cache) Add(key string, value Value) {
	// 当前存在，则拿到对头去，同时修改数据和内存大小
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbyte += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		// 加到头部去
		ele := c.ll.PushFront(&entry{key, value})
		// 字典中添加记录
		c.cache[key] = ele
		c.nbyte += int64(len(key)) + int64(value.Len())
	}

	// 内存超了，淘汰一个 , 为0是测试情况
	for c.maxBytes != 0 && c.maxBytes < c.nbyte {
		c.RemoveOldest()
	}
}

// 统计添加了多少条数据
func (c *Cache) Len() int {
	return c.ll.Len()
}
