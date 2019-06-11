package stats

// SQLS 接口计算统计
type SQLS struct {
	SQLS map[int32]*SQL
}

// NewSQLS ...
func NewSQLS() *SQLS {
	return &SQLS{
		SQLS: make(map[int32]*SQL),
	}
}

// Get 获取sql信息
func (s *SQLS) Get(sqlID int32) (*SQL, bool) {
	info, ok := s.SQLS[sqlID]
	return info, ok
}

// Store 存储sql信息
func (s *SQLS) Store(sqlID int32, info *SQL) {
	s.SQLS[sqlID] = info
}

// SQL 统计信息
type SQL struct {
	Duration    int32 // 总耗时
	MinDuration int32 // 最小耗时
	MaxDuration int32 // 最大耗时
	Count       int   // 发生次数
	ErrCount    int   // 错误次数
}

// NewSQL ...
func NewSQL() *SQL {
	return &SQL{}
}
