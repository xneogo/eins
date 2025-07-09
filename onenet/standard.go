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
 @Time    : 2025/7/9 -- 13:50
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2025 亓官竹
 @Description: onenet onenet/standard.go
*/

package onenet

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"github.com/xneogo/eins/onelog"
	"github.com/xneogo/saferun"
	"google.golang.org/protobuf/proto"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type standard struct {
	client   *http.Client
	gzipPool sync.Pool
}

func (s *standard) do(ctx context.Context, req *OneRequest, res *OneResponse) (err error) {
	var (
		start = time.Now()
		hreq  *http.Request
		hres  *http.Response
	)

	if req.LogLevel != nil {
		ctx = onelog.Level(ctx, *req.LogLevel)
	}
	defer func() {
		if err != nil {
			onelog.Ctx(ctx).Err(err).Dur("cost", time.Since(start)).Msg("http request")
		} else {
			onelog.Ctx(ctx).Info().Dur("cost", time.Since(start)).Msg("http request")
		}
	}()

	if len(req.Query) != 0 {
		if strings.Contains(req.URL, "?") {
			req.URL += "&" + req.Query.Encode()
		} else {
			req.URL += "?" + req.Query.Encode()
		}
	}
	ctx = onelog.Str(ctx, "method", req.Method)
	ctx = onelog.Str(ctx, "url", req.URL)

	switch v := req.Body.(type) {
	case io.Reader:
		hreq, err = http.NewRequestWithContext(ctx, req.Method, req.URL, v)
		if err != nil {
			return
		}
	case url.Values:
		s := v.Encode()
		ctx = onelog.StrWithSize(ctx, "reqBody", s, req.MaxLogBodySize)
		hreq, err = http.NewRequestWithContext(ctx, req.Method, req.URL, strings.NewReader(s))
		if err != nil {
			return
		}
	case string:
		ctx = onelog.StrWithSize(ctx, "reqBody", v, req.MaxLogBodySize)
		hreq, err = http.NewRequestWithContext(ctx, req.Method, req.URL, strings.NewReader(v))
		if err != nil {
			return
		}
	case []byte:
		ctx = onelog.BytesWithSize(ctx, "reqBody", v, req.MaxLogBodySize)
		hreq, err = http.NewRequestWithContext(ctx, req.Method, req.URL, bytes.NewReader(v))
		if err != nil {
			return
		}
	case proto.Message:
		var bs []byte
		bs, err = proto.Marshal(v)
		if err != nil {
			return
		}
		ctx = onelog.BytesWithSize(ctx, "reqBody", bs, req.MaxLogBodySize)
		hreq, err = http.NewRequestWithContext(ctx, req.Method, req.URL, bytes.NewReader(bs))
		if err != nil {
			return
		}
	default:
		switch req.ContentType {
		case JsonContentType:
			var bs []byte
			bs, err = json.Marshal(v)
			if err != nil {
				return
			}
			ctx = onelog.BytesWithSize(ctx, "reqBody", bs, req.MaxLogBodySize)
			hreq, err = http.NewRequestWithContext(ctx, req.Method, req.URL, bytes.NewReader(bs))
			if err != nil {
				return
			}
		default:
			hreq, err = http.NewRequestWithContext(ctx, req.Method, req.URL, nil)
			if err != nil {
				return
			}
		}
	}

	if req.ContentType != "" {
		req.Header[ContentTypeHeader] = req.ContentType
	}
	for key, val := range req.Header {
		hreq.Header.Set(key, val)
		ctx = onelog.Str(ctx, "reqHeader."+key, val)
	}

	hres, err = s.client.Do(hreq)
	if err != nil {
		return
	}

	for key := range hres.Header {
		res.Header[key] = hres.Header.Get(key)
		ctx = onelog.Str(ctx, "resHeader."+key, hres.Header.Get(key))
	}

	if v, ok := res.Res.(io.Writer); ok {
		_, err = io.Copy(v, hres.Body)
		return
	}

	var resBody []byte
	if req.MaxResBodySize != 0 {
		reader := io.LimitReader(hres.Body, req.MaxResBodySize)
		resBody, err = io.ReadAll(reader)
		if err != nil {
			return
		}
	} else {
		resBody, err = io.ReadAll(hres.Body)
		if err != nil {
			return
		}
	}

	if hres.Header.Get(ContentEncodingHeader) == "gzip" {
		reader := s.GetGzipReader(bytes.NewBuffer(resBody))
		defer s.PutGzipReader(reader)
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

func (s *standard) Call(ctx context.Context, req *OneRequest, res *OneResponse) (err error) {
	timeout := req.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	return saferun.TimeoutRun(ctx, func() error {
		return s.do(ctx, req, res)
	}, timeout)
}

func (s *standard) GetGzipReader(src io.Reader) (reader *gzip.Reader) {
	if r := s.gzipPool.Get(); r != nil {
		reader = r.(*gzip.Reader)
		_ = reader.Reset(src)
	} else {
		reader, _ = gzip.NewReader(src)
	}
	return reader
}

func (s *standard) PutGzipReader(reader *gzip.Reader) {
	_ = reader.Close()
	s.gzipPool.Put(reader)
}
