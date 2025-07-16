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
 @Time    : 2024/10/28 -- 18:02
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: etcd.go
*/

package oneetcd

import (
	"context"
	"github.com/qiguanzhu/infra/nerv/xetcd"
	"github.com/qiguanzhu/infra/seele/zregistry"
	"go.etcd.io/etcd/client/v2"
	"time"
)

type EtcdClient struct {
	*xetcd.EtcdInstance
}

func (m *EtcdClient) SetTtl(ctx context.Context, path, val string, ttl time.Duration) error {
	return m.EtcdInstance.SetTTL(ctx, path, val, ttl)
}

func (m *EtcdClient) RefreshTtl(ctx context.Context, path string, ttl time.Duration) error {
	return m.EtcdInstance.RefreshTTL(ctx, path, ttl)
}

func (m *EtcdClient) GetNode(ctx context.Context, path string) (zregistry.Node, error) {
	node, err := m.EtcdInstance.GetNode(ctx, path)
	return &EtcdNode{
		node,
	}, err
}

func (m *EtcdClient) Watch(ctx context.Context, path string, hander func(zregistry.Handler)) {
	m.EtcdInstance.Watch(ctx, path, func(response *client.Response) {
		hander(&EtcHandler{
			Response: response,
		})
	})
}

type EtcdNode struct {
	*client.Node
}

func (m *EtcdNode) IsDir() bool {
	return m.Node.Dir
}

func (m *EtcdNode) Key() string {
	return m.Node.Key
}

func (m *EtcdNode) Value() string {
	return m.Node.Value
}

func (m *EtcdNode) Ttl() int64 {
	return m.Node.TTL
}

func (m *EtcdNode) Expiration() *time.Time {
	return m.Node.Expiration
}

func (m *EtcdNode) Children() []zregistry.Node {
	nodes := make([]zregistry.Node, 0, len(m.Nodes))
	for _, n := range m.Nodes {
		nodes = append(nodes, &EtcdNode{
			n,
		})
	}
	return nodes
}

type EtcHandler struct {
	*client.Response
}

func (m *EtcHandler) Action() string {
	return m.Response.Action
}

func (m *EtcHandler) Node() zregistry.Node {
	return &EtcdNode{
		m.Response.Node,
	}
}
