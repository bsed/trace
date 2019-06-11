package service

// Orderly 排序工具
type Orderly []int64

// Len OrderlyKey 长度
func (o Orderly) Len() int {
	return len(o)
}

// Swap 交换
func (o Orderly) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

// Less 对比
func (o Orderly) Less(i, j int) bool {
	return o[i] < o[j]
}
