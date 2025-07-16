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
 @Time    : 2024/11/4 -- 16:47
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: power.go
*/

package onebooter

import (
	"context"
	"errors"
	"fmt"
	"github.com/qiguanzhu/infra/gehirn/onebooter/obgin"
	"github.com/qiguanzhu/infra/gehirn/onebooter/obgrpc"
	"github.com/qiguanzhu/infra/lcl/governor/gentity"
	"github.com/qiguanzhu/infra/lcl/governor/gmid"
	"github.com/qiguanzhu/infra/lcl/procimp"
	"github.com/qiguanzhu/infra/seele/zconfig"
	"github.com/qiguanzhu/infra/seele/zserv"
	"net"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/julienschmidt/httprouter"
	"github.com/qiguanzhu/infra/nerv/magi/xnet"
	"github.com/qiguanzhu/infra/nerv/xlog"
	"github.com/qiguanzhu/infra/nerv/xtrace"
	"github.com/qiguanzhu/tracing/go-stdlib/nethttp"
)

var ErrNilDriver = errors.New("nil driver")

type httpMiddleware func(next http.Handler) http.Handler

type DriverBuilder struct {
	c zconfig.ConfigCenter
}

func NewDriverBuilder(c zconfig.ConfigCenter) *DriverBuilder {
	return &DriverBuilder{
		c: c,
	}
}

func (dr *DriverBuilder) PowerProcessorDriver(ctx context.Context, n string, p zserv.ZProcessor, bs zserv.ServerSessionProxy[gentity.ServInfo]) (*gentity.ServInfo, error) {
	fun := "DriverBuilder.powerProcessorDriver -> "
	addr, driver := p.Driver()
	if driver == nil {
		return nil, ErrNilDriver
	}

	xlog.Infof(ctx, "%s processor:%s type:%s addr:%s", fun, n, reflect.TypeOf(driver), addr)

	switch d := driver.(type) {
	case *httprouter.Router:
		var extraHttpMiddlewares []httpMiddleware
		extraHttpMiddlewares = append(extraHttpMiddlewares, gmid.MetricMiddleware(GetGroupAndService()))
		sa, err := powerHttpWithMiddleware(addr, d, extraHttpMiddlewares...)
		if err != nil {
			return nil, err
		}
		servInfo := &gentity.ServInfo{
			Type:       gentity.PROCESSOR_HTTP,
			Addr:       sa,
			ServiceKey: bs.ServiceKey(ctx),
		}
		return servInfo, nil

	case *obgrpc.GrpcServer:
		// 添加内部拦截器的操作必须放到NewServer中, 否则无法在服务代码中完成service注册
		sa, err := powerGrpc(addr, d)
		if err != nil {
			return nil, err
		}
		servInfo := &gentity.ServInfo{
			Type:       gentity.PROCESSOR_GRPC,
			Addr:       sa,
			ServiceKey: bs.ServiceKey(ctx),
		}
		return servInfo, nil

	case *gin.Engine:
		sa, err := powerGin(addr, d)
		if err != nil {
			return nil, err
		}

		xlog.Infof(ctx, "%s load ok processor:%s serv addr:%s", fun, n, sa)
		servInfo := &gentity.ServInfo{
			Type:       gentity.PROCESSOR_GIN,
			Addr:       sa,
			ServiceKey: bs.ServiceKey(ctx),
		}
		return servInfo, nil

	case *obgin.HttpServer:
		sa, err := powerGin(addr, d.Engine)
		if err != nil {
			return nil, err
		}

		xlog.Infof(ctx, "%s load ok processor:%s serv addr:%s", fun, n, sa)
		servInfo := &gentity.ServInfo{
			Type:       gentity.PROCESSOR_GIN,
			Addr:       sa,
			ServiceKey: bs.ServiceKey(ctx),
		}
		return servInfo, nil
	case *procimp.FakeOne:
		sa, err := powerFake(addr, d)
		if err != nil {
			return nil, err
		}

		xlog.Infof(ctx, "%s load ok processor:%s serv addr:%s", fun, n, sa)
		servInfo := &gentity.ServInfo{
			Type:       gentity.PROCESSOR_Fake,
			Addr:       "",
			ServiceKey: bs.ServiceKey(ctx),
		}
		return servInfo, nil

	default:
		return nil, fmt.Errorf("processor:%s driver not recognition", n)
	}
}

