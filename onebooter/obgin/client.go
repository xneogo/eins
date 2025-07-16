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
 @Time    : 2024/11/8 -- 17:29
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: client.go
*/

package obgin

import (
	"context"
	"fmt"
	"github.com/xneogo/eins/onebooter"
	"github.com/xneogo/infra/lcl/bootup/utils"
	"github.com/xneogo/infra/lcl/governor/fallback"
	"github.com/xneogo/infra/lcl/governor/gentity"
	"github.com/xneogo/infra/lcl/governor/gstat"
	"github.com/xneogo/infra/nerv/magi/xtime"
	"github.com/xneogo/infra/seele/zconfig"
	"github.com/xneogo/infra/seele/zserv"
	"time"
)

// ClientWrapper 目前网关通过common/go/pub在使用
type ClientWrapper struct {
	fallback.Fallbacks

	clientLookup zserv.ClientLookup[gentity.ServInfo]
	configCenter zconfig.ConfigCenter
	processor    string
	breaker      *utils.Breaker
	router       utils.Router
}

func NewClientWrapper(cb zserv.ClientLookup[gentity.ServInfo], processor string) *ClientWrapper {
	return NewClientWrapperWithRouterType(cb, processor, 0)
}

func NewClientWrapperByConcurrentRouter(cb zserv.ClientLookup[gentity.ServInfo], processor string) *ClientWrapper {
	return NewClientWrapperWithRouterType(cb, processor, 1)
}

func NewClientWrapperWithRouterType(cb zserv.ClientLookup[gentity.ServInfo], processor string, routerType int) *ClientWrapper {
	return &ClientWrapper{
		clientLookup: cb,
		processor:    processor,
		breaker:      utils.NewBreaker(cb),
		router:       utils.NewRouter(routerType, cb),
	}
}

func (m *ClientWrapper) Do(hashKey string, timeout time.Duration, run func(addr string, timeout time.Duration) error) error {
	var err error
	funcName := utils.GetFuncNameWithCtx(context.Background(), 3)
	retry := m.getFuncRetry(m.clientLookup.ServKey(), funcName)
	timeout = m.getFuncTimeout(m.clientLookup.ServKey(), funcName, timeout)
	for ; retry >= 0; retry-- {
		err = m.do(hashKey, funcName, timeout, run)
		if err == nil {
			return nil
		}
	}
	return err
}

func (m *ClientWrapper) do(hashKey, funcName string, timeout time.Duration, run func(addr string, timeout time.Duration) error) error {
	fun := "ClientWrapper.Do -->"
	si := m.router.Route(context.TODO(), m.processor, hashKey)
	if si == nil {
		return fmt.Errorf("%s not find service:%s processor:%s", fun, m.clientLookup.ServPath(), m.processor)
	}
	m.router.Pre(si)
	defer m.router.Post(si)

	call := func(_ctx context.Context) error {
		return run(si.Addr, timeout)
	}

	var err error
	st := xtime.NewTimeStat()
	defer func() {
		gstat.Collector(bootup.GetServBase(), m.clientLookup.ServKey(), m.processor, st.Duration(), 0, si.ServId, funcName, err)
	}()
	err = m.breaker.Do(context.Background(), funcName, call, m.GetFallbackFunc(funcName))
	return err
}

func (m *ClientWrapper) Call(ctx context.Context, hashKey, funcName string, run func(addr string) error) error {
	fun := "ClientWrapper.Call -->"

	si := m.router.Route(ctx, m.processor, hashKey)
	if si == nil {
		return fmt.Errorf("%s not find service:%s processor:%s", fun, m.clientLookup.ServPath(), m.processor)
	}
	m.router.Pre(si)
	defer m.router.Post(si)

	call := func(_ctx context.Context) error {
		return run(si.Addr)
	}

	var err error
	st := xtime.NewTimeStat()
	defer func() {
		gstat.Collector(bootup.GetServBase(), m.clientLookup.ServKey(), m.processor, st.Duration(), 0, si.ServId, funcName, err)
	}()
	err = m.breaker.Do(ctx, funcName, call, m.GetFallbackFunc(funcName))
	return err
}

func (m *ClientWrapper) SetConfigCenter(configCenter zconfig.ConfigCenter) {
	m.configCenter = configCenter
}

// getFuncTimeout get configured timout when invoking servKey/funcName.
// `defaultTime` will be returned if it's not configured
func (m *ClientWrapper) getFuncTimeout(servKey, funcName string, defaultTime time.Duration) time.Duration {
	var configCenter zconfig.ConfigCenter
	if m.configCenter != nil {
		configCenter = m.configCenter
	}
	return utils.GetFuncTimeoutInner(configCenter, servKey, funcName, defaultTime)
}

// getFuncRetry get configured retry times when invoking servKey/funcName.
func (m *ClientWrapper) getFuncRetry(servKey, funcName string) int {
	var configCenter zconfig.ConfigCenter
	if m.configCenter != nil {
		configCenter = m.configCenter
	}
	return utils.GetFuncRetryInner(configCenter, servKey, funcName)
}
