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
 @Time    : 2025/7/9 -- 13:51
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: onenet onenet/http.go
*/

package onenet

import (
	"context"
	"github.com/rs/zerolog"
	"maps"
	"net/url"
	"time"
)

const (
	FormContentType  = "application/x-www-form-urlencoded"
	JsonContentType  = "application/json"
	ProtoContentType = "application/x-protobuf"
	ZipContentType   = "application/zip"

	ContentTypeHeader     = "Content-Type"
	AcceptEncodingHeader  = "Accept-Encoding"
	ContentEncodingHeader = "Content-Encoding"

	defaultTimeout           = time.Hour
	defaultLogBodySize int64 = 1 * 1024
	defaultResBodySize int64 = 64 * 1024 * 1024
)

type OneRequest struct {
	Method      string
	URL         string
	Query       url.Values
	Header      map[string]string
	ContentType string
	Body        any

	Timeout        time.Duration
	LogLevel       *zerolog.Level
	MaxLogBodySize int64
	MaxResBodySize int64
	PromKey        string
}

type ReqSetting func(*OneRequest)

func WithGzip(enable bool) ReqSetting {
	return func(r *OneRequest) {
		if enable {
			r.Header[AcceptEncodingHeader] = "gzip"
		} else {
			delete(r.Header, AcceptEncodingHeader)
		}
	}
}

func WithQuery(query url.Values) ReqSetting {
	return func(r *OneRequest) {
		r.Query = query
	}
}

func WithHeader(header map[string]string) ReqSetting {
	return func(r *OneRequest) {
		maps.Copy(r.Header, header)
	}
}

func WithContentType(contentType string) ReqSetting {
	return func(r *OneRequest) {
		r.ContentType = contentType
	}
}

func WithBody(body any) ReqSetting {
	return func(r *OneRequest) {
		r.Body = body
	}
}

func WithTimeout(timeout time.Duration) ReqSetting {
	return func(r *OneRequest) {
		r.Timeout = min(r.Timeout, timeout)
	}
}

func WithLogLevel(level zerolog.Level) ReqSetting {
	return func(r *OneRequest) {
		r.LogLevel = &level
	}
}

func WithMaxLogBodySize(size int64) ReqSetting {
	return func(r *OneRequest) {
		r.MaxLogBodySize = size
	}
}

func WithMaxResBodySize(size int64) ReqSetting {
	return func(r *OneRequest) {
		r.MaxResBodySize = size
	}
}

func WithPromKey(key string) ReqSetting {
	return func(r *OneRequest) {
		r.PromKey = key
	}
}

func NewOneRequest(method, ul string, settings ...ReqSetting) *OneRequest {
	req := &OneRequest{
		Method: method,
		URL:    ul,
		Header: map[string]string{
			AcceptEncodingHeader: "gzip",
		},
		Timeout:        defaultTimeout,
		MaxLogBodySize: defaultLogBodySize,
		MaxResBodySize: defaultResBodySize,
	}

	for _, set := range settings {
		set(req)
	}

	return req
}

type OneResponse struct {
	Res    interface{}
	Header map[string]string
	Status int
}

type Client interface {
	Call(ctx context.Context, req *OneRequest, res *OneResponse) error
}
