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
 @Time    : 2024/11/18 -- 12:23
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: server.go
*/

package obgrpc

import (
	"context"
	"errors"
	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcRecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/opentracing/opentracing-go"
	"github.com/qiguanzhu/infra/gehirn/colorlog"
	"github.com/qiguanzhu/infra/gehirn/onebooter/obentity"
	"github.com/qiguanzhu/infra/lcl/governor"
	"github.com/qiguanzhu/infra/lcl/governor/gentity"
	"github.com/qiguanzhu/infra/lcl/governor/gstat"
	"github.com/qiguanzhu/infra/nerv/magi/xrate"
	"github.com/qiguanzhu/infra/nerv/magi/xtime"
	"github.com/qiguanzhu/infra/nerv/xcontext"
	"github.com/qiguanzhu/infra/nerv/xlog"
	xprom "github.com/qiguanzhu/infra/nerv/xstat/xmetric/xprometheus"
	"github.com/qiguanzhu/infra/nerv/xtransport/gen-go/util/thriftutil"
	"github.com/qiguanzhu/infra/seele/zserv"
	otgrpc "github.com/qiguanzhu/tracing/go-grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"runtime"
	"strings"
)

const logRequestKey = "log_request"

type printBodyMethod struct {
	LogRequestMethodList   []string `json:"log_request_method_list" properties:"log_request_method_list"`
	NoLogRequestMethodList []string `json:"no_log_request_method_list" properties:"no_log_request_method_list"`
}

type GrpcServer struct {
	userUnaryInterceptors  []grpc.UnaryServerInterceptor
	extraUnaryInterceptors []grpc.UnaryServerInterceptor // 服务启动之前, 内部添加的拦截器, 在所有拦截器之后添加
	Server                 *grpc.Server
	Base                   zserv.ServerSessionProxy[obentity.ServInfo]
}

type FunInterceptor func(ctx context.Context, req interface{}, fun string) error

// UnaryHandler 是grpc UnaryHandler的别名, 便于统一管理grpc升级
type UnaryHandler func(ctx context.Context, req interface{}) (interface{}, error)

// UnaryServerInterceptor 是grpc UnaryServerInterceptor的别名, 便于统一管理grpc升级
type UnaryServerInterceptor func(ctx context.Context, req interface{}, info *UnaryServerInfo, handler UnaryHandler) (interface{}, error)

// UnaryServerInfo 是grpc UnaryServerInfo的别名, 便于统一管理grpc升级
type UnaryServerInfo struct {
	// Server is the service implementation the user provides. This is read-only.
	Server interface{}
	// FullMethod is the full RPC method string, i.e., /package.service/method.
	FullMethod string
}

func NewGrpcServerWithUnaryInterceptors(interceptors ...UnaryServerInterceptor) *GrpcServer {
	userUnaryInterceptors := convertUnaryInterceptors(interceptors...)

	gServ := &GrpcServer{
		userUnaryInterceptors: userUnaryInterceptors,
	}

	s, err := gServ.buildServer()
	if err != nil {
		panic(err)
	}
	gServ.Server = s
	return gServ
}

func (g *GrpcServer) internalAddExtraInterceptors(extraInterceptors ...grpc.UnaryServerInterceptor) {
	g.extraUnaryInterceptors = append(g.extraUnaryInterceptors, extraInterceptors...)
}

func (g *GrpcServer) buildServer() (*grpc.Server, error) {
	var unaryInterceptors []grpc.UnaryServerInterceptor
	var streamInterceptors []grpc.StreamServerInterceptor

	// add tracer、monitor、recovery interceptor
	recoveryOpts := []grpcRecovery.Option{
		grpcRecovery.WithRecoveryHandler(recoveryFunc),
	}
	unaryInterceptors = append(unaryInterceptors,
		grpcRecovery.UnaryServerInterceptor(recoveryOpts...),
		otgrpc.OpenTracingServerInterceptorWithGlobalTracer(otgrpc.SpanDecorator(apmSetSpanTagDecorator)),
		rateLimitInterceptor(),
		monitorServerInterceptor(),
		laneInfoServerInterceptor(),
		headInfoServerInterceptor(),
	)
	userUnaryInterceptors := g.userUnaryInterceptors
	unaryInterceptors = append(unaryInterceptors, userUnaryInterceptors...)
	unaryInterceptors = append(unaryInterceptors, g.extraUnaryInterceptors...)

	streamInterceptors = append(streamInterceptors,
		rateLimitStreamServerInterceptor(),
		otgrpc.OpenTracingStreamServerInterceptorWithGlobalTracer(),
		monitorStreamServerInterceptor(),
		grpcRecovery.StreamServerInterceptor(recoveryOpts...),
	)

	var opts []grpc.ServerOption
	opts = append(opts, grpc.UnaryInterceptor(grpcMiddleware.ChainUnaryServer(unaryInterceptors...)))
	opts = append(opts, grpc.StreamInterceptor(grpcMiddleware.ChainStreamServer(streamInterceptors...)))

	// 实例化grpc Server
	server := grpc.NewServer(opts...)
	return server, nil
}

