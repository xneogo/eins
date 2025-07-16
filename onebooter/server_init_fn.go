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
 @Time    : 2024/11/14 -- 12:06
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: server_init_fn.go
*/

package onebooter

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/qiguanzhu/infra/lcl/governor"
	"github.com/qiguanzhu/infra/lcl/governor/gentity"
	"github.com/qiguanzhu/infra/lcl/governor/gregistry/retcd"
	"github.com/qiguanzhu/infra/lcl/procimp"
	"github.com/qiguanzhu/infra/nerv/magi/xtime"
	"github.com/qiguanzhu/infra/nerv/xlog"
	"github.com/qiguanzhu/infra/nerv/xtrace"
	"github.com/qiguanzhu/infra/seele/zserv"
	"os"
	"os/signal"
	"syscall"
)

type initFn func(ctx context.Context, bs *BaseServer, args *cmdArgs) error

// ------------------------------------- Pre fn start

func (m *Server) initPowerSwitch() initFn {
	return func(ctx context.Context, bs *BaseServer, args *cmdArgs) error {
		fun := "Server.initPowerSwitch -->"
		st := xtime.NewTimeStat()
		xlog.Infof(ctx, "%s start ", fun)
		defer func() {
			xlog.Infof(ctx, "%s end, durationMs: %d", fun, st.Millisecond())
		}()

		m.powerSwitch = NewDriverBuilder(bs.ConfigCenter(ctx))
		return nil
	}
}

func (m *Server) handleModel() initFn {
	return func(ctx context.Context, locker *BaseServer, args *cmdArgs) error {
		fun := "Server.handleModel -->"

		if args.model == gentity.MODEL_MASTERSLAVE {
			lockKey := fmt.Sprintf("%s-master-slave", args.servLoc)
			if err := locker.LockGlobal(ctx, lockKey); err != nil {
				xlog.Errorf(ctx, "%s LockGlobal key: %s, err: %v", fun, lockKey, err)
				return err
			}

			xlog.Infof(ctx, "%s LockGlobal succ, key: %s", fun, lockKey)
		}

		return nil
	}
}

func (m *Server) initCircuitBreaker() initFn {
	return func(ctx context.Context, bs *BaseServer, args *cmdArgs) error {
		fun := "Server.initCircuitBreaker -->"
		// circuit breaker
		err := retcd.InitBreaker(bs.Group(ctx), bs.Name(ctx))
		if err != nil {
			xlog.Errorf(context.Background(), "%s: circuit_breaker.Init() failed, error: %+v", fun, err)
			return err
		}

		// rate limiter
		etcdInterfaceRateLimitRegistry, err := retcd.NewInterfaceRateLimitRegistry(bs.Group(ctx), bs.Name(ctx), retcd.ETCDS_CLUSTER_0)
		if err != nil {
			xlog.Errorf(context.Background(), "%s: registry.NewEtcdInterfaceRateLimitRegistry() failed, error: %+v", fun, err)
			return err
		}
		governor.RateLimitRegistry = etcdInterfaceRateLimitRegistry
		go func() {
			etcdInterfaceRateLimitRegistry.Watch()
		}()
		return nil
	}
}

// ------------------------------------- Post fn start

func (m *Server) initBackdoor() initFn {
	return func(ctx context.Context, bs *BaseServer, args *cmdArgs) error {
		fun := "Server.initBackdoor -->"

		backdoor := &procimp.BackDoorHttp{}
		err := backdoor.Init()
		if err != nil {
			xlog.Errorf(ctx, "%s init backdoor err: %v", fun, err)
			return err
		}

		bInfos, err := m.loadDriver(map[string]zserv.ZProcessor{"_PROC_BACKDOOR": backdoor}, bs)
		if err == nil {
			err = bs.Register(ctx, bInfos, gentity.LocRegBackdoor.String())
			if err != nil {
				xlog.Errorf(ctx, "%s register backdoor err: %v", fun, err)
			}

		} else {
			xlog.Warnf(ctx, "%s load backdoor driver err: %v", fun, err)
		}

		return err
	}
}

func (m *Server) initTracer() initFn {
	return func(ctx context.Context, bs *BaseServer, args *cmdArgs) error {
		fun := "Server.initTracer -->"

		err := xtrace.InitDefaultTracer(args.servLoc)
		if err != nil {
			xlog.Errorf(ctx, "%s init tracer err: %v", fun, err)
		}

		err = xtrace.InitTraceSpanFilter()
		if err != nil {
			xlog.Errorf(ctx, "%s init trace span filter fail: %s", fun, err.Error())
		}

		return err
	}
}

