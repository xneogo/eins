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
 @Time    : 2024/11/6 -- 10:18
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: server.go
*/

package onebooter

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/qiguanzhu/infra/lcl/governor/gentity"
	"github.com/qiguanzhu/infra/nerv/xlog"
	"github.com/qiguanzhu/infra/seele/zserv"
)

var server = NewServer()

// Server ...
type Server struct {
	bs          zserv.ServerSessionProxy[gentity.ServInfo]
	powerSwitch *DriverBuilder
}

// NewServer create new server
func NewServer() *Server {
	return &Server{}
}

type cmdArgs struct {
	logMaxSize    int
	logMaxBackups int
	servLoc       string
	logDir        string
	sessKey       string
	sidOffset     int
	group         string
	disable       bool
	model         gentity.ServerModel
	startType     string // 启动方式：local - 不注册至etcd

	crossRegionIdList string
	region            string
	backdoorPort      string
}

func (m *Server) parseFlag() (*cmdArgs, error) {
	var serv, logDir, skey, group, startType string
	var logMaxSize, logMaxBackups, sidOffset int

	flag.IntVar(&logMaxSize, "logmaxsize", 0, "logMaxSize is the maximum size in megabytes of the log file")
	flag.IntVar(&logMaxBackups, "logmaxbackups", 0, "logmaxbackups is the maximum number of old log files to retain")
	flag.StringVar(&serv, "serv", "", "servic name")
	flag.StringVar(&logDir, "logdir", "", "serice log dir")
	flag.StringVar(&skey, "skey", "", "service session key")
	flag.IntVar(&sidOffset, "sidoffset", 0, "service id offset for different data center")
	flag.StringVar(&group, "group", "", "service group")
	// 启动方式：local - 不注册至etcd
	flag.StringVar(&startType, "stype", "", "start up type, local is not register to etcd")
	flag.Parse()

	if len(serv) == 0 {
		return nil, fmt.Errorf("serv args needed! ")
	}

	if len(skey) == 0 {
		return nil, fmt.Errorf("skey args needed! ")
	}

	return &cmdArgs{
		logMaxSize:    logMaxSize,
		logMaxBackups: logMaxBackups,
		servLoc:       serv,
		logDir:        logDir,
		sessKey:       skey,
		sidOffset:     sidOffset,
		group:         group,
		startType:     startType,
	}, nil

}

func (m *Server) loadDriver(procs map[string]zserv.ZProcessor, bs *BaseServer) (map[string]*gentity.ServInfo, error) {
	fun := "Server.loadDriver -->"
	ctx := context.Background()

	infos := make(map[string]*gentity.ServInfo)

	for n, p := range procs {
		servInfo, err := m.powerSwitch.PowerProcessorDriver(ctx, n, p, bs)
		if errors.Is(err, ErrNilDriver) {
			xlog.Infof(ctx, "%s power: %s found no driver, skip", fun, n)
			continue
		}
		if err != nil {
			xlog.Errorf(ctx, "%s load error when power: %s, err: %v", fun, n, err)
			return nil, err
		}
		infos[n] = servInfo
		bs.addServiceInfo(n, p, servInfo)
		xlog.Infof(ctx, "%s load ok, powered: %s, serv addr: %s", fun, n, servInfo.Addr)
	}

	return infos, nil
}

// Serve handle request and return response
func (m *Server) Serve(etcdAddresses []string, useBaseLoc string, initFn func(zserv.ServerSessionProxy[gentity.ServInfo]) error, procs map[string]zserv.ZProcessor, awaitSignal bool) error {
	fun := "Server.Serve -->"

	args, err := m.parseFlag()
	if err != nil {
		xlog.Panicf(context.Background(), "%s parse arg err: %v", fun, err)
		return err
	}

	return m.Boot(etcdAddresses, useBaseLoc, args, initFn, procs, awaitSignal)
}

func (m *Server) Boot(etcdAddresses []string, useBaseLoc string, args *cmdArgs, initFn func(zserv.ServerSessionProxy[gentity.ServInfo]) error, procs map[string]zserv.ZProcessor, awaitSignal bool) error {
	fun := "Server.Init -->"
	ctx := context.Background()

	servLoc := args.servLoc
	sessKey := args.sessKey

	nbs, err := NewBaseServer(etcdAddresses, useBaseLoc, args)
	if err != nil {
		xlog.Panicf(ctx, "%s init servbase loc: %s key: %s err: %v", fun, servLoc, sessKey, err)
		return err
	}
	// todo start up 目前拆分可能不合理
	err = nbs.Startup(ctx, args)
	if err != nil {
		return err
	}

	defer xlog.AppLogSync()
	defer xlog.StatLogSync()

	nbs.AppendPreFns(
		// 初始化 power up switcher
		m.initPowerSwitch(),
		m.handleModel(),
		m.initCircuitBreaker(),
	)
	nbs.AppendPostFns(
		// NOTE: initBackdoor会启动http服务，但由于health check的http请求不需要追踪，且它是判断服务启动与否的关键，所以initTracer可以放在它之后进行
		m.initBackdoor(),
		m.initTracer(),
		// NOTE: processor 在初始化 trace middleware 前需要保证 xtrace.GlobalTracer() 初始化完毕
		m.initProcessor(procs),
		m.setGroupAndDisable(),
		m.initMetric(),
		m.awaitSignal(awaitSignal),
	)

	m.bs = nbs

	// pre init
	err = m.bs.ApplyPreFns(ctx, args)
	if err != nil {
		xlog.Panicf(ctx, "%s ApplyPreInitFns err: %v", fun, err)
		return err
	}

	// App层初始化
	xlog.Infof(ctx, "%s call initFn start", fun)
	err = initFn(nbs)
	if err != nil {
		xlog.Panicf(ctx, "%s callInitFunc err: %v", fun, err)
		return err
	}
	xlog.Infof(ctx, "%s call initFn end", fun)

	// post init
	err = m.bs.ApplyPostFns(ctx, args)
	if err != nil {
		xlog.Panicf(ctx, "%s ApplyPostFns err: %v", fun, err)
		return err
	}

	xlog.Infof(ctx, "server start success, grpc: [%s]", GetProcessorAddress(gentity.PROCESSOR_GRPC_PROPERTY_NAME))

	return nil
}

func (m *Server) MasterSlave(etcdAddresses []string, baseLoc string, initLogic func(zserv.ServerSessionProxy[gentity.ServInfo]) error, processors map[string]zserv.ZProcessor) error {
	fun := "Server.MasterSlave -->"
	ctx := context.Background()

	args, err := m.parseFlag()
	if err != nil {
		xlog.Panicf(ctx, "%s parse arg err: %v", fun, err)
		return err
	}
	args.model = gentity.MODEL_MASTERSLAVE

	return m.Boot(etcdAddresses, baseLoc, args, initLogic, processors, true)
}
