package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 哈希函数提前声明
type Hash func(data []byte) uint32

type Map struct {
	hash     Hash           // 哈希函数
	replicas int            // 虚节点的倍数
	keys     []int          // 哈希环
	hashMap  map[int]string // 虚拟节点的hash指向真实节点的原本
}

// 初始化一个一致性哈希结构体
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}

	// 当没有吃书画哈希函数的时候，使用crc32.ChecksumIEEE函数
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}

	return m
}

// 添加真实节点，可以多个
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		// 遍历到当前节点，需要一次创建每一个虚节点
		for i := 0; i < m.replicas; i++ {
			// 得到虚节点的哈希值，等下要放到哈希环中
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			// 加入到哈希环，注意这里放在哪里无所谓，后面会排序
			m.keys = append(m.keys, hash)
			// 虚节点的哈希值映射到真实节点到原本
			m.hashMap[hash] = key
		}
	}
	// 将哈希环排序，方便后续在查找到时候找到后面一个最近的
	sort.Ints(m.keys)
}

// 获得当前key应该查询分布式系统中的哪一个节点
func (m *Map) Get(key string) string {
	// 当哈希环中没有节点的时候，表示当前分布式系统中没有节点可以使用
	if len(m.keys) == 0 {
		return ""
	}

	// 得到当前查询key的哈希值，方便找哈希环后面的一个位置
	hash := int(m.hash([]byte(key)))

	// 找到哈希环后面的一个位置
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	// 找到下表后面的一个位置，如果是越界的话，表示需要使用第一个，所以直接取余数即可
	return m.hashMap[m.keys[idx%len(m.keys)]]
}
