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
 @Time    : 2025/7/8 -- 15:56
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: onegin onegin/response.go
*/

package onegin

import (
	"github.com/gin-gonic/gin"
)

type GinError struct {
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
}

func Error(c *gin.Context, status int, err *GinError) {
	c.JSON(status, err)
}

func Success(c *gin.Context, status int, data interface{}) {
	res := map[string]interface{}{
		"code": 0,
		"msg":  "ok",
	}
	if data == nil {
		c.JSON(status, res)
	} else {
		res["data"] = data
		c.JSON(status, res)
	}
}

func JustSuccess(c *gin.Context, status int) {
	res := map[string]interface{}{
		"code": 0,
		"msg":  "ok",
	}
	c.JSON(status, res)
}
