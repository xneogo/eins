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
 @Time    : 2024/10/28 -- 18:05
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: registry.go
*/

package oneetcd

import (
	"context"
	"github.com/qiguanzhu/infra/lcl/governor/gregistry/retcd"
	"github.com/qiguanzhu/infra/nerv/xetcd"
	"github.com/qiguanzhu/infra/nerv/xlog"
	"github.com/qiguanzhu/infra/seele/zregistry"
	"time"
)

var DefaultEtcdInstance *xetcd.EtcdInstance
var DefaultRegister zregistry.Register

func init() {
	defaultEtcdInstance, err := xetcd.NewEtcdInstance(retcd.ETCDS_CLUSTER_0)
	if err != nil {
		xlog.Errorf(context.Background(), "init default etcd instance error: %v", err)
		return
	}
	DefaultRegister = &EtcdClient{
		defaultEtcdInstance,
	}
}

func Get(ctx context.Context, path string) (string, error) {
	return DefaultRegister.Get(ctx, path)
}
func GetNode(ctx context.Context, path string) (zregistry.Node, error) {
	return DefaultRegister.GetNode(ctx, path)
}
func Set(ctx context.Context, path, val string) error {
	return DefaultRegister.Set(ctx, path, val)
}
func CreateDir(ctx context.Context, path string) error {
	return DefaultRegister.CreateDir(ctx, path)
}
func SetTtl(ctx context.Context, path, val string, ttl time.Duration) error {
	return DefaultRegister.SetTtl(ctx, path, val, ttl)
}
func RefreshTtl(ctx context.Context, path string, ttl time.Duration) error {
	return DefaultRegister.RefreshTtl(ctx, path, ttl)
}
func SetNx(ctx context.Context, path, val string) error {
	return DefaultRegister.SetNx(ctx, path, val)
}
func Reg(ctx context.Context, path, val string, heatbeat time.Duration, ttl time.Duration) error {
	return DefaultRegister.Reg(ctx, path, val, heatbeat, ttl)
}
func Watch(ctx context.Context, path string, hander func(zregistry.Handler)) {
	DefaultRegister.Watch(ctx, path, hander)
}
