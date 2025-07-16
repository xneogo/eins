/*
 *  ┏┓      ┏┓
 *┏━┛┻━━━━━━┛┻┓
 *┃　　　━　　  ┃
 *┃   ┳┛ ┗┳   ┃
 *┃           ┃
 *┃     ┻     ┃
 *┗━━━┓     ┏━┛
 *　　 ┃　　　┃神兽保佑
 *　　 ┃　　　┃代码无BUG！
 *　　 ┃　　　┗━━━┓
 *　　 ┃         ┣┓
 *　　 ┃         ┏┛
 *　　 ┗━┓┓┏━━┳┓┏┛
 *　　   ┃┫┫  ┃┫┫
 *      ┗┻┛　 ┗┻┛
 @Time    : 2024/11/8 -- 18:45
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: breaker.go
*/

package utils

import (
	"context"
	"errors"
	"github.com/qiguanzhu/infra/lcl/governor/gentity"
	"github.com/qiguanzhu/infra/lcl/governor/gregistry/retcd"
	"github.com/qiguanzhu/infra/nerv/xlog"
	"github.com/qiguanzhu/infra/seele/zserv"
	"sync"
	"time"
)

type ItemConf struct {
	Name   string `json:"name"`
	Enable bool   `json:"enable"`
	Source int32  `json:"source"`
}

type BreakerStat struct {
	key   string
	fail  int64
	total int64
}

type Breaker struct {
	servName     string // 形如 {servGroup}/{servName}
	clientLookup zserv.ClientLookup[gentity.ServInfo]
	rwMutex      sync.RWMutex

	statCounter map[string]*BreakerStat
	statChan    chan *BreakerStat
}

func NewBreaker(clientLookup zserv.ClientLookup[gentity.ServInfo]) *Breaker {
	m := &Breaker{
		clientLookup: clientLookup,
		servName:     clientLookup.ServKey(),
		statCounter:  make(map[string]*BreakerStat),
		statChan:     make(chan *BreakerStat, 1024*10),
	}

	go m.monitor()
	return m
}

func (m *Breaker) monitor() {
	fun := "Breaker.monitor -->"
	ctx := context.Background()

	ticker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-ticker.C:
			for _, stat := range m.statCounter {
				if stat.fail > 5 && stat.total > 5 &&
					(float64(stat.fail)/float64(stat.total)) > 0.02 {
					xlog.Errorf(ctx, "%s breaker stat, key:%s, total:%d, fail:%d", fun, stat.key, stat.total, stat.fail)
				}
			}
			m.statCounter = make(map[string]*BreakerStat)

		case stat := <-m.statChan:
			tmp := &BreakerStat{
				key: stat.key,
			}
			if item, ok := m.statCounter[stat.key]; ok {
				tmp = item
			}

			tmp.fail += stat.fail
			tmp.total += stat.total
			m.statCounter[tmp.key] = tmp
		}
	}
}

func (m *Breaker) doStat(key string, total, fail int64) {
	fun := "Breaker.doStat -->"

	stat := &BreakerStat{
		key:   key,
		total: total,
		fail:  fail,
	}

	select {
	case m.statChan <- stat:
	default:
		xlog.Errorf(context.Background(), "%s drop, key:%s, total:%d, fail:%d", fun, stat.key, stat.total, stat.fail)
	}
}

func (m *Breaker) Do(ctx context.Context, funcName string, run func(ctx context.Context) error, fallback func(context.Context, error) error) error {
	fun := "Breaker.Do -->"
	fail := int64(0)
	// 稳定性平台，接口熔断配置的 key 形如 {servGroup}/{servName}/{funcName}。这里 m.servName 已为 {servGroup}/{servName} 形式了。
	key := m.servName + "/" + funcName
	err := retcd.Do(ctx, key, run, fallback)
	if errors.Is(err, retcd.ErrCircuitBreakerRegistryNotInited) {
		// circuit_breaker 未初始化，视同无熔断。
		xlog.Warnf(ctx, " circuit breaker registry not inited! Call `circuit_breaker.Init()` in your project's `logic.Init()` first!")
		return run(ctx)
	}

	if err != nil {
		xlog.Warnf(ctx, "%s key:%s err: %s", fun, key, err)
		fail = 1
	}

	xlog.Debugf(ctx, "Breaker key:%s fail:%d", key, fail)
	m.doStat(key, 1, fail)
	return err
}
