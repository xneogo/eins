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
 @Time    : 2024/11/4 -- 16:28
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: service.go
*/

package onebooter

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/qiguanzhu/infra/lcl/governor/gentity"
	"github.com/qiguanzhu/infra/lcl/governor/gregistry/retcd"
	"github.com/qiguanzhu/infra/nerv/magi/xsync"
	"github.com/qiguanzhu/infra/nerv/xconfig"
	"github.com/qiguanzhu/infra/nerv/xconfig/xapollo"
	"github.com/qiguanzhu/infra/nerv/xcontext"
	"github.com/qiguanzhu/infra/nerv/xlog"
	stat "github.com/qiguanzhu/infra/nerv/xstat/sys"
	"github.com/qiguanzhu/infra/nerv/xtransport/gen-go/util/thriftutil"
	"github.com/qiguanzhu/infra/pkg/consts"
	"github.com/qiguanzhu/infra/seele/zconfig"
	"github.com/qiguanzhu/infra/seele/zserv"
	etcd "go.etcd.io/etcd/client/v2"
	"reflect"
	"runtime"
	"strings"
)

// BaseServer 项目启动启动基地车，定义服务启动的相关配置以及基础实现函数
type BaseServer struct {
	retcd.CrossDcRegistry

	configCenter zconfig.ConfigCenter

	// 这两个字段和注册流程相关度更高，因此放到 etcd 注册最近的结构体中
	// servId       int    // 注册 服务id
	// servLocation string // 注册 location

	servGroup string
	servName  string
	servIp    string
	copyName  string
	sessKey   string
	envGroup  string
	region    string // 地区, 与PaaS一致

	localFlag gentity.RunningAt

	preServiceInitFns  []initFn
	postServiceInitFns []initFn

	muService xsync.Mutex
	servInfos map[string]*gentity.ServInfo // guarded by muService
	services  map[string]zserv.ZProcessor  // guarded by muService

	muLocks xsync.Mutex
	locks   map[string]*xsync.Semaphore // guarded by muLocks

	muHearts xsync.Mutex
	hearts   map[string]*distLockHeart // guarded by muHearts
}

// Register key is processor to ServInfo
func (m *BaseServer) Register(ctx context.Context, svcMap map[string]*gentity.ServInfo, dir string) error {
	return m.CrossDcRegistry.RegisterService(svcMap, dir, m.envGroup, false)
}

// RegisterService SvcLocRegType Service by default
func (m *BaseServer) RegisterService(ctx context.Context, svcMap map[string]*gentity.ServInfo, dir string, cross bool) error {
	return m.CrossDcRegistry.RegisterService(svcMap, dir, m.envGroup, cross)
}

// RegisterCrossDCService SvcLocRegType Service by default
func (m *BaseServer) RegisterCrossDCService(ctx context.Context, svcMap map[string]*gentity.ServInfo, dir string) error {
	return errors.New("not implemented")
}

func (m *BaseServer) Name(ctx context.Context) string       { return m.servName }     // eg: user | introduction
func (m *BaseServer) Group(ctx context.Context) string      { return m.servGroup }    // base | market
func (m *BaseServer) ServiceKey(ctx context.Context) string { return m.GetServKey() } // base/user | market/introduction
func (m *BaseServer) Lane(ctx context.Context) string       { return m.envGroup }     // 泳道 和 regInfo.Lane 有什么区别？？
func (m *BaseServer) Region(ctx context.Context) string     { return m.region }       // region
func (m *BaseServer) Ip(ctx context.Context) string         { return m.servIp }       // ip
func (m *BaseServer) Id(ctx context.Context) int            { return m.GetServId() }  // e.g: 1
func (m *BaseServer) ServInfos(ctx context.Context) map[string]*gentity.ServInfo {
	m.muService.Lock()
	defer m.muService.Unlock()
	return m.servInfos
} // serv infos

func (m *BaseServer) FullName(ctx context.Context) string {
	return fmt.Sprintf("%s%d", m.GetServKey(), m.GetServId())
} // eg: trade/points1

