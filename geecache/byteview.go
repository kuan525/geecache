package geecache

// 存储真实的缓存值
type ByteView struct {
	b []byte
}

// 实现Value的接口
func (v ByteView) Len() int {
	return len(v.b)
}

// 返回一个拷贝，防止被篡改
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// 改格式
func (v ByteView) String() string {
	return string(v.b)
}

// 复制拷贝
func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
