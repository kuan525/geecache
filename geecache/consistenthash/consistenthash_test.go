package consistenthash

import (
	"strconv"
	"testing"
)

func TestHashing(t *testing.T) {
	// 自己声明的一个哈希函数
	hash := New(3, func(key []byte) uint32 {
		i, _ := strconv.Atoi(string(key))
		return uint32(i)
	})

	// 加入三个真实节点
	hash.Add("6", "4", "2")

	// 测试那个对应哪一个
	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	// 当前的k-v，是否是对应的
	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}

	// 插入真实节点8
	hash.Add("8")

	// 修改标答
	testCases["27"] = "8"

	// 再测试一边，因为刚才插入新节点，样例也修改
	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s, should have yielded %s", k, v)
		}
	}
}