func (m *Server) initProcessor(procs map[string]zserv.ZProcessor) initFn {
	return func(ctx context.Context, bs *BaseServer, args *cmdArgs) error {
		fun := "Server.initProcessor -->"

		for n, p := range procs {
			if len(n) == 0 {
				xlog.Errorf(ctx, "%s processor name empty", fun)
				return fmt.Errorf("processor name empty")
			}

			if n[0] == '_' {
				xlog.Errorf(ctx, "%s processor name can not prefix '_'", fun)
				return fmt.Errorf("processor name can not prefix '_'")
			}

			if p == nil {
				xlog.Errorf(ctx, "%s processor:%s is nil", fun, n)
				return fmt.Errorf("processor:%s is nil", n)
			} else {
				err := p.Init()
				if err != nil {
					xlog.Errorf(ctx, "%s processor: %s init err: %v", fun, n, err)
					return fmt.Errorf("processor:%s init err:%s", n, err)
				}
			}
		}

		infos, err := m.loadDriver(procs, bs)
		if err != nil {
			xlog.Errorf(ctx, "%s load driver err: %v", fun, err)
			return err
		}

		// 本地启动不注册至etcd
		if bs.RunningLocal(ctx) {
			return nil
		}

		err = bs.Register(ctx, infos, gentity.LocRegService.String())
		if err != nil {
			xlog.Errorf(ctx, "%s register service err: %v", fun, err)
			return err
		}

		// 注册跨机房服务
		err = bs.RegisterCrossDCService(ctx, infos, gentity.LocRegService.String())
		if err != nil {
			xlog.Errorf(ctx, "%s register cross dc failed, err: %v", fun, err)
			return err
		}

		return nil
	}

}

func (m *Server) setGroupAndDisable() initFn {
	return func(ctx context.Context, bs *BaseServer, args *cmdArgs) error {
		fun := "Server.SetGroupAndDisable -->"

		path := fmt.Sprintf("%s/%s/%s/%d/%s", bs.GetUseBaseLoc(), gentity.LocDIST, bs.GetServKey(), bs.GetServId(), gentity.LocRegManual)
		value, err := bs.getValueFromEtcd(path)
		if err != nil {
			xlog.Warnf(ctx, "%s getValueFromEtcd err, path:%s, err:%v", fun, path, err)
		}

		manual := &gentity.ManualData{}
		err = json.Unmarshal([]byte(value), manual)
		if len(value) > 0 && err != nil {
			xlog.Errorf(ctx, "%s unmarshal err, value:%s, err:%v", fun, value, err)
			return err
		}

		if manual.Ctrl == nil {
			manual.Ctrl = &gentity.ServCtrl{}
		}

		isFind := false
		for _, g := range manual.Ctrl.Groups {
			if g == args.group {
				isFind = true
				break
			}
		}

		if isFind == false {
			manual.Ctrl.Groups = append(manual.Ctrl.Groups, args.group)
		}
		if manual.Ctrl.Weight == 0 {
			manual.Ctrl.Weight = 100
		}
		manual.Ctrl.Disable = args.disable

		newValue, err := json.Marshal(manual)
		if err != nil {
			xlog.Errorf(ctx, "%s marshal err, manual:%v, err:%v", fun, manual, err)
			return err
		}

		xlog.Infof(ctx, "%s path:%s old value:%s new value:%s", fun, path, value, newValue)
		err = bs.setValueToEtcd(path, string(newValue), nil)
		if err != nil {
			xlog.Errorf(ctx, "%s setValueToEtcd err, path:%s value:%s", fun, path, newValue)
		}

		return err
	}
}

func (m *Server) initMetric() initFn {
	return func(ctx context.Context, bs *BaseServer, args *cmdArgs) error {
		fun := "Server.initMetric -->"

		metrics := procimp.NewMetricProcessor()
		err := metrics.Init()
		if err != nil {
			xlog.Warnf(ctx, "%s init metrics err: %v", fun, err)
		}

		metricInfo, err := m.loadDriver(map[string]zserv.ZProcessor{"_PROC_METRICS": metrics}, bs)
		if err == nil {
			err = bs.Register(ctx, metricInfo, gentity.LocRegMetrics.String())
			if err != nil {
				xlog.Warnf(ctx, "%s register backdoor err: %v", fun, err)
			}

		} else {
			xlog.Warnf(ctx, "%s load metrics driver err: %v", fun, err)
		}
		return err
	}
}

func (m *Server) awaitSignal(await bool) initFn {
	return func(_ context.Context, bs *BaseServer, args *cmdArgs) error {
		if !await {
			return nil
		}
		c := make(chan os.Signal, 1)
		ctx := context.Background()
		signals := []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGPIPE}
		signal.Reset(signals...)
		signal.Notify(c, signals...)

		for {
			select {
			case s := <-c:
				xlog.Infof(ctx, "receive a signal:%s", s.String())

				if s.String() == syscall.SIGTERM.String() {
					xlog.Infof(ctx, "receive a signal: %s, stop server", s.String())
					_ = bs.Shutdown(ctx)
					<-(chan int)(nil)
				}
			}
		}
	}
}
