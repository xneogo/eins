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
 @Time    : 2024/11/4 -- 17:03
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: request.go
*/

package onegovernor

import (
	"context"
	"io"
	"net/http"
)

var (
	requestIdCtxKey     = &struct{ Name string }{Name: "requestid"}
	requestMethodCtxKey = &struct{ Name string }{Name: "method"}
	requestHostCtxKey   = &struct{ Name string }{Name: "host"}
	requestPathCtxKey   = &struct{ Name string }{Name: "path"}
	requestQueryCtxKey  = &struct{ Name string }{Name: "query"}
	requestQuerysCtxKey = &struct{ Name string }{Name: "querys"}
	requestBodyCtxKey   = &struct{ Name string }{Name: "body"}
	requestIPCtxKey     = &struct{ Name string }{Name: "ip"}
	requestUACtxKey     = &struct{ Name string }{Name: "ua"}
	requestReferCtxKey  = &struct{ Name string }{Name: "refer"}
)

func Request(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		querys := make(map[string]string, len(r.URL.Query()))
		for k, v := range r.URL.Query() {
			if len(v) > 0 {
				querys[k] = v[0]
			}
		}

		body, _ := io.ReadAll(r.Body)

		ctx := r.Context()
		ctx = context.WithValue(ctx, requestMethodCtxKey, r.Method)
		ctx = context.WithValue(ctx, requestHostCtxKey, r.Host)
		ctx = context.WithValue(ctx, requestPathCtxKey, r.URL.Path)
		ctx = context.WithValue(ctx, requestQueryCtxKey, r.URL.RawQuery)
		ctx = context.WithValue(ctx, requestQuerysCtxKey, querys)
		ctx = context.WithValue(ctx, requestBodyCtxKey, body)
		ctx = context.WithValue(ctx, requestIPCtxKey, r.RemoteAddr)
		ctx = context.WithValue(ctx, requestUACtxKey, r.UserAgent())
		ctx = context.WithValue(ctx, requestReferCtxKey, r.Referer())
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func Value(ctx context.Context, key interface{}) interface{} {
	return ctx.Value(key)
}

func StringValue(ctx context.Context, key interface{}) string {
	if value, ok := ctx.Value(key).(string); ok {
		return value
	}
	return ""
}

func RequestID(ctx context.Context) string {
	return StringValue(ctx, requestIdCtxKey)
}

func RequestMethod(ctx context.Context) string {
	return StringValue(ctx, requestMethodCtxKey)
}

func RequestHost(ctx context.Context) string {
	return StringValue(ctx, requestHostCtxKey)
}

func RequestPath(ctx context.Context) string {
	return StringValue(ctx, requestPathCtxKey)
}

func RequestQuery(ctx context.Context) string {
	return StringValue(ctx, requestQueryCtxKey)
}

func RequestQuerys(ctx context.Context) map[string]string {
	if value, ok := ctx.Value(requestQuerysCtxKey).(map[string]string); ok {
		return value
	}
	return nil

}

func RequestBody(ctx context.Context) []byte {
	if value, ok := ctx.Value(requestBodyCtxKey).([]byte); ok {
		return value
	}
	return nil
}

func RequestIP(ctx context.Context) string {
	return StringValue(ctx, requestIPCtxKey)
}

func RequestUA(ctx context.Context) string {
	return StringValue(ctx, requestUACtxKey)
}

func RequestRefer(ctx context.Context) string {
	return StringValue(ctx, requestReferCtxKey)
}
