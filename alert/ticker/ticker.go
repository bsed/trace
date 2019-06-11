package ticker

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/imdevlab/g"
)

var logger *zap.Logger

// Tickers 定时器集合
type Tickers struct {
	sync.Mutex
	id         int64              // 自增任务编号
	hash       *g.Hash            // 一致性hash工具
	tickers    map[string]*Ticker // 定时器集合
	tickerTime int                // 定时时间
}

// NewTickers ...
func NewTickers(tickerNum int, tickerTime int, l *zap.Logger) *Tickers {
	tickers := &Tickers{
		hash:       g.NewHash(),
		tickers:    make(map[string]*Ticker),
		tickerTime: tickerTime,
	}
	logger = l
	rand.Seed(time.Now().UnixNano())

	for index := 0; index < tickerNum; index++ {
		tickers.hash.Add(fmt.Sprintf("%d", index))
		ticker := newTicker(tickerTime)
		// 延迟启动时间
		deferTime := rand.Intn(8)
		logger.Info("deferTime", zap.Int("deferTime", deferTime))
		// 启动定时器
		go ticker.start(deferTime)

		tickers.tickers[fmt.Sprintf("%d", index)] = ticker
	}

	return tickers
}

// NewID 申请ID
func (t *Tickers) NewID() int64 {
	t.Lock()
	// id自增
	id := t.id
	t.id++
	t.Unlock()
	return id
}

// AddTask 添加任务
func (t *Tickers) AddTask(id int64, channel chan bool) {
	// id 通过hash计算出来key
	key, err := t.hash.Get(fmt.Sprintf("%d", id))
	if err != nil {
		logger.Warn("hash get", zap.String("error", err.Error()))
		return
	}
	// 加入任务
	t.tickers[key].addTask(id, channel)
}

// RemoveTask 添加任务
func (t *Tickers) RemoveTask(id int64) {
	// id 通过hash计算出来key
	key, err := t.hash.Get(fmt.Sprintf("%d", id))
	if err != nil {
		logger.Warn("hash get", zap.String("error", err.Error()))
		return
	}
	// 加入任务
	t.tickers[key].removeTask(id)
}

// Ticker 定时器
type Ticker struct {
	sync.RWMutex
	tasks      map[int64]*Task // 任务
	tickerTime int             // 定时时间
}

func newTicker(tickerTime int) *Ticker {
	return &Ticker{
		tasks:      make(map[int64]*Task),
		tickerTime: tickerTime,
	}
}

func (t *Ticker) start(deferTime int) {
	// 延迟启动
	time.Sleep(time.Second * time.Duration(deferTime))
	ticker := time.NewTicker(time.Duration(t.tickerTime) * time.Second)
	for {
		select {
		case <-ticker.C:
			t.RLock()
			for _, task := range t.tasks {
				task.channel <- true
			}
			t.RUnlock()
			break
		}
	}
}

func (t *Ticker) addTask(id int64, channel chan bool) {
	t.Lock()
	t.tasks[id] = newTask(id, channel)
	t.Unlock()
}

func (t *Ticker) removeTask(id int64) {
	t.Lock()
	if _, ok := t.tasks[id]; ok {
		delete(t.tasks, id)
	}
	t.Unlock()
}

// Task 任务
type Task struct {
	id        int64     // 编号
	channel   chan bool // 通知管道
	inputTime time.Time // 插入时间
}

// newTask 新任务
func newTask(id int64, channel chan bool) *Task {
	return &Task{
		id:        id,
		channel:   channel,
		inputTime: time.Now(),
	}
}
