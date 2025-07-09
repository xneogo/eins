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
 @Time    : 2025/7/9 -- 14:16
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: onenet onenet/fast.go
*/

package onenet

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"github.com/xneogo/eins/onelog"
	"github.com/xneogo/saferun"
	"google.golang.org/protobuf/proto"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

type fast struct {
	client   *fasthttp.Client
	gzipPool sync.Pool
}

func (f *fast) do(ctx context.Context, req *OneRequest, res *OneResponse) (err error) {
	var (
		start = time.Now()
		hReq  = fasthttp.AcquireRequest()
		hRes  = fasthttp.AcquireResponse()
	)
	defer fasthttp.ReleaseRequest(hReq)
	defer fasthttp.ReleaseResponse(hRes)

	if req.LogLevel != nil {
		ctx = onelog.Level(ctx, *req.LogLevel)
	}
	defer func() {
		if err != nil {
			log.Ctx(ctx).Err(err).Dur("cost", time.Since(start)).Msg("http request")
		} else {
			log.Ctx(ctx).Info().Dur("cost", time.Since(start)).Msg("http request")
		}
	}()

	if len(req.Query) != 0 {
		if strings.Contains(req.URL, "?") {
			req.URL += req.Query.Encode()
		} else {
			req.URL += "?" + req.Query.Encode()
		}
	}
	ctx = onelog.Str(ctx, "method", req.Method)
	ctx = onelog.Str(ctx, "url", req.URL)

	switch v := req.Body.(type) {
	case io.Reader:
		hReq.SetBodyStream(v, -1)
	case url.Values:
		s := v.Encode()
		ctx = onelog.StrWithSize(ctx, "reqBody", s, req.MaxLogBodySize)
		hReq.SetBodyString(s)
	case string:
		ctx = onelog.StrWithSize(ctx, "reqBody", v, req.MaxLogBodySize)
		hReq.SetBodyString(v)
	case []byte:
		ctx = onelog.BytesWithSize(ctx, "reqBody", v, req.MaxLogBodySize)
		hReq.SetBodyRaw(v)
	case proto.Message:
		var bs []byte
		bs, err = proto.Marshal(v)
		if err != nil {
			return
		}
		ctx = onelog.BytesWithSize(ctx, "reqBody", bs, req.MaxLogBodySize)
		hReq.SetBodyRaw(bs)
	default:
		switch req.ContentType {
		case JsonContentType:
			var bs []byte
			bs, err = json.Marshal(v)
			if err != nil {
				return
			}
			ctx = onelog.BytesWithSize(ctx, "reqBody", bs, req.MaxLogBodySize)
			hReq.SetBodyRaw(bs)
		default:
		}
	}

	hReq.Header.SetMethod(req.Method)
	hReq.SetRequestURI(req.URL)
	if req.ContentType != "" {
		req.Header[ContentTypeHeader] = req.ContentType
	}
	for key, val := range req.Header {
		hReq.Header.Set(key, val)
		ctx = onelog.Str(ctx, "reqHeader."+key, val)
	}

	err = f.client.Do(hReq, hRes)
	if err != nil {
		return
	}

	for _, key := range hRes.Header.PeekKeys() {
		sKey := string(key)
		res.Header[sKey] = string(hRes.Header.Peek(sKey))
		ctx = onelog.Str(ctx, "resHeader."+sKey, string(hRes.Header.Peek(sKey)))
	}

	if v, ok := res.Res.(io.Writer); ok {
		_, err = v.Write(hRes.Body())
		return
	}

	var resBody []byte
	if req.MaxResBodySize != 0 {
		if len(hRes.Body()) > int(req.MaxResBodySize) {
			resBody = append(resBody, hRes.Body()[:req.MaxResBodySize]...)
		} else {
			resBody = append(resBody, hRes.Body()...)
		}
	} else {
		resBody = append(resBody, hRes.Body()...)
	}

	if string(hRes.Header.Peek(ContentEncodingHeader)) == "gzip" {
		reader := f.GetGzipReader(bytes.NewBuffer(resBody))
		defer f.PutGzipReader(reader)
		resBody, err = io.ReadAll(reader)
		if err != nil {
			return
		}
	}
	ctx = onelog.BytesWithSize(ctx, "resBody", resBody, req.MaxLogBodySize)

	switch v := res.Res.(type) {
	case *string:
		*v = string(resBody)
	case *[]byte:
		*v = resBody
	case proto.Message:
		if err := proto.Unmarshal(resBody, v); err != nil {
			return err
		}
	default:
		if err := json.Unmarshal(resBody, v); err != nil {
			return err
		}
	}

	return nil
}

func (f *fast) Call(ctx context.Context, req *OneRequest, res *OneResponse) (err error) {
	timeout := req.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	return saferun.TimeoutRun(ctx, func() error {
		return f.do(ctx, req, res)
	}, timeout)
}

func (f *fast) GetGzipReader(src io.Reader) (reader *gzip.Reader) {
	if r := f.gzipPool.Get(); r != nil {
		reader = r.(*gzip.Reader)
		_ = reader.Reset(src)
	} else {
		reader, _ = gzip.NewReader(src)
	}
	return reader
}

func (f *fast) PutGzipReader(reader *gzip.Reader) {
	_ = reader.Close()
	f.gzipPool.Put(reader)
}
