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
 @Time    : 2025/7/9 -- 17:02
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: oneginreq onegin/ginreq/args.go
*/

package ginreq

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cast"
	"github.com/xneogo/eins/onelog"
)

type KeysPortal interface {
	Get(string) string
}

type XKeys struct {
	k KeysPortal
}

func NewXKeys(k KeysPortal) *XKeys {
	return &XKeys{k}
}
func (x *XKeys) Int(key string) int {
	return cast.ToInt(x.k.Get(key))
}
func (x *XKeys) String(key string) string {
	return x.k.Get(key)
}
func (x *XKeys) Bool(key string) bool {
	return cast.ToBool(x.k.Get(key))
}
func (x *XKeys) Int32(key string) int32 {
	return cast.ToInt32(x.k.Get(key))
}
func (x *XKeys) Int64(key string) int64 {
	return cast.ToInt64(x.k.Get(key))
}

type Xargs struct {
	r *http.Request
	q url.Values
}

func (x *Xargs) Get(key string) string {
	fun := "reqQuery.Get"
	ctx := context.Background()
	if x.r != nil {
		ctx = x.r.Context()
	}
	if x.q == nil {
		if x.r.URL != nil {
			var err error
			x.q, err = url.ParseQuery(x.r.URL.RawQuery)
			if err != nil {
				onelog.Ctx(ctx).Warn().Err(err).Str("url", fmt.Sprintf("parse query q:%s err:%s", x.r.URL.RawQuery, err)).Msg(fun)
			}
		}

		if x.q == nil {
			x.q = make(url.Values)
		}
		onelog.Ctx(ctx).Debug().Str("f", fmt.Sprintf("parse query q:%s err:%s", x.r.URL.RawQuery, x.q)).Msg(fun)
	}

	return x.q.Get(key)
}

type XHeaders struct {
	r *http.Request
}

func (x *XHeaders) Get(key string) string {
	return x.r.Header.Get(key)
}
