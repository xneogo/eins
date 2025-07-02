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
 @Time    : 2025/7/1 -- 17:03
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: xsql xsql/consts.go
*/

package sqlutils

const (
	// MysqlConfNamespace mysql apollo conf namespace
	MysqlConfNamespace = "mysql"
	// MaxIdleConnsKey ...
	MaxIdleConnsKey = "maxIdleConns"
	// MaxOpenConnsKey ...
	MaxOpenConnsKey = "maxOpenConns"
	// MaxLifeTimeSecKey ...
	MaxLifeTimeSecKey = "maxLifeTimeSec"
	// TimeoutMsecKey ...
	TimeoutMsecKey = "timeoutMsec"
	// ReadTimeoutMsecKey ...
	ReadTimeoutMsecKey = "readTimeoutMsec"
	// WriteTimeoutMsecKey ...
	WriteTimeoutMsecKey = "writeTimeoutMsec"
	UserNameKey         = "username"
	PasswordKey         = "password"
	// KeySep
	KeySep = "."

	DefaultMaxIdleConns       = 64
	DefaultMaxOpenConns       = 128
	DefaultReadTimeoutSecond  = 10
	DefaultWriteTimeoutSecond = 10
	DefaultMaxLifeTimeSecond  = 3600 * 6
	DefaultTimeoutSecond      = 3
)

const (
	// DefaultTagName is the default struct tag name
	DefaultTagName = "bdb"
	CTimeFormat    = "2006-01-02 15:04:05"
)

const (
	DefaultDriver = "mysql"
	DefaultPort   = 3306
	CDSNFormat    = "%s%s=%s&"
)

const (
	WeirProxyHost = "weirproxy.service.svc.cluster.local"
	WeirProxyPort = 9021
	DefaultDbType = "weir_proxy"
)

const (
	WeirProxyHostEnv = "WEIR_PROXY_HOST"
	WeirProxyPortEnv = "WEIR_PROXY_PORT"
)
