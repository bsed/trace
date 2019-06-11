package service

// OrderlyKeys 排序工具
type OrderlyKeys []int64

// Len OrderlyKey 长度
func (o OrderlyKeys) Len() int {
	return len(o)
}

// Swap 交换
func (o OrderlyKeys) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

// Less 对比
func (o OrderlyKeys) Less(i, j int) bool {
	return o[i] < o[j]
}
