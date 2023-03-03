package geecache

import (
	"fmt"
	pb "geecache/geecachepb"
	"geecache/singleflight"
	"log"
	"sync"
)

// 回调函数接口
type Getter interface {
	Get(key string) ([]byte, error)
}

// 定义函数类型
type Getterfunc func(key string) ([]byte, error)

// 实现接口型函数，方便使用者在调用时既能够传入函数作为参数，也能够传入实现了该结构的结构体作为参数
func (f Getterfunc) Get(key string) ([]byte, error) {
	return f(key)
}

// 一个Group是一个缓存（name）的命名空间
type Group struct {
	name      string
	getter    Getter              // 回调函数
	mainCache cache               // 缓存大小
	peers     PeerPicker          // 一个接口，HTTPPool实现了他，用于将Group和HTTPPool绑定
	loader    *singleflight.Group // 防止缓存雪崩
}

var (
	mu sync.Mutex
	// 命名空间组
	groups = make(map[string]*Group)
)

// 初始化一个Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	// 当回调函数不存在
	if getter == nil {
		panic("nil Getter")
	}

	mu.Lock()
	defer mu.Unlock()

	// 初始化一个空间
	g := &Group{
		name:   name,
		getter: getter,
		// mainCache是cache，其中lru和mu并没有初始化，延时初始化
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}

	groups[name] = g
	return g
}

// 得到当前name的Group
func GetGroup(name string) *Group {
	mu.Lock()
	defer mu.Unlock()

	g := groups[name]
	return g
}

// 获取数据，一层一层的
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	// 通过Group的cache的get函数来访问
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[geecache] hit")
		return v, nil
	}

	// 当缓存中没有的时候，就需要调用load方法，进而调用getLocally(分布式场景下使用getFromPeer从其他节点获取)
	return g.load(key)
}

// 非本机节点，调用getFromPeer从远程获取，若是本机节点或失败，则回退到getLocally
// 使用g.loader.Do将其包裹起来，这样可以确保并发场景下针对相同的key，load过程只会调用一次
func (g *Group) load(key string) (value ByteView, err error) {
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			// 找到对应到节点
			if peer, ok := g.peers.PickPeer(key); ok {
				// 通过该节点得到信息
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}

		// 没有节点或者找到失败，则返回本地资源
		return g.getLocally(key)
	})

	// 当没有错误的时候，返回结果即可
	if err == nil {
		return viewi.(ByteView), nil
	}
	// 当有错误，就是空ByteView{}和err
	return
}

// 通过回调函数获取
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)

	if err != nil {
		return ByteView{}, err
	}

	// 复制一份，防止value被修改导致原数据错误
	value := ByteView{b: cloneBytes(bytes)}

	// 将当前访问的热点数据加入到缓存当中，使用lru策略管理
	g.populateCache(key, value)
	return value, nil
}

// 加入缓存，就是调用cache的add函数
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// 实现了PeerPicker接口的HTTPPool注入到Group
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 使用实现了PeerGetter接口到httpGetter从远程节点获取缓存值
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	//bytes, err := peer.Get(g.name, key)
	//if err != nil {
	//	return ByteView{}, err
	//}
	//return ByteView{b: bytes}, nil

	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}

	res := &pb.Response{}
	err := peer.Get(req, res)

	if err != nil {
		return ByteView{}, err
	}

	return ByteView{b: res.Value}, nil
}
