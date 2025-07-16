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
 @Time    : 2024/10/28 -- 16:59
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: client_grpc.go
*/

package onebooter

import (
	"context"
	"errors"
	"fmt"
	"github.com/qiguanzhu/infra/lcl/bootup/utils"
	"github.com/qiguanzhu/infra/lcl/governor/gmid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"time"

	"github.com/qiguanzhu/infra/lcl/governor/clientpool"
	"github.com/qiguanzhu/infra/lcl/governor/fallback"
	"github.com/qiguanzhu/infra/lcl/governor/gentity"
	"github.com/qiguanzhu/infra/lcl/governor/gstat"
	"github.com/qiguanzhu/infra/nerv/magi/xtime"
	"github.com/qiguanzhu/infra/nerv/xcontext"
	"github.com/qiguanzhu/infra/nerv/xlog"
	"github.com/qiguanzhu/infra/nerv/xtrace"
	"github.com/qiguanzhu/infra/seele/zconfig"
	"github.com/qiguanzhu/infra/seele/zserv"
	otgrpc "github.com/qiguanzhu/tracing/go-grpc"
	"github.com/uber/jaeger-client-go"
)

// ClientGrpc client of grpc in adapter
type ClientGrpc struct {
	fallback.Fallbacks

	clientLookup zserv.ClientLookup[gentity.ServInfo]
	configCenter zconfig.ConfigCenter
	processor    string
	breaker      *utils.Breaker
	router       utils.Router

	pool      *clientpool.ClientPool
	fnFactory func(conn *grpc.ClientConn) interface{}
}

type Provider struct {
	Ip   string
	Port uint16
}

// NewClientGrpcWithRouterType create grpc client by routerType, fn: xxServiceClient of xx, such as NewChangeBoardServiceClient
func NewClientGrpcWithRouterType(cb zserv.ClientLookup[gentity.ServInfo], processor string, capacity int, fn func(client *grpc.ClientConn) interface{}, routerType int) *ClientGrpc {
	clientGrpc := &ClientGrpc{
		clientLookup: cb,
		processor:    processor,
		breaker:      utils.NewBreaker(cb),
		router:       utils.NewRouter(routerType, cb),
		fnFactory:    fn,
	}
	// 目前为写死值，后期改为动态配置获取的方式
	pool := clientpool.NewClientPool(clientpool.DefaultMaxIdle, clientpool.DefaultMaxActive, clientGrpc.newConn, cb.ServKey())
	clientGrpc.pool = pool
	cb.AppendEventHandler(clientGrpc.deleteAddrHandler)
	return clientGrpc
}

func (m *ClientGrpc) deleteAddrHandler(addrs []string) {
	for _, addr := range addrs {
		deleteAddrFromConnPool(addr, m.pool)
	}
}

func (m *ClientGrpc) CustomizedRouteRpc(getProvider func() *Provider, fnRpc func(interface{}) error) error {
	if getProvider == nil {
		return errors.New("fun getProvider is nil")
	}
	provider := getProvider()
	return m.DirectRouteRpc(provider, fnRpc)
}

func (m *ClientGrpc) DirectRouteRpc(provider *Provider, fnRpc func(interface{}) error) error {
	if provider == nil {
		return errors.New("get Provider is nil")
	}
	si, rc, e := m.getClient(provider)
	if e != nil {
		return e
	}
	if rc == nil {
		return fmt.Errorf("not find thrift service:%s processor:%s", m.clientLookup.ServPath(), m.processor)
	}
	_ = m.router.Pre(si)
	defer m.router.Post(si)

	fnRpcWrap := func(_ctx context.Context, in interface{}) error {
		return fnRpc(in)
	}
	call := func(_ctx context.Context) error {
		return m.rpcWithContext(_ctx, si, rc, fnRpcWrap)
	}

	funcName := utils.GetFuncNameWithCtx(context.Background(), 3)
	var err error
	st := xtime.NewTimeStat()
	defer func() {
		gstat.Collector(GetServBase(), m.clientLookup.ServKey(), m.processor, st.Duration(), 0, si.ServId, funcName, err)
	}()
	err = m.breaker.Do(context.Background(), funcName, call, m.GetFallbackFunc(funcName))
	return err
}

