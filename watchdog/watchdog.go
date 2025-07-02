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
 @Time    : 2025/4/15 -- 12:08
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: watchdog watchdog/mod.go
*/

package watchdog

import (
	"context"
	"time"
)

type Woof interface {
	Locker
	Watch(ctx context.Context, k string, dur time.Duration)
}

type Locker interface {
	Lock(ctx context.Context, k string, v interface{}, dur time.Duration) error
	Unlock(ctx context.Context, k string) error
}
