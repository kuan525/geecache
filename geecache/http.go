package geecache

import (
	"bytes"
	"fmt"
	"geecache/consistenthash"
	pb "geecache/geecachepb"
	"github.com/golang/protobuf/proto"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

// 两个参数，第一个是自己的机名/IP和端口，第二个是通讯地址前缀
type HTTPPool struct {
	self        string                 // 作为节点（服务端），的ip/端口
	basePath    string                 // 约定好的前缀
	mu          sync.Mutex             // 每个服务端一把锁
	peers       *consistenthash.Map    // 根据key查找节点
	httpGetters map[string]*httpGetter // 每一个远程节点对应一个httpGetter，和baseURL有关
}

// 节点信息，ip/端口
type httpGetter struct {
	baseURL string // ip/端口
}

// 新建一个核心数据结构
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// 日志处理
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// 处理方法
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 先判断前缀是否是约定的前缀，如果不是直接返回错误
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	// 记录当前访问
	p.Log("%s  %s", r.Method, r.URL.Path)

	// 将URL拆分开，从len(p.basePath)这个位置开始，后面两段，用/分开，前一个是groupname，后一个是key
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)

	// 当拆分开不是两段的话，说明有问题，此时报错即可，400客户端问题
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// 得到groupname
	groupName := parts[0]
	// 得到key
	key := parts[1]

	// 通过name得到对应的group
	group := GetGroup(groupName)
	// 如果为空说明有问题
	if group == nil {
		http.Error(w, "no such group:"+groupName, http.StatusNotFound)
		return
	}

	// 在当前group中查找key是否存在，以及返回数据
	view, err := group.Get(key)
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	// 如果没有的话，说明服务端出问题，概念上的500
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 设置返回格式
	w.Header().Set("Content-Type", "application/octet-stream")
	// 填入返回值，没错误处理
	w.Write(body)
}

// 节点获取访问的数据
//func (h *httpGetter) Get(group string, key string) ([]byte, error) {
//	// 将url拼起来
//	u := fmt.Sprintf("%v%v/%v", h.baseURL, url.QueryEscape(group), url.QueryEscape(key))
//
//	// 作为客户端向其他节点（服务端）发送请求，得到回应
//	res, err := http.Get(u)
//	if err != nil {
//		return nil, err
//	}
//	// 延时关闭，这里没有错误处理
//	defer res.Body.Close()
//
//	if res.StatusCode != http.StatusOK {
//		return nil, fmt.Errorf("server returned: %v", res.Status)
//	}
//
//	// 将收到的消息弄到缓冲区里面
//	bytes := bytes.NewBuffer([]byte{})
//	_, err = io.Copy(bytes, res.Body)
//	if err != nil {
//		return nil, fmt.Errorf("reading response body: %v", err)
//	}
//
//	// 缓冲区转成bytes
//	return bytes.Bytes(), nil
//}

func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	u := fmt.Sprintf("%v%v/%v",
		h.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)

	res, err := http.Get(u)
	// 没有错误处理
	defer res.Body.Close()

	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	// 将收到的消息弄到缓冲区里面
	bytess := bytes.NewBuffer([]byte{})
	if _, err = io.Copy(bytess, res.Body); err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}

	// 解码一下，搞到out里面去
	if err = proto.Unmarshal(bytess.Bytes(), out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}

var _ PeerGetter = (*httpGetter)(nil)

// 实例化了一致性哈希，添加传入节点
func (p *HTTPPool) Set(peers ...string) {
	// 首先解锁，防止并发问题
	p.mu.Lock()
	defer p.mu.Unlock()

	// 利用一致性哈希的初始化方式
	p.peers = consistenthash.New(defaultReplicas, nil)
	// 加入当前传入的节点
	p.peers.Add(peers...)
	// 为每一个节点创建一个HTTP客户端的httpGetter
	p.httpGetters = make(map[string]*httpGetter, len(peers))

	// 添加HTTP客户端的方法httpGetter
	for _, peer := range peers {
		// 因为每一个节点的URL不一样，所以httpGetter中需要保存一下，存放的是对应节点的URL
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// 包装了一致性哈希算法的Get方法，根据具体的key，创建HTTP客户端从远程节点获取缓存值的能力
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 获取远程节点对应的httpGetter，不为空，也不为本地
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("pick peer %s", peer)
		// 将httpGetter返回回去
		return p.httpGetters[peer], true
	}
	// 返回空
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)