func (m *ClientGrpc) getClient(provider *Provider) (*gentity.ServInfo, clientpool.RpcClientConn, error) {
	servInfos := m.clientLookup.GetAllServAddr(m.processor)
	if len(servInfos) < 1 {
		return nil, nil, errors.New(m.processor + " server provider is empty ")
	}
	var serv *gentity.ServInfo
	addr := fmt.Sprintf("%s:%d", provider.Ip, provider.Port)
	for _, item := range servInfos {
		if item.Addr == addr {
			serv = item
			break
		}
	}
	if serv == nil {
		return nil, nil, errors.New(m.processor + " server provider is empty")
	}
	conn, err := m.pool.Get(context.Background(), serv.Addr)
	return serv, conn, err
}

func (m *ClientGrpc) RpcWithContext(ctx context.Context, hashKey string, fnRpc func(context.Context, interface{}) error) error {
	var err error
	funcName := utils.GetFuncNameWithCtx(ctx, 3)
	retry := m.getFuncRetry(m.clientLookup.ServKey(), funcName)
	for ; retry >= 0; retry-- {
		err = m.doWithContext(ctx, hashKey, funcName, fnRpc)
		if err == nil {
			return nil
		}
	}
	return err
}

func (m *ClientGrpc) doWithContext(ctx context.Context, hashKey, funcName string, fnRpc func(context.Context, interface{}) error) error {
	var err error
	si, rc := m.route(ctx, hashKey)
	if rc == nil {
		return fmt.Errorf("not find grpc service:%s processor:%s", m.clientLookup.ServPath(), m.processor)
	}
	defer func() {
		m.pool.Put(ctx, si.Addr, rc, err)
	}()

	ctx = m.injectServInfo(ctx, si)

	_ = m.router.Pre(si)
	defer m.router.Post(si)

	call := func(ctx context.Context) error {
		return m.rpcWithContext(ctx, si, rc, fnRpc)
	}

	st := xtime.NewTimeStat()
	defer func() {
		dur := st.Duration()
		gstat.Collector(GetServBase(), m.clientLookup.ServKey(), m.processor, dur, 0, si.ServId, funcName, err)
		gstat.CollectAPM(ctx, GetServName(), m.clientLookup.ServKey(), funcName, si.ServId, dur, err)
	}()
	err = m.breaker.Do(ctx, funcName, call, m.GetFallbackFunc(funcName))
	return err
}

func (m *ClientGrpc) rpcWithContext(ctx context.Context, si *gentity.ServInfo, rc clientpool.RpcClientConn, fnRpc func(context.Context, interface{}) error) error {
	c := rc.GetServiceClient()
	err := fnRpc(ctx, c)
	return err
}

func (m *ClientGrpc) route(ctx context.Context, key string) (*gentity.ServInfo, clientpool.RpcClientConn) {
	s := m.router.Route(ctx, m.processor, key)
	if s == nil {
		return nil, nil
	}
	addr := s.Addr
	conn, _ := m.pool.Get(ctx, addr)
	return s, conn
}

func (m *ClientGrpc) injectServInfo(ctx context.Context, si *gentity.ServInfo) context.Context {
	// fixme 此处set失败，不应影响后面的流程, 可能影响后续 stat 中 service & group 的标记
	ctx, _ = xcontext.SetControlCallerServerName(ctx, gmid.ServiceFromServPath(m.clientLookup.ServPath()))

	ctx, _ = xcontext.SetControlCallerServerID(ctx, fmt.Sprint(si.ServId))

	span := xtrace.SpanFromContext(ctx)
	if span == nil {
		return ctx
	}
	// 传入自己的servName进去
	if sName, ok := xcontext.GetControlCallerServerName(ctx); ok {
		span.SetBaggageItem(gentity.BaggageCallerKey, sName)
	}

	if jaegerSpan, ok := span.(*jaeger.Span); ok {
		ctx, _ = xcontext.SetControlCallerMethod(ctx, jaegerSpan.OperationName())
	}
	return ctx
}

