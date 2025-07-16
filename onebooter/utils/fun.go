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
 @Time    : 2024/11/10 -- 16:58
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: fun.go
*/

package utils

import (
	"context"
	"gitee.com/go-mid/infra/xutil"
	"github.com/qiguanzhu/infra/lcl/governor/gentity"
	"github.com/qiguanzhu/infra/nerv/xcontext"
	"github.com/qiguanzhu/infra/seele/zconfig"
	"runtime"
	"strings"
	"time"
)

const (
	// Timeout timeout(ms)
	Timeout = "timeoutMsec"
	// Retry ...
	Retry = "retry"
	// Default ...
	Default = "Default"
)

// GetFuncNameWithCtx get fun name from context, if not set then use runtime caller
func GetFuncNameWithCtx(ctx context.Context, index int) string {
	var funcName string
	if method, ok := xcontext.GetCallerMethod(ctx); ok {
		return method
	}
	pc, _, _, ok := runtime.Caller(index)
	if ok {
		funcName = runtime.FuncForPC(pc).Name()
		if index := strings.LastIndex(funcName, "."); index != -1 {
			if len(funcName) > index+1 {
				funcName = funcName[index+1:]
			}
		}
	}
	return funcName
}

// GetFuncTimeout get func timeout conf
func GetFuncTimeout(confCenter zconfig.ConfigCenter, servKey, funcName string, defaultTime time.Duration) time.Duration {
	key := xutil.Concat(servKey, ".", funcName, ".", Timeout)
	var t int
	var exist bool
	if confCenter != nil {
		if t, exist = confCenter.GetIntWithNamespace(context.TODO(), gentity.RPCConfNamespace, key); !exist {
			defaultKey := xutil.Concat(servKey, ".", Default, ".", Timeout)
			t, _ = confCenter.GetIntWithNamespace(context.TODO(), gentity.RPCConfNamespace, defaultKey)
		}
	}
	if t == 0 {
		return defaultTime
	}

	return time.Duration(t) * time.Millisecond
}

// GetFuncRetry get func retry conf
func GetFuncRetry(confCenter zconfig.ConfigCenter, servKey, funcName string) int {
	key := xutil.Concat(servKey, ".", funcName, ".", Retry)
	var t int
	var exist bool
	if confCenter != nil {
		if t, exist = confCenter.GetIntWithNamespace(context.TODO(), gentity.RPCConfNamespace, key); !exist {
			defaultKey := xutil.Concat(servKey, ".", Default, ".", Retry)
			t, _ = confCenter.GetIntWithNamespace(context.TODO(), gentity.RPCConfNamespace, defaultKey)
		}
	}
	return t
}

// GetFuncTimeoutInner get configured timout when invoking servKey/funcName.
// `defaultTime` will be returned if it's not configured
func GetFuncTimeoutInner(confCenter zconfig.ConfigCenter, servKey, funcName string, defaultTime time.Duration) time.Duration {
	key := xutil.Concat(servKey, ".", funcName, ".", Timeout)
	var t int
	var exist bool
	if confCenter != nil {
		if t, exist = confCenter.GetIntWithNamespace(context.TODO(), gentity.RPCConfNamespace, key); !exist {
			defaultKey := xutil.Concat(servKey, ".", Default, ".", Timeout)
			t, _ = confCenter.GetIntWithNamespace(context.TODO(), gentity.RPCConfNamespace, defaultKey)
		}
	}
	if t == 0 {
		return defaultTime
	}

	return time.Duration(t) * time.Millisecond
}

// GetFuncRetryInner get configured retry times when invoking servKey/funcName.
func GetFuncRetryInner(confCenter zconfig.ConfigCenter, servKey, funcName string) int {
	key := xutil.Concat(servKey, ".", funcName, ".", Retry)
	var t int
	var exist bool
	if confCenter != nil {
		if t, exist = confCenter.GetIntWithNamespace(context.TODO(), gentity.RPCConfNamespace, key); !exist {
			defaultKey := xutil.Concat(servKey, ".", Default, ".", Retry)
			t, _ = confCenter.GetIntWithNamespace(context.TODO(), gentity.RPCConfNamespace, defaultKey)
		}
	}
	return t
}
