package main

import (
	"flag"
	"fmt"
	"geecache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

// 创建一个group，缓存命名空间
func createGroup() *geecache.Group {
	// 回调函数本地读取
	return geecache.NewGroup("scores", 2<<10, geecache.Getterfunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// 启动缓存服务器，创建HTTPPool，添加节点信息，注册到gee中，启动HTTP服务，（共三个端口，8001/2/3），用户不感知
func startCacheServer(addr string, addrs []string, gee *geecache.Group) {
	// 每个服务端都一个一个核心结构体HTTPPool
	peers := geecache.NewHTTPPool(addr)
	// 将当前有到节点放进去
	peers.Set(addrs...)
	// HTTPPool注册到group
	gee.RegisterPeers(peers)
	log.Println("geecache is running at", addr)
	// 当前端口监听
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// 启动一个API服务，与用户进行交互，用户感知
func startAPIServer(apiAddr string, gee *geecache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))

	log.Println("fontend server is running at", apiAddr)
	// 启动监听
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	var port int
	var api bool
	// 指向整形变量的指针，name是命令行参数的名称，value是默认值，usage是使用说明
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	// 同理上面， 这两句是从命令行获取元素的方式
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	// 标准映射组
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	// v地址存下来
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	// 创建一个group，就是一个name对应到命名空间
	gee := createGroup()
	if api {
		// 启动API服务器
		go startAPIServer(apiAddr, gee)
	}
	// 启动缓存服务器
	startCacheServer(addrMap[port], []string(addrs), gee)
}