func (m *ClientGrpc) SetConfigCenter(configCenter zconfig.ConfigCenter) {
	m.configCenter = configCenter
}

// getFuncTimeout get configured timout when invoking servKey/funcName.
// `defaultTime` will be returned if it's not configured
func (m *ClientGrpc) getFuncTimeout(servKey, funcName string, defaultTime time.Duration) time.Duration {
	var configCenter zconfig.ConfigCenter
	if m.configCenter != nil {
		configCenter = m.configCenter
	}
	return utils.GetFuncTimeoutInner(configCenter, servKey, funcName, defaultTime)
}

// getFuncRetry get configured retry times when invoking servKey/funcName.
func (m *ClientGrpc) getFuncRetry(servKey, funcName string) int {
	var configCenter zconfig.ConfigCenter
	if m.configCenter != nil {
		configCenter = m.configCenter
	}
	return utils.GetFuncRetryInner(configCenter, servKey, funcName)
}

type grpcClientConn struct {
	serviceClient interface{}
	conn          *grpc.ClientConn
}

func (m *grpcClientConn) SetTimeout(timeout time.Duration) error {
	return fmt.Errorf("SetTimeout is not support ")
}

func (m *grpcClientConn) Close() error {
	if m.conn != nil {
		_ = m.conn.Close()
	}
	return nil
}

func (m *grpcClientConn) GetServiceClient() interface{} {
	return m.serviceClient
}

// factory function in client connection pool
func (m *ClientGrpc) newConn(addr string) (clientpool.RpcClientConn, error) {
	fun := "ClientGrpc.newConn-->"

	// 可加入多种拦截器
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// 有序
		grpc.WithChainUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptorWithGlobalTracer(otgrpc.SpanDecorator(apmSetSpanTagDecorator)),
			laneInfoUnaryClientInterceptor(),
			contextHeadUnaryClientInterceptor(),
		),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptorWithGlobalTracer(),
		),
		// grpc.WithUnaryInterceptor(),
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		xlog.Errorf(context.Background(), "%s dial addr: %s failed, err: %v", fun, addr, err)
		return nil, err
	}
	client := m.fnFactory(conn)
	return &grpcClientConn{
		serviceClient: client,
		conn:          conn,
	}, nil
}

func laneInfoUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption) error {
		lane, _ := xcontext.GetControlRouteGroup(ctx)
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}
		md.Set(gentity.LaneInfoMetadataKey, lane)
		return invoker(metadata.NewOutgoingContext(ctx, md), method, req, reply, cc, opts...)
	}
}

func contextHeadUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption) error {

		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}
		hlc, ok := xcontext.GetPropertiesHLC(ctx)
		if ok {
			md.Set(xcontext.ContextPropertiesKeyHLC, hlc)
		}
		hiiiInfo, ok := xcontext.GetPropertiesHiiiHeader(ctx)
		if ok {
			md.Set(xcontext.ContextPropertiesKeyHiiiHeader, hiiiInfo)
		}
		return invoker(metadata.NewOutgoingContext(ctx, md), method, req, reply, cc, opts...)
	}
}

func deleteAddrFromConnPool(addr string, pool *clientpool.ClientPool) {
	fun := "deleteAddrFromConnPool -->"
	xlog.Infof(context.Background(), "%s get change addr success", fun)
	pool.Mu.Lock()
	defer pool.Mu.Unlock()
	value, ok := pool.Pool.Load(addr)
	if !ok {
		return
	}
	clientPool, ok := value.(*clientpool.ConnectionPool)
	if !ok {
		xlog.Warnf(context.Background(), "%s value to connection pool false, key: %s", fun, addr)
		return
	}
	pool.Pool.Delete(addr)
	clientPool.Close()
	xlog.Infof(context.Background(), "%s close client pool success, addr: %s", fun, addr)
}
