package alert

// SQLs sqls
type SQLs struct {
	SQLs map[int32]*SQL `msg:"sqls"`
}

// NewSQLs ...
func NewSQLs() *SQLs {
	return &SQLs{
		SQLs: make(map[int32]*SQL),
	}
}

// SQL sql计算
type SQL struct {
	ID       int32 `msg:"desc"`
	Count    int   `msg:"count"`
	Errcount int   `msg:"errcount"`
	Duration int32 `msg:"duration"`
}

// NewSQL  ...
func NewSQL() *SQL {
	return &SQL{}
}
