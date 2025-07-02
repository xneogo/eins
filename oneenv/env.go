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
 @Time    : 2025/7/1 -- 16:47
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: env oneenv/env.go
*/

package oneenv

import (
	"os"

	"github.com/kelseyhightower/envconfig"
)

const (
	EnvDev  = "dev"  // 开发环境
	EnvTest = "test" // 测试环境
	EnvPre  = "pre"  // 预发环境
	EnvProd = "prod" // 生产环境
)

type Enver interface {
	GetEnv() string
	GetApp() string
	IsProd() bool
}

type Environment struct {
	Env string `envconfig:"ENV"`
	App string `envconfig:"APP"`
}

func (c *Environment) GetEnv() string {
	return c.Env
}

func (c *Environment) GetApp() string {
	return c.App
}

func (c *Environment) IsProd() bool {
	return c.Env == EnvProd
}

func (c *Environment) IsPre() bool {
	return c.Env == EnvPre
}

func (c *Environment) IsTest() bool {
	return c.Env == EnvTest
}

func (c *Environment) IsDev() bool {
	return c.Env == EnvDev
}

func Init(env Environment) error {
	return envconfig.Process("", env)
}

func GetEnvWithDefault(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}