// Startup 启动服务
func (m *BaseServer) Startup(ctx context.Context, argsIn interface{}) error {
	fun := "BaseServer.Startup -->"
	args := argsIn.(*cmdArgs)
	if args.sessKey == "" {
		return errors.New("sessKey is empty")
	}

	// 将ip存储
	if err := m.setIp(); err != nil {
		xlog.Errorf(ctx, "%s set ip error: %v", fun, err)
	}

	// 初始化日志
	xlog.Infof(ctx, "%s initLog start", fun)
	_ = m.initLog(args)
	xlog.Infof(ctx, "%s initLog end", fun)

	// 初始化服务进程打点
	xlog.Infof(ctx, "%s init stat start", fun)
	stat.Init(m.servGroup, m.servName, "")
	xlog.Infof(ctx, "%s init stat end", fun)

	return nil
}

func (m *BaseServer) ApplyPreFns(ctx context.Context, args interface{}) error {
	fun := "BaseServer.ApplyPreFns -->"
	for _, fn := range m.preServiceInitFns {
		fnName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
		xlog.Infof(ctx, "%s apply pre fn: %s start", fun, fnName)
		if err := fn(ctx, m, args.(*cmdArgs)); err != nil {
			return err
		}
		xlog.Infof(ctx, "%s apply pre fn: %s end", fun, fnName)
	}
	return nil
}
func (m *BaseServer) AppendPreFns(fn ...initFn) {
	m.preServiceInitFns = append(m.preServiceInitFns, fn...)
}
func (m *BaseServer) ApplyPostFns(ctx context.Context, args interface{}) error {
	fun := "BaseServer.ApplyPostFns -->"
	for _, fn := range m.postServiceInitFns {
		fnName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
		xlog.Infof(ctx, "%s apply post fn: %s start", fun, fnName)
		if err := fn(ctx, m, args.(*cmdArgs)); err != nil {
			return err
		}
		xlog.Infof(ctx, "%s apply post fn: %s end", fun, fnName)
	}
	return nil
}
func (m *BaseServer) AppendPostFns(fn ...initFn) {
	m.postServiceInitFns = append(m.postServiceInitFns, fn...)
}

// Shutdown 关停服务
func (m *BaseServer) Shutdown(ctx context.Context) error {
	m.CrossDcRegistry.Shutdown()
	return nil
}
func (m *BaseServer) AppendShutdownCallback(context.Context, func()) {} // 添加关停回调函数
func (m *BaseServer) Offline(ctx context.Context) bool {
	// 底层已用原子操作保证安全
	return m.CrossDcRegistry.Offline()
}

// RunningLocal return true if server is local running
func (m *BaseServer) RunningLocal(ctx context.Context) bool { return m.localFlag.Local() }

func (m *BaseServer) ConfigCenter(ctx context.Context) zconfig.ConfigCenter { return m.configCenter }

// WithControlLaneInfo wrap context with service context info, such as lane
func (m *BaseServer) WithControlLaneInfo(ctx context.Context) context.Context {
	// use grpc ctx as default
	value := ctx.Value(xcontext.ContextKeyControl)
	if value == nil {
		// if service not run on grpc .try thrift
		control := m.createControlWithLaneInfo()
		return context.WithValue(ctx, xcontext.ContextKeyControl, control)
	}

	// different route group -> lane
	v := value.(xcontext.ContextControlRouter)
	if _, ok := v.GetControlRouteGroup(); !ok {
		_ = v.SetControlRouteGroup(m.envGroup)
	}

	return ctx
}

func (m *BaseServer) createControlWithLaneInfo() *thriftutil.Control {
	route := thriftutil.NewRoute()
	route.Group = m.envGroup
	control := thriftutil.NewControl()
	control.Route = route
	return control
}

