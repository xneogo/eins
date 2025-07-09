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
 @Time    : 2025/7/9 -- 14:09
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: onenet onenet/global.go
*/

package onenet

import (
	"crypto/tls"
	"github.com/valyala/fasthttp"
	"net/http"
	"time"
)

var (
	stdCli = &standard{
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 1024,
				MaxConnsPerHost:     16384,
				IdleConnTimeout:     10 * time.Minute,
			},
			Timeout: defaultTimeout,
		},
	}

	stdInsecureCli = &standard{
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 1024,
				MaxConnsPerHost:     16384,
				IdleConnTimeout:     10 * time.Minute,
				TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: defaultTimeout,
		},
	}

	fastCli = &fast{
		client: &fasthttp.Client{
			MaxConnsPerHost:     16384,
			MaxIdleConnDuration: 10 * time.Minute,
			ReadTimeout:         defaultTimeout,
			WriteTimeout:        defaultTimeout,
		},
	}

	fastInsecureCli = &fast{
		client: &fasthttp.Client{
			MaxConnsPerHost:     16384,
			MaxIdleConnDuration: 10 * time.Minute,
			ReadTimeout:         defaultTimeout,
			WriteTimeout:        defaultTimeout,
			TLSConfig:           &tls.Config{InsecureSkipVerify: true},
		},
	}
)

func GetStd() Client {
	return stdInsecureCli
}

func GetStdInsecure() Client {
	return stdInsecureCli
}

func GetFast() Client {
	return fastCli
}

func GetFastInsecure() Client {
	return fastInsecureCli
}