func powerHttpWithMiddleware(addr string, router *httprouter.Router, middlewares ...httpMiddleware) (string, error) {
	fun := "powerHttpWithMiddleware -->"
	ctx := context.Background()

	netListen, laddr, err := listenServAddr(ctx, addr)
	if err != nil {
		return "", err
	}

	// tracing
	mw := decorateHttpMiddleware(router, middlewares...)

	go func() {
		err := http.Serve(netListen, mw)
		if err != nil {
			xlog.Panicf(ctx, "%s laddr[%s]", fun, laddr)
		}
	}()

	return laddr, nil
}

func powerHttp(addr string, router *httprouter.Router) (string, error) {
	fun := "powerHttp -->"
	ctx := context.Background()

	paddr, err := xnet.GetListenAddr(addr)
	if err != nil {
		return "", err
	}

	xlog.Infof(ctx, "%s config addr[%s]", fun, paddr)

	tcpAddr, err := net.ResolveTCPAddr("tcp", paddr)
	if err != nil {
		return "", err
	}

	netListen, err := net.Listen(tcpAddr.Network(), tcpAddr.String())
	if err != nil {
		return "", err
	}

	laddr, err := xnet.GetServAddr(netListen.Addr())
	if err != nil {
		_ = netListen.Close()
		return "", err
	}

	xlog.Infof(ctx, "%s listen addr[%s]", fun, laddr)

	// tracing
	mw := nethttp.Middleware(
		xtrace.GlobalTracer(),
		// add logging middleware
		gmid.TrafficLogMiddleware(router),
		nethttp.OperationNameFunc(func(r *http.Request) string {
			return "HTTP " + r.Method + ": " + r.URL.Path
		}),
		nethttp.MWSpanFilter(xtrace.UrlSpanFilter))

	go func() {
		err := http.Serve(netListen, mw)
		if err != nil {
			xlog.Panicf(ctx, "%s laddr[%s]", fun, laddr)
		}
	}()

	return laddr, nil
}

// powerGrpc 启动grpc ，并返回端口信息
func powerGrpc(addr string, server *GrpcServer) (string, error) {
	fun := "powerGrpc -->"
	ctx := context.Background()
	paddr, err := xnet.GetListenAddr(addr)
	if err != nil {
		return "", err
	}
	xlog.Infof(ctx, "%s config addr[%s]", fun, paddr)
	lis, err := net.Listen("tcp", paddr)
	if err != nil {
		return "", fmt.Errorf("grpc tcp Listen err:%v", err)
	}
	laddr, err := xnet.GetServAddr(lis.Addr())
	if err != nil {
		return "", fmt.Errorf(" GetServAddr err:%v", err)
	}
	xlog.Infof(ctx, "%s listen grpc addr[%s]", fun, laddr)
	go func() {
		if err := server.Server.Serve(lis); err != nil {
			xlog.Panicf(ctx, "%s grpc laddr[%s]", fun, laddr)
		}
	}()
	return laddr, nil
}

