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
 @Time    : 2024/10/12 -- 15:26
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: xetcd.go
*/

package oneetcd

import (
	"context"
	"fmt"
	"github.com/qiguanzhu/infra/nerv/magi/xtime"
	"github.com/qiguanzhu/infra/nerv/xlog"
	"github.com/qiguanzhu/infra/nerv/xtrace"
	etcdClient "go.etcd.io/etcd/client/v2"
	"time"
)

const (
	spanLogKeyPath  = "path"
	spanLogKeyValue = "value"
	spanLogKeyTTL   = "ttl"
)

// EtcdInstance ...
type EtcdInstance struct {
	API etcdClient.KeysAPI
}

// NewEtcdInstanceWithAPI ...
func NewEtcdInstanceWithAPI(api etcdClient.KeysAPI) *EtcdInstance {
	return &EtcdInstance{
		API: api,
	}
}

// NewEtcdInstance ...
func NewEtcdInstance(cluster []string) (*EtcdInstance, error) {
	cfg := etcdClient.Config{
		Endpoints: cluster,
		Transport: etcdClient.DefaultTransport,
	}
	c, err := etcdClient.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("create etcd client cfg error")
	}
	api := etcdClient.NewKeysAPI(c)
	if api == nil {
		return nil, fmt.Errorf("create etcd api error")
	}
	return NewEtcdInstanceWithAPI(api), nil
}

// Get ...
func (m *EtcdInstance) Get(ctx context.Context, path string) (string, error) {
	fun := "xetcd.EtcdInstance.Get"
	span, ctx := xtrace.StartSpanFromContext(ctx, fun)
	defer span.Finish()
	span.LogFields(xtrace.String(spanLogKeyPath, path))

	r, err := m.API.Get(ctx, path, &etcdClient.GetOptions{
		Recursive: false,
		Sort:      false,
	})
	if err != nil {
		return "", err
	}

	if r.Node == nil {
		return "", fmt.Errorf("etcdIns node value err location:%s", path)
	}

	return r.Node.Value, nil
}

// GetNode ...
func (m *EtcdInstance) GetNode(ctx context.Context, path string) (*etcdClient.Node, error) {
	fun := "xetcd.EtcdInstance.GetNode"
	span, ctx := xtrace.StartSpanFromContext(ctx, fun)
	defer span.Finish()
	span.LogFields(xtrace.String(spanLogKeyPath, path))

	r, err := m.API.Get(ctx, path, &etcdClient.GetOptions{
		Recursive: true,
		Sort:      false,
	})
	if err != nil {
		return nil, err
	}

	if r.Node == nil {
		return nil, fmt.Errorf("etcdIns node value err location:%s", path)
	}

	return r.Node, nil
}

// Set ...
func (m *EtcdInstance) Set(ctx context.Context, path, val string) error {
	fun := "xetcd.EtcdInstance.Set"
	span, ctx := xtrace.StartSpanFromContext(ctx, fun)
	defer span.Finish()
	span.LogFields(xtrace.String(spanLogKeyPath, path),
		xtrace.String(spanLogKeyValue, val))

	r, err := m.API.Set(ctx, path, val, &etcdClient.SetOptions{})
	if err != nil {
		return err
	}

	if r.Node == nil {
		return fmt.Errorf("etcdIns node value err location:%s", path)
	}

	return nil
}

// CreateDir ...
func (m *EtcdInstance) CreateDir(ctx context.Context, path string) error {
	fun := "xetcd.EtcdInstance.CreateDir"
	span, ctx := xtrace.StartSpanFromContext(ctx, fun)
	defer span.Finish()
	span.LogFields(xtrace.String(spanLogKeyPath, path))

	_, err := m.API.Set(ctx, path, "", &etcdClient.SetOptions{
		Dir:       true,
		PrevExist: etcdClient.PrevNoExist,
	})
	if err != nil {
		return err
	}
	return nil
}

// SetTTL ...
func (m *EtcdInstance) SetTTL(ctx context.Context, path, val string, ttl time.Duration) error {
	fun := "xetcd.EtcdInstance.SetTTL"
	span, ctx := xtrace.StartSpanFromContext(ctx, fun)
	defer span.Finish()
	span.LogFields(xtrace.String(spanLogKeyPath, path),
		xtrace.String(spanLogKeyValue, val),
		xtrace.Int(spanLogKeyTTL, int(ttl.Seconds())))

	_, err := m.API.Set(ctx, path, val, &etcdClient.SetOptions{
		TTL: ttl,
	})
	if err != nil {
		return err
	}

	return nil
}

