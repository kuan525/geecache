package singleflight

import "sync"

// 正在进行中，或以及结束的请求，使用sync.WaitGroup锁避免重入
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// 主要数据结构，管理不同的key的请求
type Group struct {
	mu sync.Mutex       // 锁
	m  map[string]*call // key对应的call
}

// 参数：key, 函数fn。针对相同的key，无论Do被调用多少次，函数fn都只会调用一次，等待fn调用结束了，返回返回值或错误
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock() // 保护m不被并发读写
	// 操作g.m之前，先检查是否初始化
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	// 当访问key的时候，如果以前访问过，则直接读出来，直接返回，这个相当于是吧近期（很短）访问的相同的key统一答复
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()       // 现在g.m不会被操作，可以解开
		c.wg.Wait()         // 如果请求正在进行中，则等待
		return c.val, c.err // 请求结束，返回结果
	}

	c := new(call) // 以前没有访问过，直接初始化一个call，对应当前的key
	c.wg.Add(1)    //发起请求之前加上锁
	g.m[key] = c   // 添加g.m，表面key以及有对应的请求在处理
	g.mu.Unlock()  // g.m后面不会操作，所以现在解开

	c.val, c.err = fn() // 调用fn，发起请求
	c.wg.Done()         // 请求结束上面的三行中的时候，在c.wg.Wait会阻塞住

	g.mu.Lock()      // 下面要错做g.m，需要锁住
	delete(g.m, key) //更新g.m
	g.mu.Unlock()    // 解锁

	return c.val, c.err // 将结果返回
}
