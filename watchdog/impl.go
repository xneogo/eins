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
 @Time    : 2025/6/25 -- 18:16
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: watchdog watchdog/impl.go
*/

package watchdog

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type WatchDog struct {
	rds    *redis.Client
	c      context.Context
	cancel context.CancelFunc
}

func (w *WatchDog) Lock(ctx context.Context, k string, v interface{}, dur time.Duration) error {
	res := w.rds.SetNX(ctx, k, v, dur)
	if res.Err() != nil || !res.Val() {
		return res.Err()
	}

	w.c, w.cancel = context.WithCancel(ctx)
	w.Watch(w.c, k, dur)
	return nil
}

func (w *WatchDog) Unlock(ctx context.Context, k string) error {
	if w.cancel != nil {
		w.cancel()
	}
	return w.rds.Del(ctx, k).Err()
}

func (w *WatchDog) Watch(ctx context.Context, k string, dur time.Duration) {
	go func() {
		ticker := time.NewTicker(dur / 7)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				res := w.rds.Expire(ctx, k, dur)
				if res.Err() != nil || !res.Val() {
					return
				}
			}
		}
	}()
}