// RefreshTTL ...
func (m *EtcdInstance) RefreshTTL(ctx context.Context, path string, ttl time.Duration) error {
	fun := "xetcd.EtcdInstance.RefreshTTL"
	span, ctx := xtrace.StartSpanFromContext(ctx, fun)
	defer span.Finish()
	span.LogFields(xtrace.String(spanLogKeyPath, path),
		xtrace.Int(spanLogKeyTTL, int(ttl.Seconds())))

	_, err := m.API.Set(ctx, path, "", &etcdClient.SetOptions{
		PrevExist: etcdClient.PrevExist,
		Refresh:   true,
		TTL:       ttl,
	})
	if err != nil {
		return err
	}

	return nil
}

// SetNx ...
func (m *EtcdInstance) SetNx(ctx context.Context, path, val string) error {
	fun := "xetcd.EtcdInstance.SetNx"
	span, ctx := xtrace.StartSpanFromContext(ctx, fun)
	defer span.Finish()
	span.LogFields(xtrace.String(spanLogKeyPath, path),
		xtrace.String(spanLogKeyValue, val))

	_, err := m.API.Set(ctx, path, val, &etcdClient.SetOptions{
		PrevExist: etcdClient.PrevNoExist,
	})
	if err != nil {
		return err
	}

	return nil
}

// Reg ...
func (m *EtcdInstance) Reg(ctx context.Context, path, val string, heatbeat time.Duration, ttl time.Duration) error {
	fun := "xetcd.EtcdInstance.Reg"
	var isset = true
	go func() {
		for i := 0; ; i++ {
			var err error
			if isset {
				xlog.Warnf(ctx, "%s create idx:%d val:%s", fun, i, val)
				_, err = m.API.Set(ctx, path, val, &etcdClient.SetOptions{
					TTL: ttl,
				})
				if err == nil {
					isset = false
				}
			} else {
				xlog.Infof(ctx, "%s refresh ttl idx:%d val:%s", fun, i, val)
				_, err = m.API.Set(ctx, path, "", &etcdClient.SetOptions{
					PrevExist: etcdClient.PrevExist,
					TTL:       ttl,
					Refresh:   true,
				})
			}
			if err != nil {
				xlog.Errorf(ctx, "%s reg idx:%d err:%s", fun, i, err)

			}

			time.Sleep(heatbeat)
		}
	}()

	return nil
}

// Watch ...
func (m *EtcdInstance) Watch(ctx context.Context, path string, hander func(*etcdClient.Response)) {
	fun := "xetcd.EtcdInstance.Watch"
	backoff := xtime.NewBackOffCtrl(time.Millisecond*10, time.Second*5)
	var chg chan *etcdClient.Response
	go func() {
		xlog.Infof(ctx, "%s start watch:%s", fun, path)
		for {
			if chg == nil {
				xlog.Infof(ctx, "%s loop watch new receiver:%s", fun, path)
				chg = make(chan *etcdClient.Response)
				go m.startWatch(ctx, chg, path)
			}

			r, ok := <-chg
			if !ok {
				xlog.Errorf(ctx, "%s chg info nil:%s", fun, path)
				chg = nil
				backoff.BackOff()
			} else {
				xlog.Infof(ctx, "%s update path:%s", fun, r.Node.Key)
				hander(r)
				backoff.Reset()
			}
		}
	}()
}

func (m *EtcdInstance) startWatch(ctx context.Context, chg chan *etcdClient.Response, path string) {
	fun := "EtcdInstance.startWatch -->"
	for i := 0; ; i++ {
		r, err := m.API.Get(ctx, path, &etcdClient.GetOptions{Recursive: true, Sort: false})
		if err != nil {
			xlog.Warnf(ctx, "%s get path:%s err:%s", fun, path, err)
		} else {
			chg <- r
		}
		index := uint64(0)
		if r != nil {
			index = r.Index
			fmt.Printf("%s init get action:%s nodes:%d index:%d path:%s\n", fun, r.Action, len(r.Node.Nodes), r.Index, path)
		}

		wop := &etcdClient.WatcherOptions{
			Recursive:  true,
			AfterIndex: index,
		}
		watcher := m.API.Watcher(path, wop)
		if watcher == nil {
			// slog.Errorf(ctx, "%s new watcher path:%s", fun, path)
			return
		}

		resp, err := watcher.Next(context.Background())
		// etcdIns 关闭时候会返回
		if err != nil {
			fmt.Printf("%s watch path:%s err:%s\n", fun, path, err)
			close(chg)
			return
		}
		fmt.Printf("%s next get idx:%d action:%s nodes:%d index:%d after:%d path:%s\n", fun, i, resp.Action, len(resp.Node.Nodes), resp.Index, wop.AfterIndex, path)
		// 测试发现next获取到的返回，index，重新获取总有问题，触发两次，不确定，为什么？为什么？
		// 所以这里每次next前使用的afterindex都重新get了
	}

}
