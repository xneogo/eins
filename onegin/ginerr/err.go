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
 @Time    : 2025/7/9 -- 17:05
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: ginerr onegin/ginerr/err.go
*/

package ginerr

type XError struct {
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
}

func New(code int, msg string) *XError {
	return &XError{
		Code: code,
		Msg:  msg,
	}
}

func (x *XError) Error() string {
	return x.Msg
}