func apmSetSpanTagDecorator(ctx context.Context, span opentracing.Span, method string, req, resp interface{}, grpcError error) {
	var hasError bool
	if ctx.Err() != nil {
		span.SetTag("error.ctx", true)
		hasError = true
	}
	if grpcError != nil {
		span.SetTag("error.grpc", true)
		hasError = true
	}
	if hasError {
		span.SetTag("error", true)
	}
	// set instance info tags
	sb := GetServBase()
	if sb != nil {
		span.SetTag("region", sb.Region(ctx))
		span.SetTag("ip", sb.Ip(ctx))
		span.SetTag("lane", sb.Lane(ctx))
		span.SetTag("method", method)
	}
}

func DecorateServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {

		colorlog.Infof(ctx, "================== RPC 服务端装饰开始 ==================")
		colorlog.Infof(ctx, "req is: %+v", req)
		colorlog.Infof(ctx, "called: %s", info.FullMethod)

		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			colorlog.Infof(ctx, "metadata: %+v", md)
		}

		resp, err = handler(ctx, req)
		if err != nil {
			return resp, err
		}

		colorlog.Infof(ctx, "================== RPC 服务端装饰结束 ==================")
		return resp, err
	}
}

func convertUnaryInterceptors(interceptors ...UnaryServerInterceptor) []grpc.UnaryServerInterceptor {
	var ret []grpc.UnaryServerInterceptor
	for _, interceptor := range interceptors {
		ret = append(ret, convertUnaryInterceptor(interceptor))
	}
	return ret
}

func convertUnaryInterceptor(interceptor UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		return interceptor(ctx, req, convertUnaryServerInfo(info), convertUnaryHandler(handler))
	}
}

func convertUnaryHandler(handler grpc.UnaryHandler) UnaryHandler {
	return UnaryHandler(handler)
}

func convertUnaryServerInfo(info *grpc.UnaryServerInfo) *UnaryServerInfo {
	return &UnaryServerInfo{
		Server:     info.Server,
		FullMethod: info.FullMethod,
	}
}

// rate limiter interceptor, should be before OpenTracingServerInterceptor and monitorServerInterceptor
func rateLimitInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		parts := strings.Split(info.FullMethod, "/")
		interfaceName := parts[len(parts)-1]
		caller := governor.GetCallerFromBaggage(ctx)
		err = governor.RateLimitRegistry.InterfaceRateLimit(ctx, interfaceName, caller)
		if err != nil {
			if errors.Is(err, xrate.ErrRateLimited) {
				xlog.Warnf(ctx, "rate limited: method=%s, caller=%s", info.FullMethod, caller)
			}
			return nil, err
		} else {
			return handler(ctx, req)
		}
	}
}

// server rpc cost, record to log and prometheus
func monitorServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		group, service := GetGroupAndService()
		fun := info.FullMethod
		// TODO 先做兼容，后续再补上
		gstat.GetAPIRequestCountMetric().With(xprom.LabelGroupName, group, xprom.LabelServiceName, service, xprom.LabelAPI, fun, xprom.LabelErrCode, "1").Inc()
		st := xtime.NewTimeStat()
		resp, err = handler(ctx, req)
		if shouldLogRequest(info.FullMethod) {
			xlog.Infow(ctx, "", "func", fun, "req", req, "err", err, "cost", st.Millisecond(), "resp", resp)
		} else {
			xlog.Infow(ctx, "", "func", fun, "err", err, "cost", st.Millisecond())
		}
		gstat.GetAPIRequestTimeMetric().With(xprom.LabelGroupName, group, xprom.LabelServiceName, service, xprom.LabelAPI, fun, xprom.LabelErrCode, "1").Observe(float64(st.Millisecond()))
		return resp, err
	}
}