func powerGin(addr string, router *gin.Engine) (string, error) {
	fun := "powerGin -->"
	ctx := context.Background()

	paddr, err := xnet.GetListenAddr(addr)
	if err != nil {
		return "", err
	}

	xlog.Infof(ctx, "%s config addr[%s]", fun, paddr)

	tcpAddr, err := net.ResolveTCPAddr("tcp", paddr)
	if err != nil {
		return "", err
	}

	netListen, err := net.Listen(tcpAddr.Network(), tcpAddr.String())
	if err != nil {
		return "", err
	}

	laddr, err := xnet.GetServAddr(netListen.Addr())
	if err != nil {
		_ = netListen.Close()
		return "", err
	}

	xlog.Infof(ctx, "%s listen addr[%s]", fun, laddr)

	// tracing
	mw := nethttp.Middleware(
		xtrace.GlobalTracer(),
		gmid.TrafficLogMiddleware(router),
		nethttp.OperationNameFunc(func(r *http.Request) string {
			return "HTTP " + r.Method + ": " + r.URL.Path
		}),
		nethttp.MWSpanFilter(xtrace.UrlSpanFilter))

	serv := &http.Server{Handler: mw}
	go func() {
		err := serv.Serve(netListen)
		if err != nil {
			xlog.Panicf(ctx, "%s laddr[%s]", fun, laddr)
		}
	}()

	return laddr, nil
}

// powerFake 启动fake grpc ，并返回端口信息
func powerFake(addr string, server *procimp.FakeOne) (string, error) {
	fun := "powerFake -->"
	ctx := context.Background()
	paddr, err := xnet.GetListenAddr(addr)
	if err != nil {
		return "", err
	}
	xlog.Infof(ctx, "%s config addr[%s]", fun, paddr)
	lis, err := net.Listen("tcp", paddr)
	if err != nil {
		return "", fmt.Errorf("grpc tcp Listen err:%v", err)
	}
	laddr, err := xnet.GetServAddr(lis.Addr())
	if err != nil {
		return "", fmt.Errorf(" GetServAddr err:%v", err)
	}
	xlog.Infof(ctx, "%s listen grpc addr[%s]", fun, laddr)
	go func() {
		if err := server.Server.Serve(lis); err != nil {
			xlog.Panicf(ctx, "%s grpc laddr[%s]", fun, laddr)
		}
	}()
	return laddr, nil
}

func ReloadRouter(processor string, server interface{}, driver interface{}) error {
	fun := "reloadRouter -->"

	s, ok := server.(*http.Server)
	if !ok {
		return fmt.Errorf("server type error")
	}

	switch router := driver.(type) {
	case *gin.Engine:
		mw := nethttp.Middleware(
			xtrace.GlobalTracer(),
			router,
			nethttp.OperationNameFunc(func(r *http.Request) string {
				return "HTTP " + r.Method + ": " + r.URL.Path
			}))
		s.Handler = mw
		xlog.Infof(context.Background(), "%s reload ok, processors:%s", fun, processor)
	default:
		return fmt.Errorf("processor:%s driver not recognition", processor)
	}

	return nil
}

// 添加http middleware
func decorateHttpMiddleware(router http.Handler, middlewares ...httpMiddleware) http.Handler {
	r := router
	for _, m := range middlewares {
		r = m(r)
	}
	// tracing
	mw := nethttp.MiddlewareWithGlobalTracer(
		// add logging middleware
		gmid.TrafficLogMiddleware(r),
		nethttp.OperationNameFunc(func(r *http.Request) string {
			return "HTTP " + r.Method + ": " + r.URL.Path
		}),
		nethttp.MWSpanFilter(func(r *http.Request) bool {
			return true
		}))

	return mw
}

// 打开端口监听, 并返回服务地址
func listenServAddr(ctx context.Context, addr string) (net.Listener, string, error) {
	fun := "listenServAddr --> "
	paddr, err := xnet.GetListenAddr(addr)
	if err != nil {
		return nil, "", err
	}

	xlog.Infof(ctx, "%s config addr[%s]", fun, paddr)

	tcpAddr, err := net.ResolveTCPAddr("tcp", paddr)
	if err != nil {
		return nil, "", err
	}

	netListen, err := net.Listen(tcpAddr.Network(), tcpAddr.String())
	if err != nil {
		return nil, "", err
	}

	laddr, err := xnet.GetServAddr(netListen.Addr())
	if err != nil {
		_ = netListen.Close()
		return nil, "", err
	}

	xlog.Infof(ctx, "%s listen addr[%s]", fun, laddr)
	return netListen, laddr, nil
}
