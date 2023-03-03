package geecache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

// 查看回调函数是否正常
func TestGetter(t *testing.T) {
	var f Getter = Getterfunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))

	gee := NewGroup("scores", 2<<10, Getterfunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	for k, v := range db {
		if view, err := gee.Get(k); err != nil || view.String() != v {
			t.Fatalf("failed to get value to Tom")
		}
		if _, err := gee.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}

	// 检查一个不存在的东西，却有返回结构，是不正常了，下面将返回结果输出出来
	if view, err := gee.Get("unknow"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}