// rate limiter interceptor, should be before OpenTracingStreamServerInterceptor and monitorStreamServerInterceptor
func rateLimitStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := context.Background()
		parts := strings.Split(info.FullMethod, "/")
		interfaceName := parts[len(parts)-1]

		// 暂时不支持按照调用方限流
		caller := gentity.UNSPECIFIED_CALLER
		err := governor.RateLimitRegistry.InterfaceRateLimit(ctx, interfaceName, caller)
		if err != nil {
			if errors.Is(err, xrate.ErrRateLimited) {
				xlog.Warnf(ctx, "rate limited: method=%s, caller=%s", info.FullMethod, caller)
			}
			return err
		} else {
			return handler(srv, ss)
		}
	}
}

// stream server rpc cost, record to log and prometheus
func monitorStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		fun := info.FullMethod
		group, service := GetGroupAndService()
		// TODO 先做兼容，后续再补上
		gstat.GetAPIRequestCountMetric().With(xprom.LabelGroupName, group, xprom.LabelServiceName, service, xprom.LabelAPI, fun, xprom.LabelErrCode, "1").Inc()
		st := xtime.NewTimeStat()
		err := handler(srv, ss)
		if shouldLogRequest(info.FullMethod) {
			xlog.Infow(ss.Context(), "", "func", fun, "req", srv, "err", err, "cost", st.Millisecond())
		} else {
			xlog.Infow(ss.Context(), "", "func", fun, "err", err, "cost", st.Millisecond())
		}
		gstat.GetAPIRequestTimeMetric().With(xprom.LabelGroupName, group, xprom.LabelServiceName, service, xprom.LabelAPI, fun, xprom.LabelErrCode, "1").Observe(float64(st.Millisecond()))
		return err
	}
}

func laneInfoServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		var lane string
		lanes := md[gentity.LaneInfoMetadataKey]
		if len(lanes) >= 1 {
			lane = lanes[0]
		}

		route := thriftutil.NewRoute()
		route.Group = lane
		control := thriftutil.NewControl()
		control.Route = route

		ctx = context.WithValue(ctx, xcontext.ContextKeyControl, control)
		return handler(ctx, req)
	}
}

func headInfoServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp interface{}, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		var hlc, hiiiInfo string
		values := md[xcontext.ContextPropertiesKeyHLC]
		if len(values) >= 1 {
			hlc = values[0]
		}
		values = md[xcontext.ContextPropertiesKeyHiiiHeader]
		if len(values) >= 1 {
			hiiiInfo = values[0]
		}
		ctx = xcontext.SetHeaderPropertiesHLC(ctx, hlc)
		ctx = xcontext.SetPropertiesHiiiHeader(ctx, hiiiInfo)
		return handler(ctx, req)
	}
}

func recoveryFunc(p interface{}) (err error) {
	ctx := context.Background()
	const size = 4096
	buf := make([]byte, size)
	buf = buf[:runtime.Stack(buf, false)]
	xlog.Errorf(ctx, "%v catch panic, stack: %s", p, string(buf))
	return status.Errorf(codes.Internal, "panic triggered: %v", p)
}

func shouldLogRequest(fullMethod string) bool {
	// 默认打印
	methodName, err := getMethodName(fullMethod)
	if err != nil {
		return true
	}
	center := GetConfigCenter()
	if center == nil {
		return true
	}
	printBodyMethod := printBodyMethod{}

	// 方法配置
	_ = center.UnmarshalWithNamespace(context.Background(), gentity.RPCServerConfNamespace, &printBodyMethod)
	// 不打印的优先级更高
	if methodInList(methodName, printBodyMethod.NoLogRequestMethodList) {
		return false
	}
	if methodInList(methodName, printBodyMethod.LogRequestMethodList) {
		return true
	}

	// 全局配置
	isPrint, ok := center.GetBool(context.Background(), logRequestKey)
	if !ok {
		// 默认输出
		return true
	}
	return isPrint
}

// FullMethod is the full RPC method string, i.e., /package.service/method
func getMethodName(fullMethod string) (string, error) {
	arr := strings.Split(fullMethod, "/")
	if len(arr) < 3 {
		return "", errors.New("full method is invalid")
	}
	// 根据格式/package.service/method，切割后，取method
	return arr[2], nil
}

// 方法是否在列表中
func methodInList(name string, list []string) bool {
	for _, l := range list {
		if name == l {
			return true
		}
	}
	return false
}