func (m *BaseServer) ServConfig(cfg interface{}) error {
	fun := "BaseServer.ServConfig -->"
	ctx := context.Background()
	// 获取全局配置
	path := fmt.Sprintf("%s/%s", m.GetUseBaseLoc(), gentity.BASE_LOC_ETC_GLOBAL)
	sCfgGlobal, err := retcd.GetValue(m.GetEtcdClient(), path)
	if err != nil {
		xlog.Warnf(ctx, "%s serv config global value path: %s err: %v", fun, path, err)
	}
	xlog.Infof(ctx, "%s global cfg:%s path:%s", fun, sCfgGlobal, path)

	path = fmt.Sprintf("%s/%s/%s", m.GetUseBaseLoc(), gentity.BASE_LOC_ETC, m.GetServKey())
	scfg, err := retcd.GetValue(m.GetEtcdClient(), path)
	if err != nil {
		xlog.Warnf(context.Background(), "%s serv config value path: %s err: %v", fun, path, err)
	}

	tf := xconfig.NewTierConf()
	err = tf.Load(sCfgGlobal)
	if err != nil {
		return err
	}

	err = tf.Load(scfg)
	if err != nil {
		return err
	}

	err = tf.Unmarshal(cfg)
	if err != nil {
		return err
	}

	return nil
}

func (m *BaseServer) setLocal() {
	m.localFlag = gentity.RunningAtLocal
}

// NewBaseServer ...
func NewBaseServer(etcdAddresses []string, useBaseLoc string, args *cmdArgs) (*BaseServer, error) {
	fun := "NewBaseServer -->"
	ctx := context.Background()

	client, err := retcd.InitEtcdClient(fun, etcdAddresses)(ctx)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s/%s/%s", useBaseLoc, gentity.BASE_LOC_SKEY, args.servLoc)

	xlog.Infof(ctx, "%s retryGenSid start", fun)
	sid, err := retryGenSid(client, path, args.sessKey, 3)
	if err != nil {
		return nil, err
	}

	xlog.Infof(ctx, "%s retryGenSid end, path: %s, sid: %d, skey: %s, envGroup: %s", fun, path, sid, args.sessKey, args.group)

	// init global config center
	xlog.Infof(ctx, " %s init configcenter start", fun)
	configCenter, err := xconfig.NewConfigCenter(context.TODO(), xapollo.ConfigTypeApollo, args.servLoc, []string{
		gentity.ApplicationNamespace,
		gentity.RPCConfNamespace,
		gentity.RPCServerConfNamespace,
		consts.MysqlConfNamespace})
	if err != nil {
		return nil, err
	}
	xlog.Infof(ctx, " %s init configcenter end", fun)

	crossRegionIdList, err := parseCrossRegionIdList(args.crossRegionIdList)
	if err != nil {
		xlog.Errorf(ctx, "%s parse cross region id list error, arg: %v, err: %v", fun, args.crossRegionIdList, err)
		return nil, err
	}

	bs := &BaseServer{
		CrossDcRegistry: retcd.CrossDcRegistry{},
		envGroup:        args.group,
		sessKey:         args.sessKey,
		region:          args.region,

		locks:  make(map[string]*xsync.Semaphore),
		hearts: make(map[string]*distLockHeart),

		configCenter: configCenter,
	}

	bs.Build(
		retcd.WithCrossDcEtcdClient(client),
		retcd.WithCrossDcConfigEtcd(etcdAddresses, useBaseLoc),
		retcd.WithCrossDcRegInfos(make(map[string]string)),
		retcd.WithCrossDcAppendedPostShutdownFunc(func() { xlog.Info(context.TODO(), "app shutdown") }),
		retcd.WithCrossDcServId(sid),
		retcd.WithCrossDcServLoc(args.servLoc),

		retcd.WithCrossDcClients(make(map[string]etcd.KeysAPI, 2)),
		retcd.WithCrossDcRegionIds(crossRegionIdList),
	)

	svrInfo := strings.SplitN(args.servLoc, "/", 2) // eg: base/user
	if len(svrInfo) == 2 {
		bs.servGroup = svrInfo[0]
		bs.servName = svrInfo[1]
	} else {
		xlog.Warnf(ctx, "%s servLocation:%s do not match group/service format", fun, args.servLoc)
	}

	if args.startType == gentity.START_TYPE_LOCAL {
		bs.setLocal()
	}

	// init cross register clients
	xlog.Infof(ctx, " %s init CrossRegisterCenter start", fun)
	err = retcd.InitCrossRegisterCenter(bs)
	if err != nil {
		return nil, err
	}
	xlog.Infof(ctx, " %s init CrossRegisterCenter end", fun)

	return bs, nil

}
