package geecache

import pb "geecache/geecachepb"

// 抽象一个接口，通过key获取对应的节点PeerGetter
type PeerPicker interface {
	PickPeer(key string) (pee PeerGetter, ok bool)
}

// 节点接口，从对应的group查找缓存值，节点之间访问，交给他
//type PeerGetter interface {
//	Get(group string, key string) ([]byte, error)
//}

// 参数转换为geecachepb.pb.go中的数据类型，还需要更改geecache.go和http.go中使用了PeerGetter接口的地方
type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
}
