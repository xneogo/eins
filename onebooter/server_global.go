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
 @Time    : 2024/11/12 -- 16:18
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: serve.go
*/

package onebooter

import (
	"context"
	"encoding/json"
	"github.com/qiguanzhu/infra/lcl/governor/gentity"
	"github.com/qiguanzhu/infra/nerv/xlog"
	"github.com/qiguanzhu/infra/seele/zconfig"
	"github.com/qiguanzhu/infra/seele/zserv"
	"os"
	"strings"
)

// Serve app call Serve to start server, initLogic is the init func in app, logic.InitLogic,
func Serve(etcdAddresses []string, baseLoc string, initLogic func(zserv.ServerSessionProxy[gentity.ServInfo]) error, processors map[string]zserv.ZProcessor) error {
	return server.Serve(etcdAddresses, baseLoc, initLogic, processors, true)
}

// MasterSlave Leader-Follower模式，通过etcd distribute lock进行选举
func MasterSlave(etcdAddresses []string, baseLoc string, initLogic func(zserv.ServerSessionProxy[gentity.ServInfo]) error, processors map[string]zserv.ZProcessor) error {
	return server.MasterSlave(etcdAddresses, baseLoc, initLogic, processors)
}

func GetServBase() zserv.ServerSessionProxy[gentity.ServInfo] {
	return server.bs
}

func GetServName() (servName string) {
	if server.bs != nil {
		servName = server.bs.Name(context.Background())
	}
	return
}

// GetGroupAndService return group and service name of this service
func GetGroupAndService() (group, service string) {
	serviceKey := GetServName()
	serviceKeyArray := strings.Split(serviceKey, "/")
	if len(serviceKeyArray) == 2 {
		group = serviceKeyArray[0]
		service = serviceKeyArray[1]
	}
	return
}

func GetServId() (servId int) {
	if server.bs != nil {
		servId = server.bs.Id(context.Background())
	}
	return
}

// GetConfigCenter get serv conf center
func GetConfigCenter() zconfig.ConfigCenter {
	if server.bs != nil {
		return server.bs.ConfigCenter(context.Background())
	}
	return nil
}

// GetProcessorAddress get processor ip+port by processorName
func GetProcessorAddress(processorName string) (addr string) {
	ctx := context.Background()
	if server == nil {
		return
	}
	regInfos := server.bs.RegInfos()
	for _, val := range regInfos {
		data := new(gentity.RegData)
		err := json.Unmarshal([]byte(val), data)
		if err != nil {
			xlog.Warnf(ctx, "GetProcessorAddress unmarshal, val = %s, err = %s", val, err.Error())
			continue
		}
		if servInfo, ok := data.ServMap[processorName]; ok {
			addr = servInfo.Addr
			return
		}
	}
	return
}

// Test 方便开发人员在本地启动服务、测试，实例信息不会注册到etcd
func Test(etcdAddresses []string, baseLoc, servLoc string, initLogic func(zserv.ServerSessionProxy[gentity.ServInfo]) error) error {
	args := &cmdArgs{
		logMaxSize:    0,
		logMaxBackups: 0,
		servLoc:       servLoc,
		sessKey:       "test",
		logDir:        "console",
		disable:       true,
	}
	return server.Boot(etcdAddresses, baseLoc, args, initLogic, nil, true)
}

func getRegionFromEnvOrDefault() string {
	region := os.Getenv("REGION")
	if region == "" {
		region = "cn"
	}

	return strings.ToLower(region)
}
