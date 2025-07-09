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
 @Time    : 2025/7/8 -- 15:57
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: onegin onegin/gin.go
*/

package onegin

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Route struct {
	Status  int                `json:"status"`
	Method  string             `json:"method"`
	Path    string             `json:"path"`
	Handler func(*gin.Context) `json:"handler"`
}

type Engine interface {
	AddRoutes(*Route)
}

type EngineImpl struct {
	*gin.Engine
	PrefixPath string
}

func (i EngineImpl) AddRoutes(r *Route) {
	switch r.Method {
	case http.MethodGet:
		i.GET(i.PrefixPath+r.Path, r.Handler)
	case http.MethodPost:
		i.POST(i.PrefixPath+r.Path, r.Handler)
	case http.MethodPut:
		i.PUT(i.PrefixPath+r.Path, r.Handler)
	case http.MethodPatch:
		i.PATCH(i.PrefixPath+r.Path, r.Handler)
	case http.MethodDelete:
		i.DELETE(i.PrefixPath+r.Path, r.Handler)
	case http.MethodOptions:
		i.OPTIONS(i.PrefixPath+r.Path, r.Handler)
	}
}
