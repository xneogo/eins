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
 @Time    : 2024/11/12 -- 18:20
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: base_server_inner.go
*/

package onebooter

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/qiguanzhu/infra/lcl/governor/gentity"
	"github.com/qiguanzhu/infra/nerv/magi/xnet"
	"github.com/qiguanzhu/infra/nerv/xlog"
	"github.com/qiguanzhu/infra/seele/zserv"
	etcd "go.etcd.io/etcd/client/v2"
	"sort"
	"strconv"
	"strings"
)

func (m *BaseServer) addServiceInfo(path string, service zserv.ZProcessor, servInfo *gentity.ServInfo) {
	m.muService.Lock()
	defer m.muService.Unlock()

	m.services[path] = service
	m.servInfos[path] = servInfo
}

func (m *BaseServer) setIp() error {
	addr, err := xnet.GetListenAddr("")
	if err != nil {
		return err
	}
	fields := strings.Split(addr, ":")
	if len(fields) < 1 {
		return fmt.Errorf("get listen addr error")
	}
	m.servIp = fields[0]
	return nil
}

func (m *BaseServer) getValueFromEtcd(path string) (value string, err error) {
	fun := "BaseServer.getValueFromEtcd -->"
	ctx := context.Background()

	r, err := m.GetEtcdClient().Get(context.Background(), path, &etcd.GetOptions{Recursive: false, Sort: false})
	if err != nil {
		xlog.Warnf(ctx, "%s path:%s err:%v", fun, path, err)
		return "", err
	}
	if r != nil && r.Node != nil {
		return r.Node.Value, nil
	}

	return "", nil
}

func (m *BaseServer) setValueToEtcd(path, value string, opts *etcd.SetOptions) error {
	fun := "BaseServer.setValueToEtcd -->"

	_, err := m.GetEtcdClient().Set(context.Background(), path, value, opts)
	if err != nil {
		xlog.Errorf(context.Background(), "%s path:%s value:%s opts:%v", fun, path, value, opts)
	}

	return err
}

func (m *BaseServer) isPreEnvGroup() bool {
	if m.envGroup == gentity.ENV_GROUP_PRE {
		return true
	}

	return false
}

func (m *BaseServer) initLog(args *cmdArgs) error {
	fun := "Server.initLog -->"
	ctx := context.Background()

	logDir := args.logDir
	var logConfig struct {
		Log struct {
			Level string
			Dir   string
		}
	}
	logConfig.Log.Level = "INFO"

	err := m.ServConfig(&logConfig)
	if err != nil {
		xlog.Errorf(ctx, "%s serv config err: %v", fun, err)
		return err
	}

	var logdir string
	if len(logConfig.Log.Dir) > 0 {
		logdir = fmt.Sprintf("%s/%s", logConfig.Log.Dir, m.FullName(ctx))
	}

	if len(logDir) > 0 {
		logdir = fmt.Sprintf("%s/%s", logDir, m.FullName(ctx))
	}

	if logDir == "console" {
		logdir = ""
	}

	xlog.Infof(ctx, "%s init log dir:%s name:%s level:%s", fun, logdir, args.servLoc, logConfig.Log.Level)

	// 最终根据Apollo中配置的log level决定日志级别， TODO 后续将从etcd获取日志配置的逻辑去掉，统一在Apollo内配置
	logLevel, ok := m.ConfigCenter(ctx).GetString(context.TODO(), "log_level")
	if ok {
		logConfig.Log.Level = logLevel
	}
	extraHeaders := map[string]interface{}{
		"region": m.Region(ctx),
		"lane":   m.Lane(ctx),
		"ip":     m.Ip(ctx),
	}
	_ = xlog.InitAppLogHeaders(logdir, "serv.log", convertLevel(logConfig.Log.Level), extraHeaders)
	_ = xlog.InitStatLog(logdir, "stat.log", 0, false)
	xlog.SetStatLogService(args.servLoc)
	return nil
}

func genSid(client etcd.KeysAPI, path, skey string) (int, error) {
	fun := "genSid -->"
	ctx := context.Background()
	r, err := client.Get(context.Background(), path, &etcd.GetOptions{Recursive: true, Sort: false})
	if err != nil {
		return -1, err
	}

	js, _ := json.Marshal(r)

	xlog.Infof(ctx, "%s", js)

	if r.Node == nil || !r.Node.Dir {
		return -1, fmt.Errorf("node error location:%s", path)
	}

	xlog.Infof(ctx, "%s serv:%s len:%d", fun, r.Node.Key, r.Node.Nodes.Len())

	// 获取已有的 servId，按从小到大排列
	ids := make([]int, 0)
	for _, n := range r.Node.Nodes {
		sid := n.Key[len(r.Node.Key)+1:]
		id, err := strconv.Atoi(sid)
		if err != nil || id < 0 {
			xlog.Errorf(ctx, "%s sid error key:%s", fun, n.Key)
		} else {
			ids = append(ids, id)
			if n.Value == skey {
				// 如果已经存在的sid使用的skey和设置一致，则使用之前的sid
				return id, nil
			}
		}
	}

	sort.Ints(ids)
	sid := 0
	for _, id := range ids {
		// 取不重复的最小的id
		if sid == id {
			sid++
		} else {
			break
		}
	}

	nserv := fmt.Sprintf("%s/%d", r.Node.Key, sid)
	r, err = client.Create(context.Background(), nserv, skey)
	if err != nil {
		return -1, err
	}

	jr, _ := json.Marshal(r)
	xlog.Infof(ctx, "%s newserv:%s resp:%s", fun, nserv, jr)

	return sid, nil

}

func retryGenSid(client etcd.KeysAPI, path, sKey string, try int) (int, error) {
	fun := "retryGenSid -->"
	ctx := context.Background()
	for i := 0; i < try; i++ {
		// 重试3次
		sid, err := genSid(client, path, sKey)
		if err != nil {
			xlog.Errorf(ctx, "%s gensid try: %d path: %s err: %v", fun, i, path, err)
		} else {
			return sid, nil
		}
	}

	return -1, fmt.Errorf("gensid error try:%d", try)
}

func convertLevel(level string) xlog.Level {
	level = strings.ToLower(level)
	switch level {
	case "debug":
		return xlog.DebugLevel
	case "info":
		return xlog.InfoLevel
	case "warn":
		return xlog.WarnLevel
	case "error":
		return xlog.ErrorLevel
	case "fatal":
		return xlog.FatalLevel
	case "panic":
		return xlog.PanicLevel
	default:
		return xlog.InfoLevel
	}
}

func parseCrossRegionIdList(idListStr string) ([]int, error) {
	if idListStr == "" {
		return nil, nil
	}

	idStrList := strings.Split(idListStr, ",")
	if len(idStrList) == 0 {
		return nil, errors.New("invalid id list string arg")
	}

	var ret []int
	for _, idStr := range idStrList {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return nil, err
		}
		ret = append(ret, id)
	}

	return ret, nil
}
